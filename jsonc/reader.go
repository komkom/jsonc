package jsonc

import "io"

func New(r io.RuneReader, minimize bool, space string) (*Filter, error) {

	ring, err := NewRing(r.ReadRune)

	if err != nil {
		return nil, err
	}

	return NewFilter(ring, 256, !minimize, space), nil
}
