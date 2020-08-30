package jsonc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestJson struct {
	JsonCString           string
	ExpectedJsonString    string
	ExpectedStringInError string
}

func JsonData2() []TestJson {

	return []TestJson{
		{
			JsonCString: ` 
				[ "t",  "t", [i]
				]`,
			ExpectedJsonString: `[["t","t",{"j":"i","o":"o","i":[1,2,3,4,5]}]]`,
		},
	}
}

func JsonData() []TestJson {

	return []TestJson{
		{
			JsonCString:        "[]",
			ExpectedJsonString: `[]`,
		},
		{
			JsonCString:        "{x:x,}",
			ExpectedJsonString: `{"x":"x"}`,
		},
		{
			JsonCString:        "`value`",
			ExpectedJsonString: `"value"`,
		},
		{
			JsonCString:        `"value"`,
			ExpectedJsonString: `"value"`,
		},
		{
			JsonCString:        "{some: `text`}",
			ExpectedJsonString: `{"some":"text"}`,
		},
		{
			JsonCString:        "[some `text`,]",
			ExpectedJsonString: `["some","text"]`,
		},
		{
			JsonCString:        `{x:x}`,
			ExpectedJsonString: `{"x":"x"}`,
		},
		{
			JsonCString:        `{y/* test */:y/* test */}`,
			ExpectedJsonString: `{"y":"y"}`,
		},
		{
			JsonCString: ` /* json description */
	{ 
		/* test */
		t:ttestt /* some other comment */ ,
		x:x, // should test this 
		z:[ "test" /* we are in  and in a comment http://www.some.url.com   */  
		// and a single line comment
		], 
		o:123.65e+7      
	}/* can you also add comments here */`,
			ExpectedJsonString: `{"t":"ttestt","x":"x","z":["test"],"o":123.65e+7}`,
		},
		{
			JsonCString:           `[ x, y, z,, ]`,
			ExpectedStringInError: `empty no quote state`,
		},
		{
			JsonCString:        `[ x, y, z, xx, [ "t", "t", {j:i,o:o,i:[1,2,3,4,5]}] ]`,
			ExpectedJsonString: `["x","y","z","xx",["t","t",{"j":"i","o":"o","i":[1,2,3,4,5]}]]`,
		},
		{
			JsonCString: ` [ 
			x, y, z, 
			xx, 
				[ "t", /* some test */  "t",   
					{
						j:i, // test comment
						o:o,
						i:[1,2,3,4,5]
						/* test this */
					}
				] 
			]`,
			ExpectedJsonString: `["x","y","z","xx",["t","t",{"j":"i","o":"o","i":[1,2,3,4,5]}]]`,
		},
		{
			JsonCString: `{ 1:1, /* */ 2:2,3:3,4:4,5:5}`,
		},
		{
			JsonCString: `{ 1:1, /* */ 2: /* some other comment */ 2,
			/* another comment */ 7: [1,2,3,4,5,6],
			3:3,4:4,5:5 /*hmm*/ } // test comment at the end`,
			ExpectedJsonString: `{"1":1,"2":2,"7":[1,2,3,4,5,6],"3":3,"4":4,"5":5}`,
		},
		{
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

	f := NewFilter(ring, 256, false, " ")

	for idx, d := range data {

		t.Log(idx, ` testing json`)

		b := strings.NewReader(d.JsonCString)
		ring.Clear(func() (r rune, size int, err error) {
			return b.ReadRune()
		})

		f.Clear()

		buf := &bytes.Buffer{}

		_, err := buf.ReadFrom(f)
		t.Logf("idx: %v json out:\n %s", idx, buf.Bytes())

		if d.ExpectedStringInError != `` {

			if err == nil {
				t.Fatalf("idx: %v no error found containing: %v", idx, d.ExpectedStringInError)
				return
			}

			if !strings.Contains(err.Error(), d.ExpectedStringInError) {
				t.Fatalf("idx: %v error: %v not containing: %v", idx, err.Error(), d.ExpectedStringInError)
			}
		} else if err != nil && errors.Is(err, io.EOF) {
			t.Fatalf("idx: %v unexpected error found: %v", idx, err.Error())
			return
		}

		if d.ExpectedStringInError == `` {
			assert.True(t, f.Done())
		}

		if d.ExpectedJsonString != `` {
			assert.Equal(t, d.ExpectedJsonString, string(buf.Bytes()))
		}
	}
}

func JsonDataFmt() []TestJson {

	return []TestJson{
		{
			JsonCString: ` /* json description */
	{ 
		/* test */
		t:ttestt /* some other comment */ ,
		x:x, // should test this 
		z:[ "test" /* we are in  and in a comment http://www.some.url.com   */  
		// and a single line comment
		], 
		o:123.65e+7      
	}/* can you also add comments here */`,
			ExpectedJsonString: `{"t":"ttestt","x":"x","z":["test"],"o":123.65e+7}`,
		},

		{
			JsonCString: `{ 1:1, /* */ 2: /* some other comment */ 2,
			/* another comment */ 7: [1,2,3,4,5,6],
			3:3,4:4,5:5 /*hmm*/ } // test comment at the end`,
			ExpectedJsonString: `{"1":1,"2":2,"7":[1,2,3,4,5,6],"3":3,"4":4,"5":5}`,
		},
		{
			JsonCString: `{   }`,
		},
		{
			JsonCString: `{
			a:b, c:{
			d: e,
			e: f
			}`,
		},
	}
}

func JsonDataFmt2() []TestJson {

	return []TestJson{
		{
			JsonCString: `{a:a ,    b:b}`,
		},
		{
			JsonCString: `{ a:b, c:{ d: {

			t: t,
			r: {/*test
 asda
 asdf
*/
			t:1
			t:4
			}
			},
			e/*ii*/: f       //comment
			c: [ 
			1, 3, 4, 5,     6,
			7, 8,
			9 ,20]
			}}`,
		},
		{
			JsonCString: `{}`,
		},
		{
			JsonCString: `[]`,
		},
	}
}

func TestJsoncFmt(t *testing.T) {

	f, err := os.Open(`test.jsonc`)
	require.NoError(t, err)

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	fmt.Printf("dd %s\n", data)

	//data := JsonData()

	ring, err := NewRing(32, 16, nil)
	if err != nil {
		panic(err)
	}

	fl := NewFilter(ring, 16, true, " ")

	b := strings.NewReader(string(data))
	ring.Clear(func() (r rune, size int, err error) {
		return b.ReadRune()
	})

	fl.Clear()

	buf := &bytes.Buffer{}

	_, err = buf.ReadFrom(fl)
	require.NoError(t, err)

	t.Logf("jsonc:\n\n##\n%s\n##\n", buf.Bytes())
}

func TestMultilineValue(t *testing.T) {

	file, err := os.Open(`multiline-test.jsonc`)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(file)

	ring, err := NewRing(256, 64, nil)
	require.NoError(t, err)

	f := NewFilter(ring, 256, false, " ")

	b := strings.NewReader(string(data))
	ring.Clear(func() (r rune, size int, err error) {
		return b.ReadRune()
	})

	f.Clear()

	buf := &bytes.Buffer{}

	_, err = buf.ReadFrom(f)
	require.NoError(t, err)

	ret := buf.String()
	assert.Contains(t, ret, `"quote":"\""`)
	assert.Contains(t, ret, `"solidus":"\/"`)
	assert.Contains(t, ret, `"fromFeed":"\f"`)
	assert.Contains(t, ret, `"lineFeed":"\n"`)
	assert.Contains(t, ret, `"carriageReturn":"\r"`)
	assert.Contains(t, ret, `"tab":"\t"`)
}
