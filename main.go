package main

import (
	"flag"
	"io"
	"os"

	"github.com/komkom/jsonc/reader"
)

var minimize bool

func init() {
	flag.BoolVar(&minimize, `m`, false, `minimize a jsonc stream`)
}

func main() {
	flag.Parse()

	r := reader.New(os.Stdin, minimize)

	io.Copy(os.Stdout, r)

	/*
		if err = r.Body.Close(); err != nil {
			fmt.Println(err)
		}
	*/

	/*
		fmt.Printf("__ %v\n", r)

		for _, a := range os.Args[1:] {
			fmt.Printf("__ %v\n", a)

		}
	*/
}
