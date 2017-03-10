package reader

import "io"

func New(r io.Reader, minimize bool) io.Reader {

	buf := NewBuffer(r, 256, 64)

	ring, err := NewRing(256, 64, func() (r rune, size int, err *errorf) {

		r, size, cerr := buf.ReadRune()
		if cerr != nil {
			err = cerror(cerr)
		}
		return
	})

	if err != nil {
		panic(err)
	}

	return NewFilter(ring, 256, &RootState{}, !minimize)
}
