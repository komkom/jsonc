package jsonc

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type Decoder struct {
	filter *Filter
}

type Options = func(dec Decoder)

func NewDecoder(r io.RuneReader, opts ...Options) (*Decoder, error) {

	ring, err := NewRing(256, 64, r.ReadRune)
	if err != nil {
		return nil, err
	}

	return &Decoder{filter: NewFilter(ring, 256, false, ``)}, nil
}

func (d *Decoder) Decode(v interface{}) error {

	data, err := ioutil.ReadAll(d.filter)
	if err != nil {
		return errors.Wrap(err, `jsonc filter failed`)
	}
	return json.Unmarshal(data, v)
}
