package jsonc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DecoderTestData struct {
	jsonc            string
	newUnmarshalType func() interface{}
	json             string
}

func testDecoderData() []DecoderTestData {

	return []DecoderTestData{

		{
			jsonc: `{x:x,}`,
			newUnmarshalType: func() interface{} {

				return &struct {
					X string `json:"x"`
				}{}
			},
			json: `{"x":"x"}`,
		},
		{
			jsonc: `{x:x/*comment*/}`,
			newUnmarshalType: func() interface{} {

				return &struct {
					X string `json:"x"`
				}{}
			},
			json: `{"x":"x"}`,
		},
		{
			jsonc: `{x:x/*comment*/,
			  y: [1,2,3,4,5],
			  // an other comment
			}`,
			newUnmarshalType: func() interface{} {

				return &struct {
					X string `json:"x"`
					Y []int  `json:"y"`
				}{}
			},
			json: `{"x":"x","y":[1,2,3,4,5]}`,
		},
		{
			jsonc: `[1,2,3,4,5]`,
			newUnmarshalType: func() interface{} {
				return &[]int{}
			},
			json: `[1,2,3,4,5]`,
		},
	}
}

func TestJsoncDecoder(t *testing.T) {

	tests := testDecoderData()

	for _, ts := range tests {

		dec, err := NewDecoder(strings.NewReader(ts.jsonc))
		require.NoError(t, err)

		ut := ts.newUnmarshalType()

		err = dec.Decode(ut)
		require.NoError(t, err)

		data, err := json.Marshal(ut)
		require.NoError(t, err)

		assert.Equal(t, ts.json, string(data))
	}
}
