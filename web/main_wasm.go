package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/komkom/jsonc/jsonc"
)

var Document = js.Global().Get("document")

const (
	Fmt       = `div#format-button`
	Clear     = `input#clear`
	JsoncArea = `edit`
	JsonArea  = `editjson`
	ErrorMsg  = `errormsg`
)

func main() {

	js.Global().Set(`format`, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("format\n")
		load()
		return nil
	}))

	<-make(chan struct{})
}

func load() {

	errMsg := Document.Call("getElementById", ErrorMsg)
	errMsg.Set(`innerHTML`, `testtest`)
	style := errMsg.Get(`style`)
	style.Set(`display`, `none`)

	jsonc, errpos, err := process(false)
	if err != nil {

		var edit string
		val := Document.Call("getElementById", JsoncArea).Get(`value`)
		if val.Truthy() {
			edit = val.String()
		}

		// get the line of the error
		errline := strings.Count(edit[:errpos], "\n")
		fmt.Printf("errline %v\n", errline)
		idx := strings.Index(edit[errpos:], "\n")
		style.Set(`display`, `block`)

		line := strconv.Itoa(idx + 1)
		pos := strconv.Itoa(errpos)

		if idx == -1 {
			errMsg.Set(`innerHTML`, `error line: 1 pos: `+pos)
		} else {
			errMsg.Set(`innerHTML`, `error line: `+line+` pos: `+pos)
		}

		return
	}

	Document.Call("getElementById", JsoncArea).Set(`value`, jsonc)

	json, _, err := process(true)
	if err != nil {
		panic(err)
	}

	pj, err := PrettyJson([]byte(json))
	if err != nil {
		style.Set(`display`, `block`)
		errMsg.Set(`innerHTML`, `error: `+err.Error())
		return
	}

	json = strings.Replace(pj, "\n", `<br/>`, -1)

	Document.Call("getElementById", JsonArea).Set(`innerHTML`, json)
}

func process(minimize bool) (json string, errpos int, err error) {

	var edit string
	val := Document.Call("getElementById", JsoncArea).Get(`value`)
	if val.Truthy() {
		edit = val.String()
	}

	r := strings.NewReader(edit)

	jcr, err := jsonc.New(r, minimize, "  ")
	if err != nil {
		print("error 1")
		return
	}

	buf := &bytes.Buffer{}

	io.Copy(buf, jcr)

	if jcr.Err() != nil && !errors.Is(jcr.Err(), io.EOF) {
		err = jcr.Err()

		var rerr jsonc.Error
		if ok := errors.As(err, &rerr); ok {
			errpos = rerr.Position()
		}

		print("error 2 " + err.Error())
		return
	}

	print(`success`)

	json = string(buf.Bytes())

	return
}

func PrettyJson(jsn []byte) (string, error) {

	var pretty bytes.Buffer
	err := json.Indent(&pretty, jsn, "", "&nbsp;&nbsp;&nbsp;")
	if err != nil {
		return ``, err
	}

	return string(pretty.Bytes()), nil
}
