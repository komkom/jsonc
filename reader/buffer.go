package reader

import (
	"fmt"
	"io"
	"unicode/utf8"
)

type Buffer struct {
	reader       io.Reader
	high         int
	low          int
	backedBuffer []byte
	active       []byte
	cerr         error
}

func NewBuffer(reader io.Reader, high int, low int) (b *Buffer) {

	b = &Buffer{reader: reader, high: high, low: low, backedBuffer: make([]byte, high+10)}
	return
}

func (b *Buffer) ReadRune() (r rune, size int, err error) {

	b.refill()

	if b.cerr != nil {
		err = b.cerr
		return
	}

	r, size = utf8.DecodeRune(b.active)

	b.active = b.active[size:]

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

	if b.low > len(b.active) {

		totalLength := len(b.active)

		copy(b.backedBuffer, b.active)

		buf := b.backedBuffer[len(b.active):]
		last := 0

		for b.high > totalLength {

			buf = buf[last:]

			n, err := b.reader.Read(buf)
			if n == 0 {
				break
			}

			if err != nil {
				b.cerr = err
				return
			}

			last = n
			totalLength += n
		}

		b.active = b.backedBuffer[:totalLength]
	}

	return
}
