package reader

import (
	"fmt"
	"unicode"

	"github.com/komkom/jsonc/json"
)

type TokenType int

type State interface {
	Type() TokenType
	Next(f *Filter) (err error)
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
)

type Filter struct {
	ring       *Ring
	outbuf     []byte
	outMinSize int
	stack      []State
	rootState  State
}

func NewFilter(ring *Ring, outMinSize int, rootState State) *Filter {

	return &Filter{ring: ring, outMinSize: outMinSize, rootState: rootState}
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

// root state
type RootState struct {
	baseState
}

func (r *RootState) Type() TokenType {
	return Root
}

func (r *RootState) Next(f *Filter) (err error) {

	for {
		ru := f.ring.Peek()

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
			err = fmt.Errorf("invalid first character: %v", string(ru))
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

func (r *KeyState) Next(f *Filter) (err error) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {

			//f.closeState()
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

func (v *ValueState) Next(f *Filter) (err error) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == '"' {
			//f.closeState()
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

func (r *KeyNoQuoteState) Next(f *Filter) (err error) {

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
			f.ring.Pop()
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

func (r *ValueNoQuoteState) Next(f *Filter) (err error) {

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
}

func (o *ObjectState) Type() TokenType {
	return Object
}

func (o *ObjectState) Next(f *Filter) (err error) {

	s := f.peekState()

	var hasKeyDelimiter bool
	var hasComma bool
	for {
		ru := f.ring.Peek()

		switch s.Type() {
		case Key, KeyNoQuote:

			if hasKeyDelimiter {

				if !unicode.IsSpace(ru) {
					hasKeyDelimiter = false

					if ru == ',' {
						err = fmt.Errorf("invalid comma at pos %v", f.ring.Position())
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
			}

			if ru == ':' {

				if hasKeyDelimiter {
					err = fmt.Errorf("invalid : at pos %v", f.ring.Position())
					return
				}

				hasKeyDelimiter = true
				f.pushOut(ru)
			}

		case Value, ValueNoQuote, Object, Array:

			if ru == ',' {
				if hasComma {
					err = fmt.Errorf("invalid comma at pos %v", f.ring.Position())
					return
				}
				hasComma = true
				break
			}

			if ru == '}' {

				//f.closeState()
				o.Close()
				f.pushOut(ru)
				err = f.ring.Advance()
				return
			}

			if ru == '"' {
				/*
					if s.Type() == Value || s.Type() == ValueNoQuote {
						f.pushOut(',')
					}
				*/

				if s.Type() == Object && !s.Open() {
					f.pushOut(',')
				}

				if s.Type() != Object {
					f.pushOut(',')
				}

				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(&KeyState{})
				return
			}

			if !unicode.IsSpace(ru) {

				if s.Type() == Value || s.Type() == ValueNoQuote {
					f.pushOut(',')
				}
				f.pushState(&KeyNoQuoteState{})
				return
			}
		}

		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

type ArrayState struct {
	baseState
}

func (o *ArrayState) Type() TokenType {
	return Array
}

func (r *ArrayState) Next(f *Filter) (err error) {

	s := f.peekState()

	var hasComma bool
	for {
		ru := f.ring.Peek()

		if unicode.IsSpace(ru) {
			goto next
		}

		if ru == ',' {
			if hasComma {
				err = fmt.Errorf("invalid comma at pos %v", f.ring.Position())
				return
			}
			hasComma = true
			goto next
		}

		if ru == ']' {
			f.pushOut(ru)
			//f.closeState()
			r.Close()
			err = f.ring.Advance()
			return
		}

		if hasComma || (s.Type() == Array && s.Open()) {

			if hasComma {
				f.pushOut(',')
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

		err = fmt.Errorf("invalid character %v at pos %v", string(ru), f.ring.Position())
		return

	next:
		err = f.ring.Advance()
		if err != nil {
			return
		}
	}

	return
}

func (f *Filter) Read(p []byte) (n int, err error) {

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

	if len(f.outbuf) < n {
		n = len(f.outbuf)
	}

	for i := 0; i < n; i++ {
		p[i] = f.outbuf[i]
	}

	f.outbuf = f.outbuf[n:]

	return
}

func (f *Filter) fill() error {

	state, err := f.peekFirstOpenState()
	if err != nil {
		return err
	}

	for f.outMinSize > len(f.outbuf) {

		f.printStack()
		fmt.Printf("__state %v\n", state.Type())

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

/*
func (f *Filter) closeState(stateType TokenType) error {

	for i := len(f.stack) - 1; i >= 0; i-- {

		s := f.stack[i]

		if s.Type() == stateType {
			s.Close()
			return nil
		}

		if s.Open() {
			break
		}
	}

	return fmt.Errorf("incorrect stack")
}
*/

/*
func (f *Filter) openState() (s State, err error) {

	for i := len(f.stack) - 1; i >= 0; i-- {

		s = f.stack[i]
		if s.Open() {
			return
		}
	}

	err = fmt.Errorf("no open state")
	return
}
*/

func (f *Filter) peekState() State {
	state := f.rootState
	if len(f.stack) > 0 {
		state = f.stack[len(f.stack)-1]
	}
	return state
}

func (f *Filter) peekFirstOpenState() (s State, err error) {

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

	err = fmt.Errorf("no open states found")
	return
}

func (f *Filter) pushState(s State) {
	f.stack = append(f.stack, s)
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

/*
func (f *Filter) closeState() error {
	if len(f.stack) == 0 {
		f.rootState.Close()
		return nil
	}

	for i := len(f.stack) - 1; i >= 0; i-- {
		s := f.stack[i]
		if s.Open() {
			s.Close()
			return nil
		}
	}

	return fmt.Errorf("no state to close")
}o*/
