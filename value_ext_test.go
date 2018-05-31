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
	provider, err := NewStaticProvider(map[string]interface{}{
		"scalar":   "foo",
		"mapping":  map[string]int{"one": 1, "two": 2},
		"sequence": []int{1, 2},
	})
	require.NoError(t, err, "couldn't create static provider")

	t.Run("source is provider name", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			true,     // found
		)
		assert.Equal(t, provider.Name(), v.Source(), "value source should be provider name")
	})

	t.Run("value can override provider", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"quux",   // value, doesn't match provider contents
			true,     // found
		)
		assert.Equal(t, "quux", v.String(), "unexpected string representation")
		assert.Equal(t, "quux", v.Value(), "unexpected value")
	})

	t.Run("get re-exposes provider", func(t *testing.T) {
		v := NewValue(
			provider,
			Root,    // key
			"hello", // value, doesn't match provider contents
			true,    // found
		)
		assert.Equal(t, "hello", v.Value(), "expected to use user-supplied value")
		assert.Equal(t, "foo", v.Get("scalar").Value(), "get exposes data not in top-level value") // WAT?!
	})

	t.Run("found overrides provider", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			false,    // found, doesn't match provider
		)
		assert.False(t, v.HasValue(), "unexpected string representation")
	})

	t.Run("found overrides provider", func(t *testing.T) {
		v := NewValue(
			provider,
			"scalar", // key
			"foo",    // value
			false,    // found, doesn't match provider
		)
		assert.False(t, v.HasValue(), "unexpected string representation")
	})

	t.Run("populate", func(t *testing.T) {
		var s string
		v := NewValue(
			provider,
			"scalar", // key
			"bar",    // value, doesn't match provider contents
			true,     // found
		)
		require.NoError(t, v.Populate(&s), "error on first populate")
		assert.Equal(t, "foo", s, "expected to use provider-contained value")
	})
}
