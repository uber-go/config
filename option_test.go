// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatic(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p, err := NewYAML(Static("foo"))
		require.NoError(t, err, "couldn't construct provider")
		assert.Equal(t, "foo", p.Get(Root).Value(), "unexpected value")
	})
	t.Run("error", func(t *testing.T) {
		_, err := NewYAML(Static(noYAML{}))
		require.Error(t, err, "expected error constructing provider")
	})
}

func TestExpand(t *testing.T) {
	environment := map[string]string{"FOO": "bar"}
	lookup := func(key string) (string, bool) {
		s, ok := environment[key]
		return s, ok
	}
	tests := []struct {
		desc      string
		template  string
		expectErr bool
		expectVal interface{}
	}{
		{"present unbracketed", "$FOO", false, "bar"},
		{"absent unbracketed", "$NOT_THERE", true, ""},
		{"present bracketed", "${FOO:baz}", false, "bar"},
		{"absent bracketed", "${NOT_THERE:baz}", false, "baz"},
		{"absent bracketed no default", `${NOT_THERE:""}`, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			yaml := strings.NewReader(fmt.Sprintf("key: %s", tt.template))
			p, err := NewYAML(Source(yaml), Expand(lookup))
			if tt.expectErr {
				require.Error(t, err, "expected provider construction to fail")
				return
			}
			require.NoError(t, err, "couldn't construct provider")
			assert.Equal(t, tt.expectVal, p.Get("key").Value(), "unexpected value")
		})
	}
}

func TestExpandIgnoresComments(t *testing.T) {
	// Regression test for https://github.com/uber-go/config/issues/80.
	lookup := func(_ string) (string, bool) { return "", false }
	p, err := NewYAML(
		Source(strings.NewReader("# $FOO is a foo.\n"+`key: ${FOO:""}`)),
		Expand(lookup),
	)
	require.NoError(t, err, "should ignore env vars in comments")
	assert.Equal(t, nil, p.Get("key").Value(), "should expand env vars elsewhere")
}

func TestName(t *testing.T) {
	const name = "hello"
	p, err := NewYAML(Name(name))
	require.NoError(t, err, "couldn't construct provider")
	assert.Equal(t, name, p.Name(), "unexpected provider name")
	assert.Equal(t, name, p.Get(Root).Source(), "unexpected value source")
}

func TestSourceErrors(t *testing.T) {
	f, err := os.CreateTemp("" /* dir */, "test-source-errors" /* prefix */)
	require.NoError(t, err, "couldn't create temporary file")
	// Make reads fail by deleting file.
	require.NoError(t, f.Close(), "couldn't delete temporary file")
	require.NoError(t, os.Remove(f.Name()), "couldn't delete temporary file")

	t.Run("source", func(t *testing.T) {
		_, err = NewYAML(Source(f))
		require.Error(t, err)
	})

	t.Run("file", func(t *testing.T) {
		_, err = NewYAML(File(f.Name()))
		require.Error(t, err)
	})
}
