package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/gopherjs/jquery"
	"github.com/komkom/jsonc/reader"
)

//convenience:
var jQuery = jquery.NewJQuery

const (
	Fmt      = `input#fmt`
	Json     = `input#json`
	TextArea = `textarea#edit`
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

	edit := jQuery(TextArea).Val()

	r := strings.NewReader(edit)

	jcr, err := reader.New(r, minimize)
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

	json := string(buf.Bytes())

	jQuery(TextArea).SetVal(json)
}
