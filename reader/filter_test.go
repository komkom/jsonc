package reader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type TestJson struct {
	JsonCString           string
	ExpectedJsonString    string
	ExpectedStringInError string
}

func Test2(t *testing.T) {

	j := `{ "x":"v", "x":v}`

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(j), &m)

	fmt.Printf("_____\n %v\nerr: %v\n", m, err)

}

func JsonToFormat() []string {

	return []string{
		`{

			zz:uu /* test comment */


			x:x
// some comment
	y:y 
	
	/* here wie
have some thing */
	z:z
	arr: [

	1,2,3,4,5,6, {
		
		x/*an other*/: /*some comment*/  x /* about that */, t:[x,13,5, // and this
	6,       /* about this seven */ 7,




	8,9,["4",5,7,8,9],2,3,

	4,5]}		
	], u:
	{x:
y}
}
`,
		`
// test

// application config data
{
	api : {

		// test
		// tets

		// test this
		// qa configuration against a local server
		qa : {

			// some comment here
			// some comment here
			baseurl : test // "http://localhost:1234/myapi" /* use non ssl connection */
	
			headers : [ // ugly header setup to make it work on local
			{ key: auth  value: sometoken /* auto token for dummy user */ },
			{ key: source value: local }
			]
			x : { x: y }
		}
		live : {
		
			// this should redirect to myotherdomain.com
			baseurl : "https://api.mydomain.com"

			headers /* test */  : [
			{ key : auth,  value : someothertoken}, // iiiii
			{ key :  source, value : remote}
			]
		}
	}
}
`,
	}

}

func TestFormatter(t *testing.T) {
	data := JsonToFormat()

	ring, err := NewRing(256, 64, nil)
	if err != nil {
		panic(err)
	}

	f := NewFilter(ring, 256, &RootState{}, true)

	for _, d := range data {

		t.Log(`testing json`)

		b := strings.NewReader(d)
		ring.Clear(func() (r rune, size int, err *errorf) {
			r, size, cerr := b.ReadRune()
			if cerr != nil {
				err = cerror(cerr)
			}
			return
		})

		f.Clear()

		buf := &bytes.Buffer{}

		buf.ReadFrom(f)

		fmt.Printf("___formatted\n%s", buf.Bytes())

	}

}

func JsonData() []TestJson {

	return []TestJson{
		TestJson{
			JsonCString:        `{y/* test */:y/* test */}`,
			ExpectedJsonString: `{"y":"y"}`,
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
				[ "t", /* some test */  "t",   
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
		TestJson{
			JsonCString: `{ 1:1, /* */ 2:2,3:3,4:4,5:5}`,
		},
		TestJson{
			JsonCString: `{ 1:1, /* */ 2: /* some other comment */ 2,
			/* another comment */ 7: [1,2,3,4,5,6],
			3:3,4:4,5:5 /*hmm*/ ,} // test comment at the end`,
			ExpectedJsonString: `{"1":1,"2":2,"7":[1,2,3,4,5,6],"3":3,"4":4,"5":5}`,
		},
		TestJson{
			JsonCString: `{ test// test : value 
			: key  v : h }`,
		},
	}
}

func TestJsonParser2(t *testing.T) {

	data := JsonData()

	ring, err := NewRing(256, 64, nil)
	if err != nil {
		panic(err)
	}

	f := NewFilter(ring, 256, &RootState{}, false)

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
				t.Fatalf("idx: %v expected json does not match", idx)
			}
		}
	}
}
