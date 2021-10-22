package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/komkom/jsonc/jsonc"
)

var ErrInvalidRune = fmt.Errorf(`invalid-rune`)

func NewRuneReader(reader io.Reader) *RuneReader {
	return &RuneReader{reader: reader}
}

type RuneReader struct {
	reader io.Reader
	isEOF  bool
	buf    [4]byte
	length int
}

func (r *RuneReader) decodeRune() (rune, int, error) {

	ru, size := utf8.DecodeRune(r.buf[:r.length])

	if ru == utf8.RuneError {
		return '0', 0, ErrInvalidRune
	}

	copy(r.buf[:], r.buf[size:r.length])
	r.length -= size

	return ru, size, nil
}

func (r *RuneReader) ReadRune() (rune, int, error) {

	if r.isEOF {
		if r.length != 0 {
			return r.decodeRune()
		}

		return '0', 0, io.EOF
	}

	n, err := r.reader.Read(r.buf[r.length:])
	if err != nil && !errors.Is(err, io.EOF) {
		return '0', 0, fmt.Errorf(`runeReader.ReadRune reader.Read failed %w`, err)
	}

	if errors.Is(err, io.EOF) {
		r.isEOF = true
	}

	r.length += n

	if r.length == 0 && errors.Is(err, io.EOF) {
		return '0', 0, err
	}

	return r.decodeRune()
}

func main() {

	var minimize bool
	flag.BoolVar(&minimize, "m", false, `transform to minified json`)
	flag.Parse()

	f, err := jsonc.New(NewRuneReader(os.Stdin), minimize, " ")
	if err != nil {
		fmt.Printf("no input stream, error: %v", err)
		os.Exit(1)
	}

	io.Copy(os.Stdout, f)

	if !f.Done() || (f.Err() != nil && f.Err() != io.EOF) {
		os.Exit(1)
	}
}
