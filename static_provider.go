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
	"bytes"
	"io"

	"gopkg.in/yaml.v2"
)

type staticProvider struct {
	Provider
}

// NewStaticProvider returns a provider that wraps data and it's fields can be
// accessed via Get method. It is using the yaml marshaler to encode data first,
// and is subject to panic if data contains a fixed sized array.
func NewStaticProvider(data interface{}) (Provider, error) {
	c, err := toReader(data)
	if err != nil {
		return nil, err
	}

	p, err := NewYAMLProviderFromReader(c)
	if err != nil {
		return nil, err
	}

	return staticProvider{Provider: p}, nil
}

// NewStaticProviderWithExpand returns a static provider with values replaced
// by a mapping function.
func NewStaticProviderWithExpand(
	data interface{},
	mapping func(string) (string, bool)) (Provider, error) {
	reader, err := toReader(data)
	if err != nil {
		return nil, err
	}

	p, err := NewYAMLProviderFromReaderWithExpand(mapping, reader)
	if err != nil {
		return nil, err
	}

	return staticProvider{Provider: p}, nil
}

// Name implements the Provider interface.
func (staticProvider) Name() string {
	return "static"
}

func toReader(data interface{}) (io.Reader, error) {
	b, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(b), nil
}
