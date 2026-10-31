package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/komkom/jsonc/jsonc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInOut(t *testing.T) {

	file, err := os.Open(`test-inout.txt`)
	require.NoError(t, err)
	raw, err := io.ReadAll(file)
	require.NoError(t, err)
	data := string(raw)

	// remove everything with ## end ##
	withEnd := strings.Split(data, `## end ##`)
	data = withEnd[0]
	split := strings.Split(data, "###\n")

	for _, sp := range split {

		inOut := strings.Split(sp, "##\n")
		in := inOut[0]
		out := inOut[1]
		out = strings.TrimSuffix(out, "\n")

		b := strings.NewReader(string(in))

		f, err := jsonc.New(b, true, " ")
		require.NoError(t, err)

		var res bytes.Buffer

		_, err = io.Copy(&res, f)
		require.NoError(t, err)

		assert.Equal(t, out, res.String())
	}
}
