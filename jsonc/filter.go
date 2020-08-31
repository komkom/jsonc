package jsonc

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

	outbuf []byte
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

type RootState struct {
	init bool
}

func (r *RootState) Type() TokenType {
	return Root
}

func (r *RootState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if f.format {
		if ru == '\n' {
			f.pushOut(ru)
		}
	}

	// check if comments need to be dispatched
	dispatch, err := dispatchComment(f, nil)
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
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ObjectState{})
			return err

		case '[':
			r.init = true
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ArrayState{})
			return err

		case '"':
			r.init = true
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&ValueState{})
			return err

		case '`':
			r.init = true
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
	return nil
}

type KeyState struct {
	escaped bool
}

func (o *KeyState) Type() TokenType {
	return Key
}

func (r *KeyState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if !r.escaped && ru == '"' {

		f.pushOut(ru)
		f.popState()
		return f.ring.Advance()
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
	return nil
}

type ValueState struct {
	escaped bool
}

func (v *ValueState) Type() TokenType {
	return Value
}

func (v *ValueState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if !v.escaped && ru == '\n' {
		return fmt.Errorf(`line break in string value`)
	}

	if !v.escaped && ru == '"' {
		f.pushOut(ru)
		f.popState()
		return f.ring.Advance()
	}

	v.escaped = false
	if ru == '\\' {
		v.escaped = true
	}

	f.pushOut(ru)

	err := f.ring.Advance()
	if err != nil {
		return err
	}
	return nil
}

type KeyNoQuoteState struct {
	notFirst bool
}

func (o *KeyNoQuoteState) Type() TokenType {
	return KeyNoQuote
}

func (r *KeyNoQuoteState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if !r.notFirst {
		if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) {
			return Errorf("invalid key", f.ring.Position())
		}
	}

	if !f.format {
		if !r.notFirst {
			f.pushOut('"')
		}
	}
	r.notFirst = true

	if ru == ':' || ru == '/' || unicode.IsSpace(ru) {

		if !f.format {
			f.pushOut('"')
		}
		f.popState()
		return nil
	}

	if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) {
		return Errorf("invalid key", f.ring.Position())
	}

	f.pushOut(ru)

	err := f.ring.Advance()
	if err != nil {
		return err
	}
	return nil
}

type ValueNoQuoteState struct {
	cval []byte
}

func (o *ValueNoQuoteState) Type() TokenType {
	return ValueNoQuote
}

func (r *ValueNoQuoteState) Next(f *Filter) error {

	renderValue := func() error {

		f.popState()
		if len(r.cval) == 0 {
			return Errorf("empty no quote state", f.ring.Position())
		}

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

	ru := f.ring.Peek()

	if unicode.IsSpace(ru) || ru == ',' || ru == '}' || ru == ']' || ru == '/' {

		return renderValue()
	}

	if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) && ru != '.' && ru != '+' {
		return Errorf("invalid identifier", f.ring.Position())
	}

	if ru == '\\' {
		return Errorf("invalid identifier", f.ring.Position())
	}

	r.cval = append(r.cval, byte(ru))

	err := f.ring.Advance()
	if err != nil {
		return err
	}
	return nil
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

type ValueMultilineState struct{}

func (v *ValueMultilineState) Type() TokenType {
	return Value
}

func (v *ValueMultilineState) Next(f *Filter) error {

	if f.format {
		ru := f.ring.Peek()

		if ru == '`' {
			f.pushOut(ru)
			f.popState()
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
		return nil
	}

	for {
		ru := f.ring.Peek()

		if ru == '`' {
			f.pushOut('"')
			f.popState()
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

const (
	ObjIntNextAfterComma ObjInternalState = iota
	ObjIntNext
	ObjInternalKey
	ObjInternalDelimiter
	ObjInternalValue
)

type ObjectState struct {
	internalState ObjInternalState
	lineBreaks    int
}

func (o *ObjectState) Type() TokenType {
	return Object
}

func (o *ObjectState) pop(f *Filter) error {

	if f.format {
		f.pushOutMult(o.lineBreaks, 1, '\n')
		if o.lineBreaks > 0 {
			f.pushSpaces(f.indent() - 1)
		}
		o.lineBreaks = 0
	}
	f.pushOut('}')
	f.popState()
	return f.ring.Advance()
}

func (o *ObjectState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if unicode.IsSpace(ru) || unicode.IsControl(ru) {
		if ru == '\n' {
			o.lineBreaks++
		}
		return f.ring.Advance()
	}

	// check if comments need to be dispatched
	dispatch, err := dispatchComment(f, nil)
	if err != nil {
		return err
	}

	if dispatch {
		return nil
	}

	switch o.internalState {
	case ObjIntNext:

		if ru == '}' {
			return o.pop(f)
		}

		f.pushOut(',')
		o.internalState = ObjIntNextAfterComma
		return nil

	case ObjIntNextAfterComma:

		if ru == '}' {
			return o.pop(f)
		}

		if f.format && o.lineBreaks > 0 {
			f.pushOutMult(o.lineBreaks, 1, '\n')
			if o.lineBreaks > 0 {
				f.pushSpaces(f.indent())
			}
			o.lineBreaks = 0
		}

		o.internalState = ObjInternalKey
		if ru == '"' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(&KeyState{})
			return err
		}

		f.pushState(&KeyNoQuoteState{})
		return nil

	case ObjInternalKey:

		if ru == ':' {
			f.pushOut(ru)
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
			return o.pop(f)
		}

		if ru == ',' {
			o.internalState = ObjIntNext
			return f.ring.Advance()
		}

		if f.format {
			if o.lineBreaks == 0 {
				f.pushOut(' ')
			}
			o.internalState = ObjIntNextAfterComma
		} else {
			o.internalState = ObjIntNext
		}
		return nil
	default:
		return Errorf("invalid internal object state: %v", f.ring.Position(), string(ru))
	}
}

type ArrayInternalState = int

const (
	ArrayIntStart ObjInternalState = iota
	ArrayIntValue
	ArrayIntAfterValue
)

type ArrayState struct {
	init            bool
	internalState   ArrayInternalState
	lineBreaks      int
	spaceOrControls int
}

func (ArrayState) Type() TokenType {
	return Array
}

func (a *ArrayState) Next(f *Filter) error {

	ru := f.ring.Peek()

	if unicode.IsSpace(ru) || unicode.IsControl(ru) {
		if ru == '\n' {
			a.lineBreaks++
		}

		a.spaceOrControls++
		return f.ring.Advance()
	}

	// check if comments need to be dispatched
	dispatch, err := dispatchComment(f, nil)
	if err != nil {
		return err
	}

	if dispatch {
		return nil
	}

	switch a.internalState {
	case ArrayIntAfterValue:

		spaceOrControls := a.spaceOrControls
		a.spaceOrControls = 0

		switch ru {
		case ']':
			a.internalState = ArrayIntValue
			return nil
		case ',':
			a.internalState = ArrayIntValue
			if f.format {
				f.pushOut(',')
			}
			return f.ring.Advance()
		}

		if spaceOrControls > 0 {
			if f.format && a.lineBreaks == 0 {
				f.pushOut(' ')
			}
			a.internalState = ArrayIntValue
			return nil
		}

		return Errorf("invalid character after value", f.ring.Position())

	case ArrayIntStart, ArrayIntValue:

		if ru == ']' {
			if f.format {
				f.pushOutMult(a.lineBreaks, 1, '\n')
				f.pushSpaces(f.indent() - 1)
			}
			f.popState()
			f.pushOut(ru)
			return f.ring.Advance()
		}

		if a.internalState != ArrayIntStart && !f.format {
			f.pushOut(',')
		}

		if f.format {
			f.pushOutMult(a.lineBreaks, 1, '\n')
			if a.lineBreaks > 0 {
				f.pushSpaces(f.indent())
			}
			a.lineBreaks = 0
		}

		a.internalState = ArrayIntAfterValue
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
	default:
		return Errorf("invalid internal array state: %v", f.ring.Position(), string(ru))
	}
}

func dispatchComment(f *Filter, postHook func() error) (shouldDispatch bool, err error) {

	ru := f.ring.Peek()

	if ru == '/' {
		err = f.ring.Advance()
		if err != nil {
			return
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
				f.pushBytes([]byte(" //"))
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
				f.pushBytes([]byte(" /*"))
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

type CommentState struct{}

func (c *CommentState) Type() TokenType {
	return Comment
}

func (c *CommentState) Next(f *Filter) error {

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
			f.popState()
		}
		return err
	}
	return nil
}

type CommentMultiLineState struct {
	escaped bool
}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(f *Filter) error {

	ru := f.ring.Peek()
	if !c.escaped && ru == '*' {

		err := f.ring.Advance()
		if err != nil {
			return err
		}

		ru = f.ring.Peek()
		if ru == '/' {

			if f.format {
				f.pushOut('*')
				f.pushOut('/')
			}

			f.popState()
			return f.ring.Advance()
		}

		f.ring.Pop()
	}

	c.escaped = false
	if ru == '\\' {
		c.escaped = true
	}

	if f.format {
		f.pushOut(ru)
	}

	err := f.ring.Advance()
	if err != nil {
		return err
	}
	return nil
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

	if errors.Is(err, io.EOF) && f.peekState().Type() == Root {
		f.done = true
	}

	return n, err
}

func (f *Filter) fill() error {

	state := f.peekState()
	for f.outMinSize > len(f.outbuf) {
		err := state.Next(f)
		if err != nil {
			return err
		}
		state = f.peekState()
	}

	return nil
}

func (f *Filter) peekState() State {
	if len(f.stack) > 0 {
		return f.stack[len(f.stack)-1]
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

func (f *Filter) indent() int {

	var indent int
	for _, s := range f.stack {
		if s.Type() == Object || s.Type() == Array {
			indent++
		}
	}
	return indent
}

func (f *Filter) pushSpaces(c int) {
	for i := 0; i < c; i++ {
		f.pushOut(' ')
	}
}

func (f *Filter) pushOut(r rune) {
	f.outbuf = append(f.outbuf, byte(r))
}

func (f *Filter) pushBytes(b []byte) {
	f.outbuf = append(f.outbuf, b...)
}

func (f *Filter) pushOutMult(t int, max int, r rune) {

	if t > max {
		t = max
	}

	for i := 0; i < t; i++ {
		f.pushOut(r)
	}
}
