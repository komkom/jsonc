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

	for {
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
	}
}

type KeyState struct{}

func (o *KeyState) Type() TokenType {
	return Key
}

func (r *KeyState) Next(f *Filter) error {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {

			f.pushOut(ru)
			f.popState()
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

type ValueState struct{}

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
			f.pushOut(ru)
			f.popState()
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

		if ru == '\\' {
			return Errorf("invalid key", f.ring.Position())
		}

		f.pushOut(ru)

		err := f.ring.Advance()
		if err != nil {
			return err
		}
	}
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

	for {
		ru := f.ring.Peek()

		if unicode.IsSpace(ru) || ru == ',' || ru == '}' || ru == ']' || ru == '/' {

			return renderValue()
		}

		if ru == '\\' {
			return Errorf("invalid identifier", f.ring.Position())
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

type ValueMultilineState struct{}

func (v *ValueMultilineState) Type() TokenType {
	return Value
}

func (v *ValueMultilineState) Next(f *Filter) error {

	if f.format {
		for {
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
		}
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

var (
	ObjIntNext ObjInternalState = 0

	ObjInternalKey       ObjInternalState = 1
	ObjInternalDelimiter ObjInternalState = 2
	ObjInternalValue     ObjInternalState = 3
)

type ObjectState struct {
	internalState ObjInternalState
	hasLineBreak  bool
}

func (o *ObjectState) Type() TokenType {
	return Object
}

func (o *ObjectState) Next(f *Filter) error {

	for {
		ru := f.ring.Peek()

		if f.format {
			if ru == '\n' {
				o.hasLineBreak = true
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

		if unicode.IsSpace(ru) {
			goto next
		}

		switch o.internalState {
		case ObjIntNext:

			if ru == '}' {
				f.pushOut(ru)
				f.popState()
				return f.ring.Advance()
			}

			if o.hasLineBreak {
				f.pushSpaces(f.indent())
			}
			o.hasLineBreak = false

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
				f.pushOut(ru)
				f.popState()
				return f.ring.Advance()
			}

			if ru == ',' {
				f.pushOut(ru)
				o.internalState = ObjIntNext
				return f.ring.Advance()
			}

			if !f.format {
				f.pushOut(',')
			} else {
				f.pushOut(' ')
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
	init         bool
	hasComma     bool
	hasLineBreak bool
}

func (ArrayState) Type() TokenType {
	return Array
}

func (a *ArrayState) Next(f *Filter) error {

	var dontResetComma bool

	defer func() {
		if !dontResetComma {
			a.hasComma = false
		}
	}()

	for {
		ru := f.ring.Peek()

		if f.format {
			if ru == '\n' {
				a.hasLineBreak = true
				f.pushOut(ru)
			}
		}

		// check if comments need to be dispatched
		dispatch, err := dispatchComment(f, nil)
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
			if a.hasComma {
				return Errorf("invalid comma", f.ring.Position())
			}
			a.hasComma = true

			if f.format {
				f.pushOut(',')
			}

			goto next
		}

		if a.hasLineBreak {
			f.pushSpaces(f.indent())
		}
		a.hasLineBreak = false

		if ru == ']' {
			f.pushOut(ru)
			f.popState()
			return f.ring.Advance()
		}

		if a.init {
			if !f.format {
				f.pushOut(',')
			}
		}
		a.init = true

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

type CommentState struct{}

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
				f.popState()
			}
			return err
		}
	}
}

type CommentMultiLineState struct{}

func (c *CommentMultiLineState) Type() TokenType {
	return CommentMultiLine
}

func (c *CommentMultiLineState) Next(f *Filter) error {

	var escaped bool
	for {
		ru := f.ring.Peek()
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
