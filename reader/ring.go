package reader

import "fmt"

type ReadRune func() (r rune, size int, err *errorf)

type Ring struct {
	buf         []rune
	readRune    ReadRune
	minSize     int
	maxSize     int
	position    int
	absPosition int
}

func NewRing(maxSize int, minSize int, readRune ReadRune) (r *Ring, err *errorf) {

	if maxSize <= minSize {
		err = errorFmt("maxSize(%v) <= minSize(%v)", maxSize, minSize)
		return
	}

	r = &Ring{readRune: readRune, minSize: minSize, maxSize: maxSize}
	if readRune != nil {
		err = r.fill()
	}
	return
}

func (r *Ring) Clear(rr ReadRune) (err *errorf) {
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

func (r *Ring) Advance() (err *errorf) {

	if r.position+1 >= len(r.buf) {

		if len(r.buf) > r.maxSize {

			r.buf = r.buf[r.maxSize-r.minSize:]
			r.position -= (r.maxSize - r.minSize)
			if r.position < 0 {
				panic(fmt.Errorf("unexpected error"))
			}
		}

		err = r.fill()
		if err != nil {
			return
		}
	}

	r.position += 1
	return
}

func (r *Ring) fill() (err *errorf) {
	ru, _, err := r.readRune()
	if err != nil {
		return
	}

	r.buf = append(r.buf, ru)
	r.absPosition += 1
	return
}

func (r *Ring) Peek() rune {

	return r.buf[r.position]
}

func (r *Ring) Pop() *errorf {

	if r.position == 0 && len(r.buf) == 0 {
		return nil
	}

	if r.position == 0 {
		return errorFmt("buffer underrun")
	}

	r.position -= 1
	return nil
}
