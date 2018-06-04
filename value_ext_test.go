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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValueIntegration(t *testing.T) {
	// A simple configuration with a single key and a scalar value is sufficient
	// to expose critical bugs here.
	provider, err := NewYAMLProviderFromBytes([]byte("scalar: foo"))
	require.NoError(t, err, "couldn't create static provider")

	populated := func(t testing.TB, v Value) string {
		var s string
		require.NoError(t, v.Populate(&s), "couldn't populate string")
		return s
	}

	defaulted := func(t testing.TB, v Value) string {
		val, err := v.WithDefault("quux")
		require.NoError(t, err, "couldn't set default")
		return populated(t, val)
	}

	t.Run("arguments are internally consistent", func(t *testing.T) {
		// All arguments are internally consistent: value and found match the
		// contents of provider at key.
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			true,     // found
		)

		// Only need to test these once, since it's not critical.
		assert.Equal(t, provider.Name(), v.Source(), "value source should be provider name")
		assert.Equal(t, "foo", v.String(), "unexpected fmt.Stringer implementation")

		assert.Equal(t, "foo", v.Value(), "unexpected Value")
		assert.Equal(t, "foo", populated(t, v), "unexpected Populate")
		assert.Equal(t, "foo", v.Get(Root).Value(), "unexpected Value after Get")
		assert.Equal(t, "foo", defaulted(t, v), "unexpected Value after WithDefault")
		assert.True(t, v.HasValue(), "unexpected HasValue")
	})

	t.Run("value doesn't match provider", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"baz",    // value, doesn't match provider
			true,     // found
		)
		assert.Equal(t, "baz", v.Value(), "unexpected Value")                         // great, using supplied value
		assert.Equal(t, "foo", populated(t, v), "unexpected Populate")                // WAT
		assert.Equal(t, "foo", v.Get(Root).Value(), "unexpected Value after Get")     // WAT
		assert.Equal(t, "foo", defaulted(t, v), "unexpected Value after WithDefault") // WAT
		assert.True(t, v.HasValue(), "unexpected HasValue")
	})

	t.Run("found doesn't match provider", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			false,    // found, doesn't match provider
		)
		assert.Equal(t, "foo", v.Value(), "unexpected Value")
		assert.Equal(t, "foo", populated(t, v), "unexpected Populate")
		assert.Equal(t, "foo", v.Get(Root).Value(), "unexpected Value after Get")
		assert.Equal(t, "foo", defaulted(t, v), "unexpected Value after WithDefault")
		assert.False(t, v.HasValue(), "unexpected HasValue") // WAT
	})
}
