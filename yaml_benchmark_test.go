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

	"github.com/stretchr/testify/require"
)

// Current benchmark data:
// BenchmarkYAMLCreateSingleFile-8                    117 allocs/op
// BenchmarkYAMLCreateMultiFile-8                     204 allocs/op
// BenchmarkYAMLSimpleGetLevel1-8                       0 allocs/op
// BenchmarkYAMLSimpleGetLevel3-8                       0 allocs/op
// BenchmarkYAMLSimpleGetLevel7-8                       0 allocs/op
// BenchmarkYAMLPopulate-8                             18 allocs/op
// BenchmarkYAMLPopulateNested-8                       42 allocs/op
// BenchmarkYAMLPopulateNestedMultipleFiles-8          52 allocs/op
// BenchmarkYAMLPopulateNestedTextUnmarshaler-8       233 allocs/op
// BenchmarkZapConfigLoad-8                           136 allocs/op

func BenchmarkYAMLCreateSingleFile(b *testing.B) {
	for n := 0; n < b.N; n++ {
		providerOneFile(b)
	}
}

func BenchmarkYAMLCreateMultiFile(b *testing.B) {
	for n := 0; n < b.N; n++ {
		providerTwoFiles(b)
	}
}

func BenchmarkYAMLSimpleGetLevel1(b *testing.B) {
	provider, err := NewYAMLProviderFromBytes([]byte(`foo: 1`))
	require.NoError(b, err, "Can't create a YAML provider")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo")
	}
}

func BenchmarkYAMLSimpleGetLevel3(b *testing.B) {
	provider, err := NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    baz: 1
`))

	require.NoError(b, err, "Can't create a YAML provider")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo.bar.baz")
	}
}

func BenchmarkYAMLSimpleGetLevel7(b *testing.B) {
	provider, err := NewYAMLProviderFromBytes([]byte(`
foo:
  bar:
    baz:
      alpha:
        bravo:
          charlie:
            foxtrot: 1
`))

	require.NoError(b, err, "Can't create a YAML provider")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		provider.Get("foo.bar.baz.alpha.bravo.charlie.foxtrot")
	}
}

func BenchmarkYAMLPopulate(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	p := providerOneFile(b)
	c := &creds{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api.credentials").Populate(c)
	}
}

func BenchmarkYAMLPopulateNested(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	type api struct {
		URL         string
		Timeout     int
		Credentials creds
	}

	p := providerOneFile(b)
	s := &api{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api").Populate(s)
	}
}

func BenchmarkYAMLPopulateNestedMultipleFiles(b *testing.B) {
	type creds struct {
		Username string
		Password string
	}

	type api struct {
		URL         string
		Timeout     int
		Credentials creds
		Version     string
		Contact     string
	}

	p := providerTwoFiles(b)
	s := &api{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get("api").Populate(s)
	}
}

func BenchmarkYAMLPopulateNestedTextUnmarshaler(b *testing.B) {
	type protagonist struct {
		Hero duckTaleCharacter
	}

	type series struct {
		Episodes []protagonist
	}

	p, err := NewYAMLProviderFromFiles("./testdata/textUnmarshaller.yaml")
	require.NoError(b, err, "Can't create a YAML provider")

	s := &series{}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Get(Root).Populate(s)
	}
}

func providerOneFile(b *testing.B) Provider {
	p, err := NewYAMLProviderFromFiles("./testdata/benchmark1.yaml")
	require.NoError(b, err, "Can't create a yaml provider")

	return p
}

func providerTwoFiles(b *testing.B) Provider {
	p, err := NewYAMLProviderFromFiles(
		"./testdata/benchmark1.yaml",
		"./testdata/benchmark2.yaml",
	)

	require.NoError(b, err, "Can't create a YAML provider")

	return p
}
