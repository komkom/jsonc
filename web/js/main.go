package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gopherjs/jquery"
	"github.com/komkom/jsonc/reader"
)

var jQuery = jquery.NewJQuery

const (
	Fmt      = `input#fmt`
	Json     = `input#json`
	TextArea = `div#edit`
)

func main() {

	//show jQuery Version on console:
	print("22 Your current jQuery version is: " + jQuery().Jquery)

	jQuery(Json).On(jquery.CLICK, func(e jquery.Event) {
		json, err := process(true)
		if err != nil {
			panic(err)
		}

		b := PrettyJson([]byte(json))

		jQuery(TextArea).SetHtml(string(b))
	})

	jQuery(Fmt).On(jquery.CLICK, func(e jquery.Event) {
		json, err := process(false)
		if err != nil {
			panic(err)
		}

		jQuery(TextArea).SetHtml(json)
	})
}

func process(minimize bool) (json string, err error) {

	edit := jQuery(TextArea).Html()
	print(`____i ` + edit)

	edit = strings.Replace(edit, `<br>`, "\n", -1)

	edit = strings.Replace(edit, `<div>\n</div>`, "\n", -1)

	edit = strings.Replace(edit, `&nbsp;`, ` `, -1)

	edit = strings.Replace(edit, `<div>`, "\n", -1)

	print(`____o ` + edit)

	edit = strings.Replace(edit, `</div>`, "", -1)

	print(edit)

	r := strings.NewReader(edit)

	jcr, err := reader.New(r, minimize, "&nbsp;&nbsp;")
	if err != nil {
		print("error 1")
		return
	}

	buf := &bytes.Buffer{}

	io.Copy(buf, jcr)

	f := jcr.(*reader.Filter)
	jcrErr := f.Error()
	if jcrErr != nil && jcrErr.Err() != io.EOF {
		err = jcrErr.Err()
		print("error 2 " + err.Error())
		return
	}

	print(`success`)

	json = string(buf.Bytes())

	//print(json)

	json = strings.Replace(json, "\n", `<br/>`, -1)

	return
	//print(json)
}

func PrettyJson(jsn []byte) (prettyJson []byte) {

	var pretty bytes.Buffer
	err := json.Indent(&pretty, jsn, "&nbsp;&nbsp;", "&nbsp;")
	if err != nil {
		return
	}

	prettyJson = pretty.Bytes()
	return
}
