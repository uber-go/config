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

type providerGroup struct {
	name      string
	providers []Provider
}

// NewProviderGroup creates a configuration provider from a group of providers.
// The highest priority provider is the last.
func NewProviderGroup(name string, providers ...Provider) Provider {
	return providerGroup{
		name:      name,
		providers: providers,
	}
}

// Get iterates through the providers and return the value merged from
// underlying of providers.
//
// The merge strategy for two objects
// from the first provider(A) and the last provider(B) is following:
// * if A and B are maps, A and B will form a new map with keys from
// A and B and values from B will overwrite values of A. e.g.
//   A:                B:                 merge(A, B):
//     keep:A            new:B              keep:A
//     update:fromA      update:fromB       update:fromB
//                                          new:B
//
// * if A is a map and B is not, this function will panic,
// e.g. key:value and -slice
//
// * in all the remaining cases B will overwrite A.
func (p providerGroup) Get(key string) Value {
	var res interface{}
	found := false
	for _, provider := range p.providers {
		if val := provider.Get(key); val.HasValue() && !val.IsDefault() {
			res = mergeMaps(res, val.value)
			found = true
		}
	}

	cv := NewValue(p, key, res, found, nil)

	// here we add a new root, which defines the "scope" at which
	// Populates will look for values.
	cv.root = p
	return cv
}

// Name returns the name this provider was created with.
func (p providerGroup) Name() string {
	return p.name
}
