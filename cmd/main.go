package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/komkom/jsonc/jsonc"
)

func main() {

	var minimize bool
	flag.BoolVar(&minimize, "m", false, `transform to minified json`)
	flag.Parse()

	f, err := jsonc.New(os.Stdin, minimize, " ")
	if err != nil {
		fmt.Printf("no input stream, error: %v", err)
		os.Exit(1)
	}

	io.Copy(os.Stdout, f)

	if !f.Done() || (f.Err() != nil && f.Err() != io.EOF) {
		os.Exit(1)
	}
}
