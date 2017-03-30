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
	Fmt       = `input#fmt`
	Clear     = `input#clear`
	JsoncArea = `div#edit`
	JsonArea  = `div#editjson`
)

func main() {

	//show jQuery Version on console:
	print("22 Your current jQuery version is: " + jQuery().Jquery)

	jQuery(Clear).On(jquery.CLICK, func(e jquery.Event) {

		jQuery(JsoncArea).SetHtml(``)
		jQuery(JsonArea).SetHtml(``)
	})

	jQuery(JsoncArea).On(jquery.FOCUSOUT, func(e jquery.Event) {
		load()
	})

	jQuery(Fmt).On(jquery.CLICK, func(e jquery.Event) {
		load()
	})
}

func load() {

	jsonc, err := process(false)
	if err != nil {
		panic(err)
	}

	jsonc = strings.Replace(jsonc, "\n", `<br/>`, -1)
	jQuery(JsoncArea).SetHtml(jsonc)

	json, err := process(true)
	if err != nil {
		panic(err)
	}

	b := PrettyJson([]byte(json))
	json = strings.Replace(string(b), "\n", `<br/>`, -1)
	jQuery(JsonArea).SetHtml(json)

}

func process(minimize bool) (json string, err error) {

	edit := jQuery(JsoncArea).Html()
	print(`____i ` + edit)

	edit = strings.Replace(edit, `<br>`, "\n", -1)

	edit = strings.Replace(edit, `<div>\n</div>`, "\n", -1)

	edit = strings.Replace(edit, `&nbsp;`, ` `, -1)

	edit = strings.Replace(edit, `<div>`, "\n", -1)

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

	return
}

func PrettyJson(jsn []byte) (prettyJson []byte) {

	var pretty bytes.Buffer
	err := json.Indent(&pretty, jsn, "", "&nbsp;&nbsp;&nbsp;")
	if err != nil {
		return
	}

	prettyJson = pretty.Bytes()
	return
}
