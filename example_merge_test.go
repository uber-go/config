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

package config_test

import (
	"fmt"
	"log"

	"go.uber.org/config"
)

func ExampleNewProviderGroup() {
	base, err := config.NewYAMLProviderFromBytes([]byte(`
config:
  name: fx
  pool: development
  ports:
  - 80
  - 8080
`))

	if err != nil {
		log.Fatal(err)
	}

	prod, err := config.NewYAMLProviderFromBytes([]byte(`
config:
  pool: production
  ports:
  - 443
`))

	if err != nil {
		log.Fatal(err)
	}

	// Provider is going to keep name from the base provider,
	// but ports and pool values will be overridden by prod.
	p, err := config.NewProviderGroup("merge", base, prod)
	if err != nil {
		log.Fatal(err)
	}

	var c struct {
		Name  string
		Pool  string
		Ports []uint
	}

	if err := p.Get("config").Populate(&c); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", c)
	// Output:
	// {Name:fx Pool:production Ports:[443]}
}
