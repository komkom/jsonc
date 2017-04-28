package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/jquery"
	"github.com/komkom/jsonc/reader"
)

var jQuery = jquery.NewJQuery

const (
	Fmt       = `div#format-button`
	Clear     = `input#clear`
	JsoncArea = `textarea#edit`
	JsonArea  = `pre#editjson`
	ErrorMsg  = `div#errormsg`
)

func main() {

	jQuery(JsoncArea).On(jquery.FOCUSOUT, func(e jquery.Event) {
		load()
	})

	jQuery(Fmt).On(jquery.CLICK, func(e jquery.Event) {
		load()
	})
}

func load() {

	jQuery(ErrorMsg).SetText(``)

	// hide error
	jQuery(ErrorMsg).Hide()

	jsonc, errpos, err := process(false)
	if err != nil {

		edit := jQuery(JsoncArea).Val()

		// get the line of the error
		errline := strings.Count(edit[:errpos], "\n")

		idx := strings.Index(edit[errpos:], "\n")

		// show error
		jQuery(ErrorMsg).Show()

		if idx == -1 {
			jQuery(ErrorMsg).SetText(`error parsing: ` + edit[errpos:])
		} else {
			jQuery(ErrorMsg).SetText(`error parsing: ` + edit[errpos:errpos+idx])
		}

		js.Global.Call("selectEditorLine", errline)
		return
	}

	jQuery(JsoncArea).SetVal(jsonc)

	json, _, err := process(true)
	if err != nil {
		panic(err)
	}

	pj := PrettyJson([]byte(json))

	json = strings.Replace(string(pj), "\n", `<br/>`, -1)
	jQuery(JsonArea).SetHtml(json)

	js.Global.Call("initTextArea")
}

func process(minimize bool) (json string, errpos int, err error) {

	edit := jQuery(JsoncArea).Val()

	r := strings.NewReader(edit)

	jcr, err := reader.New(r, minimize, "  ")
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
		errpos = jcrErr.Position()
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
