package reader

import (
	"bytes"
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

type TestJson struct {
	JsonCString           string
	ExpectedJsonString    string
	ExpectedStringInError string
}

func JsonData() []TestJson {

	return []TestJson{
		TestJson{
			JsonCString: `{"y":"y"}`,
		},
		TestJson{
			JsonCString:        `{x:x}`,
			ExpectedJsonString: `{"x":"x"}`,
		},

		TestJson{
			JsonCString: ` /* json description */
	{ 
		/* test */
		t:ttestt /* some other comment */ ,
		x:x, // should test this 
		z:[ "test" /* we are in  and in a comment http://www.some.url.com   */  
		// and a single line comment
		], 
		o:123.65e+7,      
	}/* can you also add comments here */`,
			ExpectedJsonString: `{"t":"ttestt","x":"x","z":["test"],"o":123.65e+7}`,
		},
		TestJson{
			JsonCString:           `[ x, y, z,, ]`,
			ExpectedStringInError: `pos: 10 invalid comma`,
		},
		TestJson{
			JsonCString:        `[ x, y, z, x:x, [ "t", "t", {j:i,o:o,i:[1,2,3,4,5]}] ]`,
			ExpectedJsonString: `["x","y","z","x:x",["t","t",{"j":"i","o":"o","i":[1,2,3,4,5]}]]`,
		},
		TestJson{
			JsonCString: ` [ 
			x, y, z, 
			x:x, 
				[ "t",   "t",   
					{
						j:i, // test comment
						o:o,
						i:[1,2,3,4,5]
						/* test this */
					}
				] 
			]`,
			ExpectedJsonString: `["x","y","z","x:x",["t","t",{"j":"i","o":"o","i":[1,2,3,4,5]}]]`,
		},
	}
}

func TestJsonParser2(t *testing.T) {

	data := JsonData()

	ring, err := NewRing(8, 4, nil)
	if err != nil {
		panic(err)
	}

	f := NewFilter(ring, 16, &RootState{})

	for idx, d := range data {

		t.Log(`testing json`)

		b := strings.NewReader(d.JsonCString)
		ring.Clear(func() (r rune, size int, err *errorf) {
			r, size, cerr := b.ReadRune()
			if cerr != nil {
				err = cerror(cerr)
			}
			return
		})

		f.Clear()

		buf := &bytes.Buffer{}

		_, err := buf.ReadFrom(f)
		t.Logf("json out:\n %s", buf.Bytes())

		if d.ExpectedStringInError != `` {

			if err == nil {
				t.Fatalf("idx: %v no error found containing: %v: %v", idx, d.ExpectedStringInError)
				return
			}

			if !strings.Contains(err.Error(), d.ExpectedStringInError) {
				t.Fatalf("idx: %v error: %v not containing: %v", idx, err.Error(), d.ExpectedStringInError)
			}
		} else if err != nil && err.Error() != `EOF` {
			t.Fatalf("idx: %v unexpected error found: %v", idx, err.Error())
			return
		}

		if d.ExpectedStringInError == `` {
			if !f.Done() {
				t.Fatalf("idx: %v parsing not done", idx)
			}
		}

		if d.ExpectedJsonString != `` {

			if d.ExpectedJsonString != string(buf.Bytes()) {
				t.Fatalf("idx: %v expected json does not match")
			}
		}
	}
}

/*
func TestJsonParser(t *testing.T) {

	s := `[ x ]`

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
*/
