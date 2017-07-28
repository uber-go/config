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
	"errors"
	"fmt"
	"sync"
)

type cachedProvider struct {
	sync.RWMutex
	cache map[string]Value

	Provider
}

// newCachedProvider returns a concurrent safe provider,
// that caches values of the underlying provider.
func newCachedProvider(p Provider) (Provider, error) {
	if p == nil {
		return nil, errors.New("received a nil provider")
	}

	return &cachedProvider{
		Provider: p,
		cache:    make(map[string]Value),
	}, nil
}

// Name returns a name of the underlying provider.
func (p *cachedProvider) Name() string {
	return fmt.Sprintf("cached %q", p.Provider.Name())
}

// Get retrieves a Value and caches it internally.
// The value is cached only if it is found.
func (p *cachedProvider) Get(key string) Value {
	p.RLock()
	if v, ok := p.cache[key]; ok {
		p.RUnlock()
		return v
	}

	p.RUnlock()

	v := p.Provider.Get(key)
	v.provider = p
	p.Lock()
	p.cache[key] = v
	p.Unlock()

	return v
}
