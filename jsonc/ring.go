package jsonc

import (
	"fmt"
)

type ReadRune func() (r rune, size int, err error)

type Ring struct {
	buf         []rune
	readRune    ReadRune
	position    int
	absPosition int
}

func NewRing(readRune ReadRune) (r *Ring, err error) {

	r = &Ring{readRune: readRune}
	if readRune != nil {
		err = r.fill()
	}
	return
}

func (r *Ring) Clear(rr ReadRune) (err error) {
	r.readRune = rr
	r.buf = nil
	r.position = 0
	r.absPosition = 0

	err = r.fill()
	return
}

func (r *Ring) Position() int {
	return r.absPosition
}

func (r *Ring) Advance() error {

	if r.position+1 >= len(r.buf) {

		err := r.fill()
		if err != nil {
			return err
		}
	}

	r.absPosition += 1
	r.position += 1
	return nil
}

func (r *Ring) fill() error {
	ru, _, err := r.readRune()
	if err != nil {
		return err
	}
	r.buf = append(r.buf, ru)
	return nil
}

func (r *Ring) Peek() rune {
	return r.buf[r.position]
}

func (r *Ring) Pop() error {

	if r.position == 0 {
		return fmt.Errorf("buffer underrun")
	}

	r.position -= 1
	return nil
}
