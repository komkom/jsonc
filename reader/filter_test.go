package reader

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func backtrace(r *Ring) (b string) {

	for {
		b += string(r.Peek())
		err := r.Pop()
		if err != nil {
			break
		}
	}

	return
}

func TestTest(t *testing.T) {

	s := `some test st uu tt`
	b := strings.NewReader(s)

	ring, err := NewRing(8, 4, func() (r rune, size int, err error) {
		return b.ReadRune()
	})

	if err != nil {
		t.Fatalf("error: %v", err.Error())
	}

	for {
		ru := ring.Peek()

		fmt.Printf("%s", string(ru))

		err := ring.Advance()
		if err == io.EOF {
			fmt.Println(``)
			break
		}
	}

	v := backtrace(ring)
	fmt.Printf("b: %v", v)
}
