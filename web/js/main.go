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
	Fmt       = `input#fmt`
	Clear     = `input#clear`
	JsoncArea = `textarea#edit`
	JsonArea  = `div#editjson`
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

	jsonc, err := process(false)
	if err != nil {
		panic(err)
	}

	jQuery(JsoncArea).SetVal(jsonc)

	json, err := process(true)
	if err != nil {
		panic(err)
	}

	//b := PrettyJson([]byte(json))
	json = strings.Replace(json, "\n", `<br/>`, -1)
	jQuery(JsonArea).SetHtml(json)

	js.Global.Call("initTextArea")
}

func process(minimize bool) (json string, err error) {

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
