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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderGroup(t *testing.T) {
	t.Parallel()
	pg := NewProviderGroup("test-group", NewYAMLProviderFromBytes([]byte(`id: test`)))
	assert.Equal(t, "test-group", pg.Name())
	assert.Equal(t, "test", pg.Get("id").AsString())
}

func TestProviderGroupScope(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello": map[string]int{"world": 42}}
	pg := NewProviderGroup("test-group", NewStaticProvider(data))
	assert.Equal(t, 42, pg.Get("hello").Get("world").AsInt())
}


func TestScope_WithGetFromValue(t *testing.T) {
	t.Parallel()
	mock := NewYAMLProviderFromBytes([]byte(`uber.fx: go-lang`))
	scope := NewScopedProvider("", mock)
	require.Equal(t, "go-lang", scope.Get("uber.fx").AsString())
	require.False(t, scope.Get("uber").HasValue())

	base := scope.Get("uber")
	require.Equal(t, "go-lang", base.Get("fx").AsString())
	require.False(t, base.Get("").HasValue())

	uber := base.Get(Root)
	require.Equal(t, "go-lang", uber.Get("fx").AsString())
	require.False(t, uber.Get("").HasValue())

	fx := uber.Get("fx")
	require.Equal(t, "go-lang", fx.Get("").AsString())
	require.False(t, fx.Get("fx").HasValue())
}

func TestProviderGroupScopingValue(t *testing.T) {
	t.Parallel()
	fst := []byte(`
logging:`)

	snd := []byte(`
logging:
  enabled: true
`)
	pg := NewProviderGroup("group", NewYAMLProviderFromBytes(snd), NewYAMLProviderFromBytes(fst))
	assert.True(t, pg.Get("logging").Get("enabled").AsBool())
}

func TestProviderGroup_GetChecksAllProviders(t *testing.T) {
	t.Parallel()

	pg := NewProviderGroup("test-group",
		NewStaticProvider(map[string]string{"name": "test", "desc": "test"}),
		NewStaticProvider(map[string]string{"owner": "tst@example.com", "name": "fx"}))

	require.NotNil(t, pg)

	var svc map[string]string
	require.NoError(t, pg.Get(Root).Populate(&svc))
	assert.Equal(t, map[string]string{"name": "fx", "owner": "tst@example.com", "desc": "test"}, svc)
}
