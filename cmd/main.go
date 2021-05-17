package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/komkom/jsonc/jsonc"
)

var (
	minimize = kingpin.Flag(`minimize`, `transform to minified json`).Short('m').Bool()
)

func main() {

	f, err := jsonc.New(os.Stdin, *minimize, " ")
	if err != nil {
		fmt.Printf("no input stream, error: %v", err)
		os.Exit(1)
	}

	io.Copy(os.Stdout, f)

	if !f.Done() || (f.Err() != nil && f.Err() != io.EOF) {
		os.Exit(1)
	}
}
