// Copyright (c) 2017-2018 Uber Technologies, Inc.
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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValueParameterValidation(t *testing.T) {
	// The 1.0 release of this package included a NewValue implementation that
	// had many, many bugs. To fix those bugs, we now validate the internal
	// consistency of all arguments to NewValue at construction time. Because we
	// can't change the signature of NewValue without breaking backward
	// compatibility, the function now panics if the user-supplied parameters
	// aren't sensible.
	provider, err := NewYAML(Source(strings.NewReader("scalar: foo")))
	require.NoError(t, err, "couldn't create provider")

	t.Run("arguments are internally consistent", func(t *testing.T) {
		// All arguments are internally consistent: value and found match the
		// contents of provider at key.
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			true,     // found
		)

		assert.Equal(t, provider.Name(), v.Source(), "value source should be provider name")
		assert.Equal(t, "foo", v.String(), "unexpected fmt.Stringer implementation")
		assert.Equal(t, "foo", v.Value(), "unexpected Value")
		assert.True(t, v.HasValue(), "unexpected HasValue")
		assert.Equal(t, "foo", v.Get(Root).Value(), "unexpected Value after Get")

		var s string
		require.NoError(t, v.Populate(&s), "couldn't populate string")
		assert.Equal(t, "foo", s, "unexpected Populate")

		defaulted, err := v.WithDefault("bar")
		require.NoError(t, err, "couldn't set default")
		assert.Equal(t, "foo", defaulted.Value(), "unexpected Value after WithDefault")
	})

	t.Run("value doesn't match provider", func(t *testing.T) {
		assert.Panics(t, func() {
			NewValue(
				provider,
				"scalar", // key
				"baz",    // value, doesn't match provider
				true,     // found
			)
		})
	})

	t.Run("found doesn't match provider", func(t *testing.T) {
		assert.Panics(t, func() {
			NewValue(
				provider,
				"scalar", // key
				"foo",    // value
				false,    // found, doesn't match provider
			)
		})
		assert.Panics(t, func() {
			NewValue(
				provider,
				"not_present", // key
				nil,           // value
				true,          // found, doesn't match provider
			)
		})
	})

	t.Run("value isn't serializable as YAML", func(t *testing.T) {
		assert.Panics(t, func() {
			NewValue(
				provider,
				"scalar", // key
				noYAML{}, // value, can't be serialized to YAML
				true,     // found
			)
		})
	})
}

func TestUnserializableDefault(t *testing.T) {
	p := NopProvider{}
	_, err := p.Get(Root).WithDefault(noYAML{})
	require.Error(t, err, "expected setting default to fail")
}

func TestScopedProvider(t *testing.T) {
	p, err := NewYAML(Source(strings.NewReader("foo: {bar: baz}")))
	require.NoError(t, err, "couldn't construct provider")

	t.Run("prefix", func(t *testing.T) {
		s := NewScopedProvider("foo", p)
		assert.Equal(t, "baz", s.Get("bar").Value(), "unexpected value")
	})

	t.Run("no prefix", func(t *testing.T) {
		s := NewScopedProvider("", p)
		assert.Equal(t, "baz", s.Get("foo.bar").Value(), "unexpected value")
	})
}

func TestProviderGroup(t *testing.T) {
	first, err := NewYAML(Source(strings.NewReader("key: {foo: bar}")))
	require.NoError(t, err, "couldn't construct first provider")
	second, err := NewYAML(Source(strings.NewReader("key: {baz: quux}")))
	require.NoError(t, err, "couldn't construct second provider")

	p, err := NewProviderGroup("group", first, second)
	require.NoError(t, err, "couldn't group providers")
	assert.Equal(t, "group", p.Name(), "unexpected name")

	var cfg map[string]string
	require.NoError(t, p.Get("key").Populate(&cfg), "couldn't populate map")
	assert.Equal(t, map[string]string{
		"foo": "bar",
		"baz": "quux",
	}, cfg, "expected to deep-merge providers")
}

func TestSingleProviders(t *testing.T) {
	environment := map[string]string{"FOO": "bar"}
	lookup := func(key string) (string, bool) {
		s, ok := environment[key]
		return s, ok
	}
	run := func(t testing.TB, p Provider, err error) {
		require.NoError(t, err, "couldn't construct provider")
		assert.Equal(t, "bar", p.Get("foo").Value(), "unexpected value")
	}

	t.Run("expanded static", func(t *testing.T) {
		p, err := NewStaticProviderWithExpand(map[string]string{
			"foo": "$FOO",
		}, lookup)
		run(t, p, err)
	})

	t.Run("static", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]string{
			"foo": "bar",
		})
		run(t, p, err)
	})

	t.Run("files present", func(t *testing.T) {
		p, err := NewYAMLProviderWithExpand(lookup, "testdata/config.yaml")
		run(t, p, err)
	})

	t.Run("files missing", func(t *testing.T) {
		_, err := NewYAMLProviderFromFiles("testdata/not_there.yaml")
		require.Error(t, err, "expected error reading nonexistent file")
		assert.Contains(t, err.Error(), "no such file or directory", "unexpected error message")
	})

	t.Run("bytes", func(t *testing.T) {
		p, err := NewYAMLProviderFromBytes([]byte("foo: bar"))
		run(t, p, err)
	})
}
