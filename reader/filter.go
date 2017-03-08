package reader

import (
	"fmt"
	"io"
	"unicode"
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
	done       bool
	err        error
}

type baseState struct {
	closed bool
}

func (b baseState) Close() {
	b.closed = true
}

func (b baseState) Open() bool {
	return !b.closed
}

// root state
type RootState struct {
	baseState
}

func (r RootState) Type() TokenType {
	return Root
}

func (r RootState) Next(f *Filter) (err error) {

	for {
		ru := f.ring.Peek()
		if ru == '{' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(ObjectState{})
			return
		}

		if ru == '[' {
			f.pushOut(ru)
			err = f.ring.Advance()
			f.pushState(ArrayState{})
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

func (o KeyState) Type() TokenType {
	return Key
}

func (r KeyState) Next(f *Filter) (err error) {

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
	KeyState
}

func (o ValueState) Type() TokenType {
	return Value
}

type KeyNoQuoteState struct {
	baseState
}

func (o KeyNoQuoteState) Type() TokenType {
	return KeyNoQuote
}

func (r KeyNoQuoteState) Next(f *Filter) (err error) {

	var escaped bool
	for {
		ru := f.ring.Peek()

		if !escaped && ru == ':' {
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

type ValueNoQuoteState struct {
	baseState
}

func (o ValueNoQuoteState) Type() TokenType {
	return ValueNoQuote
}

func (r ValueNoQuoteState) Next(f *Filter) (err error) {

	var escaped bool
	var cval []byte

	for {
		ru := f.ring.Peek()

		if !escaped && unicode.IsSpace(ru) {
			// validate
			// insert " if needed
			r.Close()
			err = f.ring.Advance()
			return
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

func (o ObjectState) Type() TokenType {
	return Object
}

func (r ObjectState) Next(f *Filter) (err error) {

	s := f.peekState()

	var hasKeyDelimiter bool
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
						f.pushState(ArrayState{})
						return
					}

					if ru == '{' {
						f.pushOut(ru)
						err = f.ring.Advance()
						f.pushState(ObjectState{})
						return
					}

					if ru == '"' {
						f.pushOut(ru)
						err = f.ring.Advance()
						f.pushState(KeyState{})
						return
					}

					f.pushState(KeyNoQuoteState{})
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

			break
		case Value, ValueNoQuote, Object, Array:
			break
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

func (o ArrayState) Type() TokenType {
	return Object
}

func (r ArrayState) Next(f *Filter) (err error) {

	var hasComma bool
	for {
		ru := f.ring.Peek()

		if ru == ',' {
			if hasComma {
				err = fmt.Errorf("invalid comma at pos %v", f.ring.Position())
				return
			}
			hasComma = true
			continue
		}

		if ru == ']' {
			f.pushOut(ru)
			r.Close()
			err = f.ring.Advance()
			return
		}

		if hasComma {

			if ru == '[' {
				f.pushOut(',')
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(ArrayState{})
				return
			}

			if ru == '{' {
				f.pushOut(',')
				f.pushOut(ru)
				err = f.ring.Advance()
				f.pushState(ObjectState{})
				return
			}

			if ru == '"' {

				f.pushOut(ru)
				err = f.ring.Advance()
				// TODO value state
				return
			}

			if !unicode.IsSpace(ru) {

				// TODO push non quote value
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

func (f *Filter) Read(p []byte) (n int, err error) {

	tor := f.outMinSize
	if tor > len(p) {
		tor = len(p)
	}

	for tor > len(f.outbuf) {

		if f.done {
			n = 0
			err = io.EOF
		}

		err = f.fill()
		if err != nil {
			return
		}
	}

	for i := 0; i < tor; i++ {
		p[i] = f.outbuf[i]
	}

	f.outbuf = f.outbuf[tor:]

	if f.done {
		err = io.EOF
	}

	return
}

func (f *Filter) fill() error {

	if f.done {
		return nil
	}

	state := f.peekState()

	for f.outMinSize > len(f.outbuf) {

		err := state.Next(f)

		if err != nil {
			return err
		}

		// if the current state is closed get next open one.
		if !f.peekState().Open() {

			/*
				err := f.closeState(state.Type())
				if err != nil {
					return err
				}
			*/

			state, err = f.openState()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

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

func (f *Filter) peekState() State {
	state := f.rootState
	if len(f.stack) > 0 {
		state = f.stack[len(f.stack)-1]
	}
	return state
}

func (f *Filter) pushState(s State) {
	f.stack = append(f.stack, s)
}

func (f *Filter) pushOut(r rune) {
	f.outbuf = append(f.outbuf, byte(r))
}
