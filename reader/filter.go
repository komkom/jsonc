package reader

import (
	"errors"
	"fmt"
	"io"
	"unicode"
)

type TokenType int

type State interface {
	Type() TokenType
	Next(f *Filter) error
	Close()
	Open() bool
}

const (
	Root             TokenType = iota // 0
	Object                            // 1
	Key                               // 2
	KeyNoQuote                        // 3
	Value                             // 4
	ValueNoQuote                      // 5
	ValueMultiline                    // 6
	Array                             // 7
	Comment                           // 8
	CommentMultiLine                  // 9
)

type Filter struct {
	ring         *Ring
	outMinSize   int
	stack        []State
	rootState    *RootState
	done         bool
	err          error
	format       bool
	newlineCount int
	space        string

	outbuf            []byte
	lastWasWhitespace bool
	embed             bool
}

func NewFilter(ring *Ring, outMinSize int, format bool, space string) *Filter {

	return &Filter{ring: ring, outMinSize: outMinSize, rootState: &RootState{}, format: format, space: space}
}

func (f *Filter) Clear() {
	f.rootState.init = false
	f.outbuf = nil
	f.stack = nil
	f.done = false
	f.err = nil
}

func (f *Filter) Done() bool {
	return f.done
}

func (f *Filter) Err() error {
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
	init bool
	baseState
}

func (r *RootState) Type() TokenType {
	return Root
}

func (r *RootState) Next(f *Filter) error {

	var nlcount int
	for {
		ru := f.ring.Peek()

		if ru == '\n' {
			nlcount++
		}

		// check if comments need to be dispatched
		dispatch, err := dispatchComment(f, nlcount, nil)
		if err != nil {
			return err
		}

		if dispatch {
			return nil
		}

		if !r.init {
			switch ru {
			case '{':
				r.init = true
				f.newLine(nlcount)
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ObjectState{})
				return err

			case '[':
				r.init = true
				f.newLine(nlcount)
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ArrayState{})
				return err

			case '"':
				r.init = true
				f.newLine(nlcount)
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ValueState{})
				return err

			case '`':
				r.init = true
				f.newLine(nlcount)
				if f.format {
					f.pushOut(ru)
				} else {
					f.pushOut('"')
				}
				err = f.ring.Advance()
				f.pushState(&ValueMultilineState{})
				return err
			}
		}

		if !unicode.IsSpace(ru) {
			return Errorf("invalid first character: %v", -1, string(ru))
		}

		err = f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

type KeyState struct {
	baseState
}

func (o *KeyState) Type() TokenType {
	return Key
}

func (r *KeyState) Next(f *Filter) error {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {

			r.Close()

			f.pushOut(ru)
			return f.ring.Advance()
		}

		escaped = false
		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err := f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

type ValueState struct {
	baseState
}

func (v *ValueState) Type() TokenType {
	return Value
}

func (v *ValueState) Next(f *Filter) error {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '\n' {
			return fmt.Errorf(`line break in string value`)
		}

		if !escaped && ru == '"' {
			v.Close()

			f.pushOut(ru)
			return f.ring.Advance()
		}

		escaped = false
		if ru == '\\' {
			escaped = true
		}

		f.pushOut(ru)

		err := f.ring.Advance()
		if err != nil {
			return err
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

func (r *KeyNoQuoteState) Next(f *Filter) error {

	for {
		ru := f.ring.Peek()

		if !r.notFirst {
			if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) {
				return Errorf("invalid key", f.ring.Position())
			}
		}

		if !f.format {
			if !r.notFirst {
				f.pushOut('"')
				r.notFirst = true
			}
		}

		if !r.escaped {

			// check if comments need to be dispatched
			dispatch, err := dispatchComment(f, 0, func() error {
				if !f.format {
					f.pushOut('"')
				}
				r.Close()
				return nil
			})
			if err != nil {
				return err
			}

			if dispatch {
				return nil
			}

			if ru == ':' || unicode.IsSpace(ru) {

				if !f.format {
					f.pushOut('"')
				}
				r.Close()
				return nil
			}
		}

		r.escaped = false
		if ru == '\\' {
			r.escaped = true
		}

		f.pushOut(ru)

		err := f.ring.Advance()
		if err != nil {
			return err
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

func (r *ValueNoQuoteState) Next(f *Filter) error {

	renderValue := func() error {

		if len(r.cval) == 0 {
			return Errorf("empty no quote state", f.ring.Position())
		}

		r.Close()

		// check if quotes are not needed
		v := string(r.cval)
		if IsNumber(v) ||
			v == `true` ||
			v == `false` ||
			v == `null` {

			f.pushBytes(r.cval)
			return nil
		}

		if !unicode.IsLetter(([]rune(v))[0]) {
			return Errorf("invalid identifier", f.ring.Position())
		}

		if f.format {
			f.pushBytes(r.cval)
			return nil
		}

		// quote the value
		f.pushOut('"')
		f.pushBytes(r.cval)
		f.pushOut('"')

		return nil
	}

	for {
		ru := f.ring.Peek()

		if !r.escaped {

			// check if comments need to be dispatched
			dispatch, err := dispatchComment(f, 0, func() error {
				return renderValue()
			})
			if err != nil {
				return err
			}

			if dispatch {
				return nil
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

		err := f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

const (
	quotationMark  = '"'
	solidus        = '\u002F'
	formFeed       = '\u000C'
	lineFeed       = '\n'
	carriageReturn = '\r'
	tab            = '\u0009'
)

var (
	multilineEscapes = []struct {
		code    rune
		replace rune
	}{
		{code: quotationMark, replace: '"'},
		{code: solidus, replace: solidus},
		{code: formFeed, replace: 'f'},
		{code: lineFeed, replace: 'n'},
		{code: carriageReturn, replace: 'r'},
		{code: tab, replace: 't'},
	}
)

func needsReplacement(r rune) (replace rune, ok bool) {

	for _, o := range multilineEscapes {
		if o.code == r {
			return o.replace, true
		}
	}
	return replace, false
}

type ValueMultilineState struct {
	baseState
}

func (v *ValueMultilineState) Type() TokenType {
	return Value
}

func (v *ValueMultilineState) Next(f *Filter) error {

	if f.format {
		for {
			ru := f.ring.Peek()

			if ru == '`' {
				v.Close()
				f.pushOut(ru)
				return f.ring.Advance()
			}

			if ru == '\\' {
				return fmt.Errorf("character \\ found in multiline string")

			}

			f.pushOut(ru)
			err := f.ring.Advance()
			if err != nil {
				return err
			}
		}
	}

	for {
		ru := f.ring.Peek()

		if ru == '`' {
			v.Close()
			f.pushOut('"')
			return f.ring.Advance()
		}

		if ru == '\\' {
			return fmt.Errorf("character \\ found in multiline string")
		}

		if rep, ok := needsReplacement(ru); ok {
			f.pushOut('\\')
			f.pushOut(rep)
			err := f.ring.Advance()
			if err != nil {
				return err
			}

			continue
		}

		if unicode.IsSpace(ru) {
			f.pushOut(' ')
			err := f.ring.Advance()
			if err != nil {
				return err
			}
			continue
		}

		f.pushOut(ru)
		err := f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

type ObjInternalState = int

var (
	ObjIntNext ObjInternalState = 0

	ObjInternalKey       ObjInternalState = 1
	ObjInternalDelimiter ObjInternalState = 2
	ObjInternalValue     ObjInternalState = 3
)

type ObjectState struct {
	baseState
	internalState ObjInternalState
}

func (o *ObjectState) Type() TokenType {
	return Object
}

func (o *ObjectState) Next(f *Filter) error {

	var nlcount int
	for {
		ru := f.ring.Peek()
		if ru == '\n' {
			nlcount++
		}

		// check if comments need to be dispatched
		dispatch, err := dispatchComment(f, nlcount, nil)
		if err != nil {
			return err
		}

		if dispatch {
			return nil
		}

		if unicode.IsSpace(ru) {
			goto next
		}

		switch o.internalState {
		case ObjIntNext:

			o.internalState = ObjInternalKey
			if ru == '"' {

				f.newLine(nlcount)
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&KeyState{})
				return err
			}

			f.newLine(nlcount)
			f.pushState(&KeyNoQuoteState{})
			return nil

		case ObjInternalKey:

			if ru == ':' {
				f.pushOut(ru)
				if f.format {
					f.pushSpace()
				}
				o.internalState = ObjInternalDelimiter
				return f.ring.Advance()
			}

			return Errorf("error parsing object rune: %v", f.ring.Position(), string(ru))

		case ObjInternalDelimiter:

			o.internalState = ObjInternalValue
			switch ru {
			case '[':
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ArrayState{})
				return err

			case '{':
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ObjectState{})
				return err

			case '"':
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&ValueState{})
				return err

			case '`':
				if f.format {
					f.pushOut(ru)
				} else {
					f.pushOut('"')
				}
				err = f.ring.Advance()
				f.pushState(&ValueMultilineState{})
				return err
			default:
				f.pushState(&ValueNoQuoteState{})
				return nil
			}

		case ObjInternalValue:

			if ru == '}' {
				o.Close()
				f.pushOut(ru)
				return f.ring.Advance()
			}

			if ru == ',' {
				f.pushOut(ru)
				o.internalState = ObjIntNext
				return f.ring.Advance()
			}

			if !f.format {
				f.pushOut(',')
			}
			o.internalState = ObjIntNext
			return nil
		}

	next:
		err = f.ring.Advance()
		if err != nil {
			return err
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

func (r *ArrayState) Next(f *Filter) error {

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
		dispatch, err := dispatchComment(f, nlcount, nil)
		if err != nil {
			return err
		}

		if dispatch {
			dontResetComma = true
			return nil
		}

		if unicode.IsSpace(ru) {
			goto next
		}

		if ru == ',' {
			if r.hasComma {
				return Errorf("invalid comma", f.ring.Position())
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
			return f.ring.Advance()
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
			return err
		}

		if ru == '{' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ObjectState{})
			return err
		}

		if ru == '"' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ValueState{})
			return err
		}

		if ru == '`' {
			if f.format {
				f.pushOut(ru)
			} else {
				f.pushOut('"')
			}
			err = f.ring.Advance()
			f.pushState(&ValueMultilineState{})
			return err

		}

		if !unicode.IsSpace(ru) {
			f.pushState(&ValueNoQuoteState{})
			return nil
		}

		return Errorf("invalid character (1) %v", f.ring.Position(), string(ru))

	next:
		err = f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

func dispatchComment(f *Filter, nlcount int, postHook func() error) (shouldDispatch bool, err error) {

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

func (c *CommentState) Next(f *Filter) error {

	for {
		ru := f.ring.Peek()

		if ru == '\n' {
			f.popState()
			return nil
		}

		if f.format {
			f.pushOut(ru)
		}

		err := f.ring.Advance()
		if err != nil {
			if errors.Is(err, io.EOF) {
				f.embed = true
				f.popState()
			}
			return nil
		}
	}
}

type CommentMultiLineState struct {
	baseState
}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(f *Filter) error {

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

			err := f.ring.Advance()
			if err != nil {
				return err
			}

			ru = f.ring.Peek()
			if ru == '/' {

				if f.format {
					f.pushOut('/')
				}

				f.embed = true

				f.popState()
				return f.ring.Advance()
			}

			f.ring.Pop()
		}

		if ru == '\\' {
			escaped = true
		}

		err := f.ring.Advance()
		if err != nil {
			return err
		}
	}
}

func (f *Filter) Read(p []byte) (n int, err error) {

	if f.err != nil {
		return 0, f.err
	}

	n = f.outMinSize
	if n > len(p) {
		n = len(p)
	}

	for n > len(f.outbuf) {

		err = f.fill()
		if err != nil {
			break
		}
	}
	f.err = err

	if len(f.outbuf) < n {
		n = len(f.outbuf)
	}

	for i := 0; i < n; i++ {
		p[i] = f.outbuf[i]
	}

	f.outbuf = f.outbuf[n:]

	s := f.peekFirstOpenState()
	if s.Type() == Root && (s.(*RootState)).init {
		s.Close()
		f.done = true
	}

	return n, err
}

func (f *Filter) fill() error {

	state := f.peekFirstOpenState()
	for f.outMinSize > len(f.outbuf) {
		err := state.Next(f)
		if err != nil {
			return err
		}

		state = f.peekFirstOpenState()
	}
	return nil
}

func (f *Filter) peekState() State {
	var state State = f.rootState
	if len(f.stack) > 0 {
		state = f.stack[len(f.stack)-1]
	}
	return state
}

func (f *Filter) peekFirstOpenState() State {

	for i := len(f.stack) - 1; i >= 0; i-- {
		s := f.stack[i]
		if s.Open() {
			return s
		}
	}
	return f.rootState
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
