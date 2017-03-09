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
	ring       *Ring
	outbuf     []byte
	outMinSize int
	stack      []State
	rootState  State
	done       bool
	err        *errorf
	format     bool
}

func NewFilter(ring *Ring, outMinSize int, rootState State, format bool) *Filter {

	return &Filter{ring: ring, outMinSize: outMinSize, rootState: rootState, format: format}
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

	for {
		ru := f.ring.Peek()

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f)
		if cerr != nil {
			err = cerr
			return
		}

		if dispatch {
			return
		}

		if ru == '{' {

			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ObjectState{})
			return
		}

		if ru == '[' {
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

		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
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

		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

type KeyNoQuoteState struct {
	baseState
}

func (o *KeyNoQuoteState) Type() TokenType {
	return KeyNoQuote
}

func (r *KeyNoQuoteState) Next(f *Filter) (err *errorf) {

	var escaped bool
	first := true
	for {
		ru := f.ring.Peek()

		if first {
			f.pushOut('"')
			first = false
		}

		if !escaped && ru == ':' {

			f.pushOut('"')
			r.Close()
			return
		}

		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

type ValueNoQuoteState struct {
	baseState
}

func (o *ValueNoQuoteState) Type() TokenType {
	return ValueNoQuote
}

func (r *ValueNoQuoteState) Next(f *Filter) (err *errorf) {

	var escaped bool
	var cval []byte

	renderValue := func() {

		r.Close()

		// check if quotes are not needed
		v := string(cval)
		if json.IsNumber(v) ||
			v == `true` ||
			v == `false` ||
			v == `null` {

			f.pushBytes(cval)
			return
		}

		// quote the value
		f.pushOut('"')
		f.pushBytes(cval)
		f.pushOut('"')
	}

	for {
		ru := f.ring.Peek()

		if !escaped {

			if unicode.IsSpace(ru) || ru == ',' || ru == '}' || ru == ']' {
				renderValue()
				return
			}
		}

		if ru == '\\' {
			escaped = true
		}

		cval = append(cval, byte(ru))

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
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

	for {
		ru := f.ring.Peek()

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f)
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
				break
			}

			err = errorF("error parsing object %v", f.ring.Position())
			return

		case Value, ValueNoQuote, Object, Array:

			if ru == ',' {
				if o.hasComma {
					err = errorF("invalid comma", f.ring.Position())
					return
				}
				o.hasComma = true
				break
			}

			if ru == '}' {

				o.Close()
				f.newLine()

				f.pushOut(ru)
				err = f.ring.Advance()
				return
			}

			if ru == '"' {

				if s.Type() == Object && !s.Open() {
					f.pushOut(',')
				}

				if s.Type() != Object {
					f.pushOut(',')
				}

				f.pushOut(ru)
				err = f.ring.Advance()
				f.newLine()
				f.pushState(&KeyState{})
				return
			}

			if s.Type() == Value || s.Type() == ValueNoQuote {
				f.pushOut(',')
			}
			f.newLine()
			f.pushState(&KeyNoQuoteState{})
			return
		}

	next:
		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

type ArrayState struct {
	baseState
	hasComma bool
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

	for {
		ru := f.ring.Peek()

		// check if comments need to be dispatched
		dispatch, cerr := dispatchComment(f)
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
			goto next
		}

		if ru == ']' {

			r.Close()
			f.newLine()
			f.pushOut(ru)
			err = f.ring.Advance()
			return
		}

		if r.hasComma || (s.Type() == Array && s.Open()) {

			if r.hasComma {
				f.pushOut(',')
			}

			f.newLine()

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

		err = errorF("invalid character (1) %v", f.ring.Position(), string(ru))
		return

	next:
		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

func dispatchComment(f *Filter) (shouldDispatch bool, err *errorf) {

	ru := f.ring.Peek()

	if ru == '/' {
		err = f.ring.Advance()
		if err != nil {
			return
		}

		ru = f.ring.Peek()
		if ru == '/' {
			shouldDispatch = true
			err = f.ring.Advance()
			f.pushState(&CommentState{})
			return
		}

		if ru == '*' {
			shouldDispatch = true
			err = f.ring.Advance()
			f.pushState(&CommentMultiLineState{})
			return
		}

		err = errorF("invalid character (2) %v", f.ring.Position(), string(ru))
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
			err = f.ring.Advance()
			return
		}

		err = f.ring.Advance()

		if err != nil {
			if err.error == io.EOF {
				f.popState()
			}

			return
		}
	}

	return
}

type CommentMultiLineState struct {
	baseState
}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(f *Filter) (err *errorf) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '*' {

			err = f.ring.Advance()
			if err != nil {
				return
			}

			ru = f.ring.Peek()
			if ru == '/' {
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

	return
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

func (f *Filter) newLine() {

	if f.format {

		f.pushOut('\n')

		idx := 0
		for _, s := range f.stack {
			if s.Open() && (s.Type() == Array || s.Type() == Object) {
				idx++
			}
		}

		for i := 0; i < idx; i++ {
			f.pushOut(' ')

			f.pushOut(' ')
		}
	}
}

func (f *Filter) pushOut(r rune) {
	f.outbuf = append(f.outbuf, byte(r))
}

func (f *Filter) pushBytes(b []byte) {
	f.outbuf = append(f.outbuf, b...)
}

func (f *Filter) printStack() {
	for _, s := range f.stack {
		fmt.Printf("(type: %v open %v) ", s.Type(), s.Open())
	}
	fmt.Println(``)
}
