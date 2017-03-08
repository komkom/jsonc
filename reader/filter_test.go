package reader

import (
	"bytes"
	"fmt"
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

/*
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
*/

func TestFilter(t *testing.T) {

	//s := `{ t:ttestt,x:x, z:[ "test" ]      }`

	//s := `{ z:[ "test", test2, ["x"], {"x":"x"} ]      }`

	//s := `[     ]`
	//s := `[{x:x}, [u], "test"]`
	s := `{ k1:v, k2:v2}`
	b := strings.NewReader(s)

	ring, err := NewRing(8, 4, func() (r rune, size int, err error) {
		return b.ReadRune()
	})

	if err != nil {
		t.Fatalf("error: %v", err.Error())
	}

	f := NewFilter(ring, 16, &RootState{})

	buf := &bytes.Buffer{}

	n, err := buf.ReadFrom(f)
	if err != nil {
		fmt.Printf("err: %v", err.Error())
		return
	}

	fmt.Printf("read %v out: \n%s\n", n, buf.Bytes())
}
