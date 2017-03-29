package main

import (
	"bytes"
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
		process(true)
	})

	jQuery(Fmt).On(jquery.CLICK, func(e jquery.Event) {
		process(false)
	})
}

func process(minimize bool) {

	edit := jQuery(TextArea).Html()
	print(`____i ` + edit)

	edit = strings.Replace(edit, `<br>`, "\n", -1)

	edit = strings.Replace(edit, `<div>\n</div>`, "\n", -1)

	edit = strings.Replace(edit, `&nbsp;`, ` `, -1)

	edit = strings.Replace(edit, `<div>`, "\n", -1)

	edit = strings.Replace(edit, `</div>`, "\n", -1)

	print(edit)

	r := strings.NewReader(edit)

	jcr, err := reader.New(r, minimize, "&nbsp;")
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

	json := string(buf.Bytes())

	i //print(json)

	json = strings.Replace(json, "\n", `<br/>`, -1)

	//print(json)
	jQuery(TextArea).SetHtml(json)
}
