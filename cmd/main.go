package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/komkom/jsonc/jsonc"
)

var (
	minimize = kingpin.Flag(`minimize`, `transform to minified json`).Short('m').Bool()
	path     = kingpin.Flag(`path`, `a jsonc file or a directory containing jsonc to process`).Short('p').String()
)

func moveFile(to string, from string) error {

	fromF, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromF.Close()

	toF, err := os.Create(to)
	if err != nil {
		return err
	}
	defer toF.Close()

	_, err = io.Copy(toF, fromF)
	if err != nil {
		return err
	}

	return os.Remove(from)
}

func process(dir string, files []os.FileInfo, minimize bool) error {

	for _, fi := range files {

		if strings.HasSuffix(fi.Name(), `.jsonc`) {

			name := strings.Split(fi.Name(), `.jsonc`)[0]
			var w io.Writer
			var r io.Reader
			var jcr io.Reader

			f, err := os.Open(dir + `/` + fi.Name())
			if err != nil {
				return err
			}
			defer f.Close()
			r = f

			fending := `.tmp`
			if minimize {
				fending = `.json`

			}

			f, err = os.Create(dir + `/` + name + fending)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f

			jcr, err = jsonc.New(r, minimize, " ")
			if err != nil {
				return err
			}

			io.Copy(w, jcr)

			if !minimize {
				err := moveFile(dir+`/`+name+`.jsonc`, dir+`/`+name+fending)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func main() {
	kingpin.Parse()

	if *path != `` {

		check := func(err error) {

			if err != nil {
				fmt.Printf("path: %v error: %v", *path, err)
				os.Exit(1)
			}
		}

		fi, err := os.Stat(*path)
		check(err)

		dir, err := filepath.Abs(*path)
		check(err)

		if fi.Mode().IsRegular() {
			dir = filepath.Dir(dir)
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():

			files, err := ioutil.ReadDir(*path)
			check(err)

			err = process(dir, files, *minimize)
			check(err)

		case mode.IsRegular():

			process(dir, []os.FileInfo{fi}, *minimize)
			check(err)

			break
		}

	} else {

		r, err := jsonc.New(os.Stdin, *minimize, " ")
		if err != nil {
			fmt.Printf("no input stream, error: %v", err)
			os.Exit(1)
		}

		io.Copy(os.Stdout, r)
	}
}
