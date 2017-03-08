package reader

/*
import (
	"fmt"
	"io"
	"unicode/utf8"
)

type Buffer struct {
	reader   io.Reader
	high     int
	buf      []byte
	position int
	cerr     error
}

func NewBuffer(reader io.Reader, high int) (b *Buffer) {

	b = &Buffer{reader: reader, high: high}
	return
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.refill()

	if len(p) == 0 {
		return
	}

	n = len(b.buf) - b.position

	if n == 0 {
		err = io.EOF
		return
	}

	if n > len(p) {
		n = len(p)
	}

	for i := 0; i < n; i++ {
		p[i] = b.buf[b.position+i]
	}

	b.position += n

	if b.cerr != nil {
		err = b.cerr
		return
	}

	return
}

func (b *Buffer) ReadRune() (r rune, err error) {
	b.refill()

	if b.cerr != nil {
		err = b.cerr
		return
	}

	if len(b.buf) <= b.position {
		err = io.EOF
		return
	}

	r, s := utf8.DecodeRune(b.buf[b.position:])

	b.position += s

	if r == utf8.RuneError {
		err = fmt.Errorf("rune error")
		b.cerr = err
		return
	}

	return
}

func (b *Buffer) refill() {
	if b.cerr != nil {
		return
	}

	if b.position >= b.high {

		b.buf = b.buf[b.high:]
		b.position -= b.high
	}

	if b.high > len(b.buf)-b.position {
		buf := make([]byte, b.high/2)
		for b.high > len(b.buf)-b.position {
			n, err := b.reader.Read(buf)
			if n == 0 {
				break
			}

			if err != nil {
				b.cerr = err
				return
			}

			b.buf = append(b.buf, buf[:n]...)
		}

		b.buf = b.buf[b.position:]
		b.position = 0
	}

	return
}
*/
