package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/komkom/jsonc/jsonc"
	"github.com/pkg/errors"
)

type Rune struct {
	R    rune
	Size int
}

type RuneReader struct {
	reader  io.Reader
	isEOF   bool
	buf     [64]byte
	bufLen  int
	runeBuf []Rune
}

func (r *RuneReader) popRune() (Rune, bool) {

	if len(r.runeBuf) == 0 {
		return Rune{}, false
	}

	ru := r.runeBuf[0]
	r.runeBuf = r.runeBuf[1:]
	return ru, true
}

func (r *RuneReader) ReadRune() (rune, int, error) {

	if ru, ok := r.popRune(); ok {
		return ru.R, ru.Size, nil
	}

	if r.isEOF {
		if r.bufLen != 0 {
			return '0', 0, fmt.Errorf(`runeReader.ReadRune invalid EOF (1)`)
		}
		return '0', 0, fmt.Errorf(`runeReader.ReadRune EOF (1) %w`, io.EOF)
	}

	n, err := r.reader.Read(r.buf[r.bufLen:])

	if err != nil && !errors.Is(err, io.EOF) {
		return '0', 0, fmt.Errorf(`runeReader.ReadRune reader.Read failed %w`, err)
	}

	if n == 0 && err != nil {
		if r.bufLen != 0 {
			return '0', 0, fmt.Errorf(`runeReader.ReadRune invalid EOF (2)`)
		}
		return '0', 0, fmt.Errorf(`runeReader.ReadRune EOF (2) %w`, err)
	}

	if n == 0 {
		panic(`runeReader.ReadRune reader.Read unexpected error`)
	}

	if errors.Is(err, io.EOF) {
		r.isEOF = true
	}

	r.bufLen += n
	var offset int
	for {
		ru, size := utf8.DecodeRune(r.buf[offset:r.bufLen])

		if ru == utf8.RuneError {
			break
		}

		r.runeBuf = append(r.runeBuf, Rune{R: ru, Size: size})
		offset += size
	}

	// memcopy & set length
	copy(r.buf[:], r.buf[offset:r.bufLen])
	r.bufLen = r.bufLen - offset

	ru, ok := r.popRune()
	if !ok {
		return '0', 0, fmt.Errorf(`runeReader.ReadRune invalid rune`)
	}

	return ru.R, ru.Size, nil
}

func main() {

	var minimize bool
	flag.BoolVar(&minimize, "m", false, `transform to minified json`)
	flag.Parse()

	f, err := jsonc.New(&RuneReader{reader: os.Stdin}, minimize, " ")
	if err != nil {
		fmt.Printf("no input stream, error: %v", err)
		os.Exit(1)
	}

	io.Copy(os.Stdout, f)

	if !f.Done() || (f.Err() != nil && f.Err() != io.EOF) {
		os.Exit(1)
	}
}
