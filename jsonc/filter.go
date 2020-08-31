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
	Next(r rune, f *Filter) error
}

var ErrDontAdvance = fmt.Errorf(`dont-advance`)

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

func (r *RootState) Next(ru rune, f *Filter) error {

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
		return ErrDontAdvance
	}

	if !r.init {
		switch ru {
		case '{':
			r.init = true
			f.pushOut(ru)
			f.pushState(&ObjectState{})
			return nil

		case '[':
			r.init = true
			f.pushOut(ru)
			f.pushState(&ArrayState{})
			return nil

		case '"':
			r.init = true
			f.pushOut(ru)
			f.pushState(&ValueState{})
			return nil

		case '`':
			r.init = true
			if f.format {
				f.pushOut(ru)
			} else {
				f.pushOut('"')
			}
			f.pushState(&ValueMultilineState{})
			return nil
		}
	}

	if !unicode.IsSpace(ru) {
		return Errorf("invalid first character: %v", -1, string(ru))
	}
	return nil
}

type KeyState struct {
	escaped bool
}

func (o *KeyState) Type() TokenType {
	return Key
}

func (k *KeyState) Next(ru rune, f *Filter) error {

	if !k.escaped && ru == '"' {

		f.pushOut(ru)
		f.popState()
		return nil
	}

	k.escaped = false
	if ru == '\\' {
		k.escaped = true
	}

	f.pushOut(ru)
	return nil
}

type ValueState struct {
	escaped bool
}

func (v *ValueState) Type() TokenType {
	return Value
}

func (v *ValueState) Next(ru rune, f *Filter) error {

	if !v.escaped && ru == '\n' {
		return fmt.Errorf(`line break in string value`)
	}

	if !v.escaped && ru == '"' {
		f.pushOut(ru)
		f.popState()
		return nil
	}

	v.escaped = false
	if ru == '\\' {
		v.escaped = true
	}

	f.pushOut(ru)
	return nil
}

type KeyNoQuoteState struct {
	notFirst bool
}

func (o *KeyNoQuoteState) Type() TokenType {
	return KeyNoQuote
}

func (k *KeyNoQuoteState) Next(ru rune, f *Filter) error {

	if !k.notFirst {
		if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) {
			return Errorf("invalid key", f.ring.Position())
		}
	}

	if !f.format {
		if !k.notFirst {
			f.pushOut('"')
		}
	}
	k.notFirst = true

	if ru == ':' || ru == '/' || unicode.IsSpace(ru) {

		if !f.format {
			f.pushOut('"')
		}
		f.popState()
		return ErrDontAdvance
	}

	if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) {
		return Errorf("invalid key", f.ring.Position())
	}

	f.pushOut(ru)
	return nil
}

type ValueNoQuoteState struct {
	cval []byte
}

func (o *ValueNoQuoteState) Type() TokenType {
	return ValueNoQuote
}

func (v *ValueNoQuoteState) Next(ru rune, f *Filter) error {

	renderValue := func() error {

		f.popState()
		if len(v.cval) == 0 {
			return Errorf("empty no quote state", f.ring.Position())
		}

		// check if quotes are not needed
		s := string(v.cval)
		if IsNumber(s) ||
			s == `true` ||
			s == `false` ||
			s == `null` {

			f.pushBytes(v.cval)
			return ErrDontAdvance
		}

		if !unicode.IsLetter(([]rune(s))[0]) {
			return Errorf("invalid identifier", f.ring.Position())
		}

		if f.format {
			f.pushBytes(v.cval)
			return ErrDontAdvance
		}

		// quote the value
		f.pushOut('"')
		f.pushBytes(v.cval)
		f.pushOut('"')

		return ErrDontAdvance
	}

	if unicode.IsSpace(ru) || ru == ',' || ru == '}' || ru == ']' || ru == '/' {
		return renderValue()
	}

	if !unicode.IsLetter(ru) && !unicode.IsDigit(ru) && ru != '.' && ru != '+' {
		return Errorf("invalid identifier", f.ring.Position())
	}

	if ru == '\\' {
		return Errorf("invalid identifier", f.ring.Position())
	}

	v.cval = append(v.cval, byte(ru))
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

func (*ValueMultilineState) Next(ru rune, f *Filter) error {

	if f.format {

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

	if ru == '`' {
		f.pushOut('"')
		f.popState()
		return nil
	}

	if ru == '\\' {
		return fmt.Errorf("character \\ found in multiline string")
	}

	if rep, ok := needsReplacement(ru); ok {
		f.pushOut('\\')
		f.pushOut(rep)
		return nil
	}

	if unicode.IsSpace(ru) {
		f.pushOut(' ')
		return nil
	}

	f.pushOut(ru)
	return nil
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
	return nil
}

func (o *ObjectState) Next(ru rune, f *Filter) error {

	if unicode.IsSpace(ru) || unicode.IsControl(ru) {
		if ru == '\n' {
			o.lineBreaks++
		}
		return nil
	}

	// check if comments need to be dispatched
	dispatch, err := dispatchComment(f, nil)
	if err != nil {
		return err
	}

	if dispatch {
		return ErrDontAdvance
	}

	switch o.internalState {
	case ObjIntNext:

		if ru == '}' {
			return o.pop(f)
		}

		f.pushOut(',')
		o.internalState = ObjIntNextAfterComma
		return ErrDontAdvance

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
			f.pushState(&KeyState{})
			return nil
		}

		f.pushState(&KeyNoQuoteState{})
		return ErrDontAdvance

	case ObjInternalKey:

		if ru == ':' {
			f.pushOut(ru)
			o.internalState = ObjInternalDelimiter
			return nil
		}

		return Errorf("error parsing object rune: %v", f.ring.Position(), string(ru))

	case ObjInternalDelimiter:

		o.internalState = ObjInternalValue
		switch ru {
		case '[':
			f.pushOut(ru)
			f.pushState(&ArrayState{})
			return nil

		case '{':
			f.pushOut(ru)
			f.pushState(&ObjectState{})
			return nil

		case '"':
			f.pushOut(ru)
			f.pushState(&ValueState{})
			return nil

		case '`':
			if f.format {
				f.pushOut(ru)
			} else {
				f.pushOut('"')
			}
			f.pushState(&ValueMultilineState{})
			return nil
		default:
			f.pushState(&ValueNoQuoteState{})
			return ErrDontAdvance
		}

	case ObjInternalValue:

		if ru == '}' {
			return o.pop(f)
		}

		if ru == ',' {
			o.internalState = ObjIntNext
			return nil
		}

		if f.format {
			if o.lineBreaks == 0 {
				f.pushOut(' ')
			}
			o.internalState = ObjIntNextAfterComma
		} else {
			o.internalState = ObjIntNext
		}
		return ErrDontAdvance
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

func (a *ArrayState) Next(ru rune, f *Filter) error {

	if unicode.IsSpace(ru) || unicode.IsControl(ru) {
		if ru == '\n' {
			a.lineBreaks++
		}

		a.spaceOrControls++
		return nil
	}

	// check if comments need to be dispatched
	dispatch, err := dispatchComment(f, nil)
	if err != nil {
		return err
	}

	if dispatch {
		return ErrDontAdvance
	}

	switch a.internalState {
	case ArrayIntAfterValue:

		spaceOrControls := a.spaceOrControls
		a.spaceOrControls = 0

		switch ru {
		case ']':
			a.internalState = ArrayIntValue
			return ErrDontAdvance
		case ',':
			a.internalState = ArrayIntValue
			if f.format {
				f.pushOut(',')
			}
			return nil
		}

		if spaceOrControls > 0 {
			if f.format && a.lineBreaks == 0 {
				f.pushOut(' ')
			}
			a.internalState = ArrayIntValue
			return ErrDontAdvance
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
			return nil
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
			f.pushState(&ArrayState{})
			return nil

		case '{':
			f.pushOut(ru)
			f.pushState(&ObjectState{})
			return nil

		case '"':
			f.pushOut(ru)
			f.pushState(&ValueState{})
			return nil

		case '`':
			if f.format {
				f.pushOut(ru)
			} else {
				f.pushOut('"')
			}
			f.pushState(&ValueMultilineState{})
			return nil
		default:
			f.pushState(&ValueNoQuoteState{})
			return ErrDontAdvance
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

func (*CommentState) Next(ru rune, f *Filter) error {

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
	return ErrDontAdvance
}

type CommentMultiLineState struct {
	escaped bool
}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(ru rune, f *Filter) error {

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
			return nil
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
		err := state.Next(f.ring.Peek(), f)
		if err != nil && !errors.Is(err, ErrDontAdvance) {
			return err
		}

		if !errors.Is(err, ErrDontAdvance) {
			err = f.ring.Advance()
			if err != nil {
				return err
			}
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
