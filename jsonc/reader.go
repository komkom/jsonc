package jsonc

import "io"

func New(r io.Reader, minimize bool, space string) (*Filter, error) {

	buf := NewBuffer(r, 256, 64)

	ring, err := NewRing(256, 64, func() (r rune, size int, err error) {
		return buf.ReadRune()
	})

	if err != nil {
		return nil, err
	}

	return NewFilter(ring, 256, !minimize, space), nil
}
