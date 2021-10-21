package jsonc

import "io"

func New(r io.RuneReader, minimize bool, space string) (*Filter, error) {

	ring, err := NewRing(256, 64, r.ReadRune)

	if err != nil {
		return nil, err
	}

	return NewFilter(ring, 256, !minimize, space), nil
}
