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

	s := ` /* json description */
	{ 
		/* test */
		t:ttestt /* some other comment */ ,
		x:x, // should test this 
		z:[ "test" /* we are in  and in a comment http://www.some.url.com   */  
		// and a single line comment
		], 
		o:123.65e+7,      
	}`

	//s := `{ z:[ "test", test2, ["x"], {"x":"x", x2:[uu]}, null ]      }`

	//s := `[     ]`
	//s := `[{x:x}, [u], "test"]`
	//s := `{   x:   y   }   `
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
		fmt.Printf("err: %v\n", err.Error())
		return
	}

	if !f.Done() {
		fmt.Printf("json could not be parsed %v\n", f.Error())
	}

	fmt.Printf("read %v out: \n%s\n", n, buf.Bytes())
}
