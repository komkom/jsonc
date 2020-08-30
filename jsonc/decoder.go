package jsonc

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
)

type Decoder struct {
	r      io.Reader
	log    *log.Logger
	filter *Filter
}

type Options = func(dec Decoder)

func NewDecoder(r io.Reader, opts ...Options) (*Decoder, error) {
	dec := Decoder{r: r}

	buf := NewBuffer(r, 256, 64)

	ring, err := NewRing(256, 64, func() (r rune, size int, err error) {
		return buf.ReadRune()
	})
	if err != nil {
		return nil, err
	}

	dec.filter = NewFilter(ring, 256, false, ``)
	return &dec, nil
}

func (d *Decoder) Decode(v interface{}) error {

	data, err := ioutil.ReadAll(d.filter)
	if err != nil {
		return errors.Wrap(err, `jsonc filter failed`)
	}
	return json.Unmarshal(data, v)
}
