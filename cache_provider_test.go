// Copyright (c) 2017 Uber Technologies, Inc.
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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheProviderName(t *testing.T) {
	t.Parallel()
	s, err := NewStaticProvider(nil)
	require.NoError(t, err)

	c, err := newCachedProvider(s)
	require.NoError(t, err, "Can't create a cached provider")

	assert.Equal(t, `cached "static"`, c.Name())
}

func TestCachedProvider_ConstructorErrorsOnNil(t *testing.T) {
	t.Parallel()

	_, err := newCachedProvider(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "received a nil provider")
}

type testCachedProvider struct {
	NopProvider
	f func(string) Value
}

func (t testCachedProvider) Get(key string) Value {
	return t.f(key)
}

func TestCachedProvider_GetNewValues(t *testing.T) {
	t.Parallel()

	count := 0
	m := testCachedProvider{}
	m.f = func(key string) Value {
		if count > 0 {
			t.Fatal("cache was called more than once")
		}
		count++
		return NewValue(m, key, "Simpsons", true)
	}

	p, err := newCachedProvider(m)
	require.NoError(t, err, "Can't create a cached provider")

	v := p.Get("cartoon")

	v = v.Get(Root)
	require.True(t, v.HasValue())
	assert.Equal(t, "Simpsons", v.Value())

	v2 := p.Get("cartoon")
	assert.Equal(t, v, v2)
	assert.Equal(t, p, v.provider)
}

func TestCachedProviderConcurrentUse(t *testing.T) {
	t.Parallel()

	m := testCachedProvider{}
	var count int32
	m.f = func(key string) Value {
		if atomic.LoadInt32(&count) > 1 {
			t.Fatal("cache was called more than twice")
		}

		return NewValue(m, key, "Simpsons", true)
	}

	p, err := newCachedProvider(m)
	require.NoError(t, err, "Can't create a cached provider")

	wg := sync.WaitGroup{}
	wg.Add(2)
	get := func() {
		x := p.Get(Root)
		assert.True(t, x.HasValue())
		wg.Done()
	}

	go get()
	go get()

	wg.Wait()
}
