package config

import (
	"bytes"
	"io"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/transform"
)

func TestEscapeVariablesQuick(t *testing.T) {
	transformer := newExpandTransformer(func(k string) (string, bool) {
		return "expanded", true
	})
	// Round-tripping any string through escaping, then expansion should return
	// the original unchanged.
	round := func(s string) bool {
		escaped := escapeVariables([]byte(s))
		r := transform.NewReader(bytes.NewReader(escaped), transformer)
		unescaped, err := io.ReadAll(r)
		require.NoError(t, err, "failed to read expanded config")
		return s == string(unescaped)
	}
	if err := quick.Check(round, nil); err != nil {
		t.Error(err)
	}
}
