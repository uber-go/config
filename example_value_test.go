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

func ExampleValue_Populate() {
	p, err := config.NewStaticProvider(map[string]interface{}{
		"example": map[string]interface{}{
			"name":    "uber",
			"founded": 2009,
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	var c struct {
		Name    string
		Founded int
	}

	if err := p.Get("example").Populate(&c); err != nil {
		log.Fatal(err)
	}

	fmt.Println(c)
	// Output:
	// {uber 2009}
}

func ExampleValue_Populate_slice() {
	p, err := config.NewYAMLProviderFromBytes([]byte(`
slice:
  - 1
  - 2
  - 3
`))

	if err != nil {
		log.Fatal(err)
	}

	var s []int
	if err := p.Get("slice").Populate(&s); err != nil {
		log.Fatal(err)
	}

	fmt.Println(s)
	// Output:
	// [1 2 3]
}

func ExampleValue_Populate_error() {
	p, err := config.NewYAMLProviderFromBytes([]byte(`
bool: notBool
`))

	if err != nil {
		log.Fatal(err)
	}

	var b bool
	fmt.Println(p.Get("bool").Populate(&b))
	// Output:
	// for key "bool": strconv.ParseBool: parsing "notBool": invalid syntax
}

func ExampleValue_WithDefault() {
	p, err := config.NewYAMLProviderFromBytes([]byte(`
example:
  override: fromYAML
`))

	if err != nil {
		log.Fatal(err)
	}

	type example struct {
		Base     string
		Override string
	}

	d := example{Override: "default", Base: "default"}
	fmt.Printf("Default value:\n%+v\n", d)

	v, err := p.Get("example").WithDefault(d)
	if err != nil {
		log.Fatal(err)
	}

	if err := v.Populate(&d); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Populate:\n%+v\n", d)
	// Output:
	// Default value:
	// {Base:default Override:default}
	// Populate:
	// {Base:default Override:fromYAML}
}
