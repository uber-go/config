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

	"github.com/spf13/cast"
	"go.uber.org/config"
)

func ExampleValue_Value() {
	p, err := config.NewStaticProvider(map[string]interface{}{
		"string": "example",
		"float":  "2.718281828",
		"int":    42,
		"bool":   true,
	})

	if err != nil {
		log.Fatal(err)
	}

	s := cast.ToString(p.Get("string").Value())
	fmt.Println(s)

	f := cast.ToString(p.Get("float").Value())
	fmt.Println(f)

	i := cast.ToInt(p.Get("int").Value())
	fmt.Println(i)

	b := cast.ToBool(p.Get("bool").Value())
	fmt.Println(b)
	// Output:
	// example
	// 2.718281828
	// 42
	// true
}

func ExampleValue_Value_slices() {
	p, err := config.NewStaticProvider(map[string]interface{}{
		"strings": []string{"one", "two", "three"},
		"ints":    []int{1, 2, 3},
	})

	if err != nil {
		log.Fatal(err)
	}

	s := cast.ToStringSlice(p.Get("strings").Value())
	fmt.Println(s)

	i := cast.ToSlice(p.Get("ints").Value())
	fmt.Println(i)

	// Output:
	// [one two three]
	// [1 2 3]
}

func ExampleValue_Value_maps() {
	p, err := config.NewStaticProvider(map[string]interface{}{
		"struct": struct{ Car string }{Car: "Mazda"},
		"map":    map[string]int{"uno": 1},
	})

	if err != nil {
		log.Fatal(err)
	}

	s := cast.ToStringMap(p.Get("struct").Value())
	fmt.Println(s)

	m := cast.ToStringMap(p.Get("map").Value())
	fmt.Println(m)

	// Output:
	// map[car:Mazda]
	// map[uno:1]
}
