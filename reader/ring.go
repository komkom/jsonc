package reader

import "fmt"

type ReadRune func() (r rune, size int, err error)

type Ring struct {
	buf         []rune
	readRune    ReadRune
	minSize     int
	maxSize     int
	position    int
	absPosition int
}

func NewRing(maxSize int, minSize int, readRune ReadRune) (r *Ring, err error) {

	if maxSize <= minSize {
		return nil, fmt.Errorf("maxSize(%v) <= minSize(%v)", maxSize, minSize)
	}

	r = &Ring{readRune: readRune, minSize: minSize, maxSize: maxSize}
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
	return r.absPosition + r.position - len(r.buf)
}

func (r *Ring) Advance() error {

	if r.position+1 >= len(r.buf) {

		if len(r.buf) > r.maxSize {

			r.buf = r.buf[r.maxSize-r.minSize:]
			r.position -= (r.maxSize - r.minSize)
			if r.position < 0 {
				panic(fmt.Errorf("unexpected error"))
			}
		}

		err := r.fill()
		if err != nil {
			return err
		}
	}

	r.position += 1
	return nil
}

func (r *Ring) fill() error {
	ru, _, err := r.readRune()
	if err != nil {
		return err
	}

	r.buf = append(r.buf, ru)
	r.absPosition += 1
	return nil
}

func (r *Ring) Peek() rune {
	return r.buf[r.position]
}

func (r *Ring) Pop() error {

	if r.position == 0 && len(r.buf) == 0 {
		return nil
	}

	if r.position == 0 {
		return fmt.Errorf("buffer underrun")
	}

	r.position -= 1
	return nil
}
