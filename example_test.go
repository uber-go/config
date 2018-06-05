// Copyright (c) 2017-2018 Uber Technologies, Inc.
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
	"strings"

	"go.uber.org/config"
)

func Example() {
	// A struct to represent the configuration of a self-contained unit of your
	// application.
	type cfg struct {
		Parameter string
	}

	// Two sources of YAML configuration to merge. We could also use
	// config.Static to supply some configuration as a Go struct.
	base := strings.NewReader("module: {parameter: foo}")
	override := strings.NewReader("module: {parameter: bar}")

	// Merge the two sources into a Provider. Later sources are higher-priority.
	// See the top-level package documentation for details on the merging logic.
	provider, err := config.NewYAML(config.Source(base), config.Source(override))
	if err != nil {
		panic(err)
	}

	var c cfg
	if err := provider.Get("module").Populate(&c); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", c)
	// Output:
	// {Parameter:bar}
}

func ExampleValue_WithDefault() {
	provider, err := config.NewYAML(config.Static(map[string]string{
		"key": "value",
	}))
	if err != nil {
		panic(err)
	}
	// Using config.Root as a key retrieves the whole configuration.
	base := provider.Get(config.Root)

	// Applying a default is equivalent to serializing it to YAML, writing the
	// serialized bytes to default.yaml, and then merging default.yaml and the
	// existing configuration. Maps are deep-merged!
	//
	// Since we're setting the default for a key that's not already in the
	// configuration, new_key will now be set to new_value. From now on, it's
	// impossible to tell whether the value of new_key came from the original
	// provider or a call to WithDefault.
	defaulted, err := base.WithDefault(map[string]string{
		"new_key": "new_value",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(defaulted)

	// If we try to use WithDefault again to set different defaults for the two
	// existing keys, nothing happens - since both keys already have scalar
	// values, those values overwrite the new defaults in the merge. See the
	// package-level documentation for a more detailed discussion of the merge
	// logic.
	again, err := defaulted.WithDefault(map[string]string{
		"key":     "ignored",
		"new_key": "ignored",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(again)
}
