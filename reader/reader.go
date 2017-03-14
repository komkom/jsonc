package reader

import "io"

func New(r io.Reader, minimize bool) (reader io.Reader, err error) {

	buf := NewBuffer(r, 256, 64)

	ring, errf := NewRing(256, 64, func() (r rune, size int, err *errorf) {

		r, size, cerr := buf.ReadRune()
		if cerr != nil {
			err = cerror(cerr)
		}
		return
	})

	if errf != nil {
		err = errf
		return
	}

	reader = NewFilter(ring, 256, &RootState{}, !minimize)
	return
}
