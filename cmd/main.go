package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/komkom/jsonc/jsonc"
)

func main() {

	var minimize bool
	flag.BoolVar(&minimize, "m", false, `transform to minified json`)
	flag.Parse()

	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("read all failed, error: %v", err)
		os.Exit(1)
	}

	b := strings.NewReader(string(in))

	f, err := jsonc.New(b, minimize, " ")
	if err != nil {
		fmt.Printf("no input stream, error: %v", err)
		os.Exit(1)
	}

	io.Copy(os.Stdout, f)

	if !f.Done() || (f.Err() != nil && f.Err() != io.EOF) {
		os.Exit(1)
	}
}
