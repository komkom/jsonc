package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errReader struct {
	err        error
	middleRune byte
}

func (r errReader) Read(p []byte) (n int, err error) {
	if len(p) <= 3 {
		panic(`errReader buffer invalid`)
	}

	p[0] = byte('a')
	p[1] = r.middleRune
	p[2] = byte('c')

	return 3, r.err
}

func TestRuneReader_withErrReader(t *testing.T) {

	r := errReader{err: io.EOF, middleRune: byte('b')}
	rur := &RuneReader{reader: r}

	// test first EOF path
	var runes []rune
	for {
		ru, _, err := rur.ReadRune()
		if errors.Is(err, io.EOF) {
			assert.True(t, strings.Contains(err.Error(), `runeReader.ReadRune EOF (1)`))
			break
		}

		require.NoError(t, err)
		runes = append(runes, ru)
	}
	assert.Equal(t, `abc`, string(runes))

	// test first invalid EOF path
	r = errReader{err: io.EOF, middleRune: byte('\255')}
	rur = &RuneReader{reader: r}

	runes = nil
	for {
		ru, _, err := rur.ReadRune()
		if err != nil {
			assert.Equal(t, err.Error(), `runeReader.ReadRune invalid EOF (1)`)
			return
		}

		require.NoError(t, err)
		runes = append(runes, ru)
	}
}

func TestRuneReader_withBufferString(t *testing.T) {

	tests := []struct {
		data []byte
		err  string
	}{
		{data: []byte(`test this`)},
		{data: []byte(``)},
		{
			data: []byte{byte('\255'), byte('a')},
			err:  `runeReader.ReadRune invalid rune`,
		},
		{
			data: []byte{byte('a'), byte('\255'), byte('b')},
			err:  `runeReader.ReadRune invalid EOF (2)`,
		},
		{
			data: append([]byte{byte('a'), byte('\255')}, messageWith(128)...),
			err:  `runeReader.ReadRune invalid rune`,
		},
		{
			data: messageWith(131),
		},
	}

	for _, ts := range tests {

		buf := bytes.NewBuffer(ts.data)
		rur := &RuneReader{reader: buf}

		var runes []rune
		var err error
		var ru rune
		for {
			ru, _, err = rur.ReadRune()
			if errors.Is(err, io.EOF) {
				assert.True(t, strings.Contains(err.Error(), `runeReader.ReadRune EOF (2)`))
				err = nil
				break
			}

			if err != nil {
				break
			}

			require.NoError(t, err)

			runes = append(runes, ru)
		}

		if ts.err != `` {
			require.Error(t, err)
			assert.True(t, strings.Contains(err.Error(), ts.err), err.Error())
			continue
		}
		require.NoError(t, err)

		resp := string(runes)
		assert.Equal(t, string(ts.data), resp)
	}
}

func messageWith(numberOfBytes int) []byte {
	var data []byte
	for i := 0; i < numberOfBytes; i++ {
		data = append(data, byte('a'))
	}
	return data
}
