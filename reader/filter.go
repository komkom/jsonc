package reader

import (
	"fmt"
	"io"
	"unicode"

	"github.com/komkom/jsonc/json"
)

type TokenType int

type State interface {
	Type() TokenType
	Next(f *Filter) (err *errorf)
	Close()
	Open() bool
}

const (
	Root TokenType = iota
	Object
	Key
	KeyNoQuote
	Value
	ValueNoQuote
	Array
	Comment
	CommentMultiLine
)

type Filter struct {
	ring         *Ring
	outMinSize   int
	stack        []State
	rootState    State
	done         bool
	err          *errorf
	format       bool
	newlineCount int
	space        string

	outbuf            []byte
	lastWasWhitespace bool
	embed             bool
}

func NewFilter(ring *Ring, outMinSize int, rootState State, format bool, space string) *Filter {

	return &Filter{ring: ring, outMinSize: outMinSize, rootState: rootState, format: format, space: space}
}

func (f *Filter) Clear() {
	f.outbuf = nil
	f.stack = nil
	f.done = false
	f.err = nil
}

func (f *Filter) Done() bool {
	return f.done
}

func (f *Filter) Error() *errorf {
	return f.err
}

type baseState struct {
	closed bool
}

func (b *baseState) Close() {
	b.closed = true
}

func (b *baseState) Open() bool {
	return !b.closed
}

type RootState struct {
	baseState
}

func (r *RootState) Type() TokenType {
	return Root
}

func (r *RootState) Next(f *Filter) (err *errorf) {

	var nlcount int
	for {
		ru := f.ring.Peek()
		if ru == '\n' {
			nlcount++
		}

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f, nlcount, nil)
		if cerr != nil {
			err = cerr
			return
		}

		if dispatch {
			return
		}

		if ru == '{' {

			f.newLine(nlcount)
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ObjectState{})
			return
		}

		if ru == '[' {

			f.newLine(nlcount)
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ArrayState{})
			return
		}

		if !unicode.IsSpace(ru) {
			err = errorF("invalid first character: %v", -1, string(ru))
			return
		}

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type KeyState struct {
	baseState
}

func (o *KeyState) Type() TokenType {
	return Key
}

func (r *KeyState) Next(f *Filter) (err *errorf) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {

			r.Close()

			f.pushOut(ru)
			err = f.ring.Advance()
			return
		}

		escaped = false
		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type ValueState struct {
	baseState
}

func (v *ValueState) Type() TokenType {
	return Value
}

func (v *ValueState) Next(f *Filter) (err *errorf) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {
			v.Close()

			f.pushOut(ru)
			err = f.ring.Advance()
			return
		}

		escaped = false
		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type KeyNoQuoteState struct {
	baseState
	escaped  bool
	notFirst bool
}

func (o *KeyNoQuoteState) Type() TokenType {
	return KeyNoQuote
}

func (r *KeyNoQuoteState) Next(f *Filter) (err *errorf) {

	for {
		ru := f.ring.Peek()

		if !f.format {
			if !r.notFirst {
				f.pushOut('"')
				r.notFirst = true
			}
		}

		if !r.escaped {

			// check if comments need to be dispatched
			dispatch, cerr := dispatchComment(f, 0, func() *errorf {
				if !f.format {
					f.pushOut('"')
				}
				r.Close()
				return nil
			})
			if cerr != nil {
				err = cerr
				return
			}

			if dispatch {
				return
			}

			if ru == ':' || unicode.IsSpace(ru) {

				if !f.format {
					f.pushOut('"')
				}
				r.Close()
				return
			}
		}

		r.escaped = false
		if ru == '\\' {
			r.escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type ValueNoQuoteState struct {
	baseState
	escaped bool
	cval    []byte
}

func (o *ValueNoQuoteState) Type() TokenType {
	return ValueNoQuote
}

func (r *ValueNoQuoteState) Next(f *Filter) (err *errorf) {

	renderValue := func() (err *errorf) {

		if len(r.cval) == 0 {
			return errorF("empty no quote state", f.ring.Position())
		}

		r.Close()

		// check if quotes are not needed
		v := string(r.cval)
		if json.IsNumber(v) ||
			v == `true` ||
			v == `false` ||
			v == `null` {

			f.pushBytes(r.cval)
			return
		}

		if f.format {
			f.pushBytes(r.cval)
			return
		}

		// quote the value
		f.pushOut('"')
		f.pushBytes(r.cval)
		f.pushOut('"')

		return
	}

	for {
		ru := f.ring.Peek()

		if !r.escaped {

			// check if comments need to be dispatched
			dispatch, cerr := dispatchComment(f, 0, func() *errorf {
				return renderValue()
			})
			if cerr != nil {
				err = cerr
				return
			}

			if dispatch {
				return
			}

			if unicode.IsSpace(ru) || ru == ',' || ru == '}' || ru == ']' {

				return renderValue()
			}
		}

		r.escaped = false
		if ru == '\\' {
			r.escaped = true
		}

		r.cval = append(r.cval, byte(ru))

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type ObjectState struct {
	baseState
	hasComma        bool
	hasKeyDelimiter bool
}

func (o *ObjectState) Type() TokenType {
	return Object
}

func (o *ObjectState) Next(f *Filter) (err *errorf) {

	s := f.peekState()

	var dontResetState bool

	defer func() {
		if !dontResetState {
			o.hasComma = false
			o.hasKeyDelimiter = false
		}
	}()

	var nlcount int

	for {
		ru := f.ring.Peek()
		if ru == '\n' {
			nlcount++
		}

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f, nlcount, nil)
		if cerr != nil {
			err = cerr
			return
		}

		if dispatch {
			dontResetState = true
			return
		}

		if unicode.IsSpace(ru) {
			goto next
		}

		switch s.Type() {
		case Key, KeyNoQuote:

			f.newLine(nlcount)
			nlcount = 0

			if o.hasKeyDelimiter {

				o.hasKeyDelimiter = false

				if ru == ',' {
					err = errorF("invalid comma", f.ring.Position())
					return
				}

				if ru == '[' {
					f.pushOut(ru)
					err = f.ring.Advance()
					f.pushState(&ArrayState{})
					return
				}

				if ru == '{' {
					f.pushOut(ru)
					err = f.ring.Advance()
					f.pushState(&ObjectState{})
					return
				}

				if ru == '"' {
					f.pushOut(ru)
					err = f.ring.Advance()
					f.pushState(&ValueState{})
					return
				}

				f.pushState(&ValueNoQuoteState{})
				return
			}

			if ru == ':' {

				if o.hasKeyDelimiter {
					err = errorF("invalid character", f.ring.Position())
					return
				}

				o.hasKeyDelimiter = true
				f.pushOut(ru)
				if f.format {
					//f.pushOut(' ')
					f.pushSpace()
				}
				break
			}

			err = errorF("error parsing object rune: %v", f.ring.Position(), string(ru))
			return

		case Value, ValueNoQuote, Object, Array:

			if ru == ',' {
				if o.hasComma {
					err = errorF("invalid comma", f.ring.Position())
					return
				}
				o.hasComma = true

				if f.format {

					f.pushOut(',')
					//f.pushOut(' ')
				}

				break
			}

			f.embed = true

			if ru == '}' {

				o.Close()
				f.newLine(nlcount)

				f.pushOut(ru)
				err = f.ring.Advance()
				return
			}

			if !f.format && ((s.Type() == Object && !s.Open()) || s.Type() != Object) {
				f.pushOut(',')

			}

			/*
				if s.Type() != Object {
					f.pushOut(',')
					if f.format {
						f.pushOut(' ')
					}
				}
			*/

			if ru == '"' {

				f.newLine(nlcount)
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&KeyState{})
				return
			}

			f.newLine(nlcount)
			f.pushState(&KeyNoQuoteState{})
			return
		}

	next:
		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

type ArrayState struct {
	baseState
	hasComma     bool
	didLinebreak bool
}

func (o *ArrayState) Type() TokenType {
	return Array
}

func (r *ArrayState) Next(f *Filter) (err *errorf) {

	s := f.peekState()

	var dontResetComma bool

	defer func() {
		if !dontResetComma {
			r.hasComma = false
		}
	}()

	var nlcount int
	for {
		ru := f.ring.Peek()

		if ru == '\n' {
			r.didLinebreak = true
			nlcount++
		}

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f, nlcount, nil)
		if cerr != nil {
			err = cerr
			return
		}

		if dispatch {
			dontResetComma = true
			return
		}

		if unicode.IsSpace(ru) {
			goto next
		}

		if ru == ',' {
			if r.hasComma {
				err = errorF("invalid comma", f.ring.Position())
				return
			}
			r.hasComma = true

			if f.format {
				f.pushOut(',')
				f.embed = true
			}

			goto next
		}

		if ru == ']' {

			r.Close()
			if r.didLinebreak {
				f.newLine(1)
			}
			f.pushOut(ru)
			err = f.ring.Advance()
			return
		}

		if s.Type() != Array || !s.Open() {
			if !f.format {
				f.pushOut(',')
			} else if !r.hasComma {
				f.embed = true
			}
		}

		if nlcount > 0 {
			f.newLine(nlcount)
		}

		if ru == '[' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ArrayState{})
			return
		}

		if ru == '{' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ObjectState{})
			return
		}

		if ru == '"' {

			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ValueState{})
			return
		}

		if !unicode.IsSpace(ru) {
			f.pushState(&ValueNoQuoteState{})
			return
		}

		err = errorF("invalid character (1) %v", f.ring.Position(), string(ru))
		return

	next:
		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

func dispatchComment(f *Filter, nlcount int, postHook func() *errorf) (shouldDispatch bool, err *errorf) {

	ru := f.ring.Peek()

	if ru == '/' {
		err = f.ring.Advance()
		if err != nil {
			return
		}

		if nlcount > 0 {
			f.newLine(nlcount)
		}

		ru = f.ring.Peek()
		if ru == '/' {

			if postHook != nil {
				err = postHook()
				if err != nil {
					return
				}
			}

			shouldDispatch = true
			err = f.ring.Advance()
			if f.format {
				if !f.lastWasWhitespace {
					//f.pushOut(' ')
					f.pushSpace()
				}
				f.pushBytes([]byte("//"))
			}
			f.pushState(&CommentState{})
			return
		}

		if ru == '*' {
			if postHook != nil {
				postHook()
			}

			shouldDispatch = true
			if f.format {
				if !f.lastWasWhitespace {
					//f.pushOut(' ')
					f.pushSpace()
				}

				f.pushBytes([]byte("/*"))
			}
			err = f.ring.Advance()
			f.pushState(&CommentMultiLineState{})
			return
		}
		err = f.ring.Pop()

		return
	}

	return
}

type CommentState struct {
	baseState
}

func (c *CommentState) Type() TokenType {
	return Comment
}

func (c *CommentState) Next(f *Filter) (err *errorf) {

	for {
		ru := f.ring.Peek()

		if ru == '\n' {
			f.popState()
			//err = f.ring.Advance()
			return
		}

		if f.format {
			f.pushOut(ru)
		}

		err = f.ring.Advance()

		if err != nil {
			if err.error == io.EOF {
				f.embed = true
				f.popState()
			}

			return
		}
	}
}

type CommentMultiLineState struct {
	baseState
}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(f *Filter) (err *errorf) {

	var escaped bool
	var hasNewline bool
	for {
		ru := f.ring.Peek()
		if f.format {

			if !hasNewline || ru == '\n' || !unicode.IsSpace(ru) {

				hasNewline = false

				if ru == '\n' {
					f.newLine(1)
					hasNewline = true
				} else {
					f.pushOut(ru)
				}
			}
		}

		if !escaped && ru == '*' {

			err = f.ring.Advance()
			if err != nil {
				return
			}

			ru = f.ring.Peek()
			if ru == '/' {

				if f.format {
					f.pushOut('/')
				}

				f.embed = true

				f.popState()
				err = f.ring.Advance()
				return
			}

			f.ring.Pop()
		}

		if ru == '\\' {
			escaped = true
		}

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}
}

func (f *Filter) Read(p []byte) (n int, err error) {

	if f.err != nil {
		err = f.err
		return
	}

	var cerr *errorf

	n = f.outMinSize
	if n > len(p) {
		n = len(p)
	}

	for n > len(f.outbuf) {

		cerr = f.fill()
		if cerr != nil {
			err = cerr
			break
		}
	}

	if len(f.outbuf) < n {
		n = len(f.outbuf)
	}

	for i := 0; i < n; i++ {
		p[i] = f.outbuf[i]
	}

	f.outbuf = f.outbuf[n:]

	s, e := f.peekFirstOpenState()
	if e == nil && s.Type() == Root {
		f.done = true
	}

	f.err = cerr

	return
}

func (f *Filter) fill() *errorf {

	state, err := f.peekFirstOpenState()
	if err != nil {
		return err
	}

	for f.outMinSize > len(f.outbuf) {

		//f.printStack()
		//fmt.Printf("__state %v\n", state.Type())

		err := state.Next(f)

		if err != nil {
			return err
		}

		state, err = f.peekFirstOpenState()
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Filter) peekState() State {
	state := f.rootState
	if len(f.stack) > 0 {
		state = f.stack[len(f.stack)-1]
	}
	return state
}

func (f *Filter) peekFirstOpenState() (s State, err *errorf) {

	if len(f.stack) == 0 {
		s = f.rootState
		return
	}

	for i := len(f.stack) - 1; i >= 0; i-- {
		s = f.stack[i]
		if s.Open() {
			return
		}
	}

	s = f.rootState
	return
}

func (f *Filter) pushState(s State) {

	f.stack = append(f.stack, s)
}

func (f *Filter) popState() {
	if len(f.stack) == 0 {
		return
	}
	f.stack = f.stack[:len(f.stack)-1]
}

func (f *Filter) newLine(newLineCount int) {

	if f.format && newLineCount > 0 {

		if newLineCount > 2 {
			newLineCount = 2
		}

		for i := 0; i < newLineCount; i++ {
			f.pushOut('\n')
		}

		idx := 0
		for _, s := range f.stack {
			if s.Open() && (s.Type() == Array || s.Type() == Object) {
				idx++
			}
		}

		for i := 0; i < idx; i++ {
			//f.pushOut(' ')
			//f.pushOut(' ')

			f.pushSpace()
			f.pushSpace()
		}
	}
}

func (f *Filter) pushSpace() {

	f.embed = false
	f.outbuf = append(f.outbuf, []byte(f.space)...)
	f.lastWasWhitespace = true
}

func (f *Filter) pushOut(r rune) {

	f.embedIfNeeded(r)

	f.outbuf = append(f.outbuf, byte(r))
	f.lastWasWhitespace = unicode.IsSpace(r)
}

func (f *Filter) pushBytes(b []byte) {

	// always embed
	f.embedIfNeeded('x')

	f.outbuf = append(f.outbuf, b...)
	f.lastWasWhitespace = false
}

func (f *Filter) embedIfNeeded(r rune) {
	if f.format {
		if f.embed && !unicode.IsSpace(r) {
			f.outbuf = append(f.outbuf, []byte(f.space)...)
		}
		f.embed = false
	}
}

func (f *Filter) printStack() {
	for _, s := range f.stack {
		fmt.Printf("(type: %v open %v) ", s.Type(), s.Open())
	}
	fmt.Println(``)
}
