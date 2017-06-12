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
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
)

const (
	// ServiceNameKey is the config key of the service name.
	ServiceNameKey = "name"

	// ServiceDescriptionKey is the config key of the service
	// description.
	ServiceDescriptionKey = "description"

	// ServiceOwnerKey is the config key for a service owner.
	ServiceOwnerKey = "owner"
)

// LookupFunc is a type alias for a function to look for environment variables,
type LookUpFunc func(string) (string, bool)

// FileInfo represents a file to load.
type FileInfo struct {
	Name        string
	Interpolate bool
}

func LoadDefaults() (Provider, error) {
	env := "development"
	if val, ok := os.LookupEnv("ENVIRONMENT"); ok {
		env = val
	}

	p, err := LoadFromFiles(
		[]string{"config"},
		[]FileInfo{
			{Name: "base.yaml", Interpolate: true},
			{Name: fmt.Sprintf("%s.yaml", env), Interpolate: true},
			{Name: "secrets.yaml"},
		},
		os.LookupEnv)

	if err != nil {
		return nil, err
	}

	// Load commandLineProvider
	var s StringSlice
	f := flag.CommandLine
	if f.Lookup("roles") == nil {
		f.Var(&s, "roles", "")
	}

	return NewProviderGroup("global", p, NewCommandLineProvider(f, os.Args[1:])), nil
}

// LoadTestProvider will read configuration base.yaml and test.yaml from a
func LoadTestProvider() (Provider, error) {
	return LoadFromFiles(
		[]string{"config"},
		[]FileInfo{
			{Name: "base.yaml", Interpolate: true},
			{Name: "test.yaml", Interpolate: true},
		},
		os.LookupEnv)
}

// LoadFromFiles reads configuration `files` from `dirs` using `lookUp` function for interpolation.
// First both slices are interpolated with the provided `lookUp` function.
// Then all the files are loaded from the all dirs. For example:
// ```
// LoadFromFiles([]string{"dir1", "dir2"},[]FileInfo{{"base.yaml"},{"test.yaml"}}, nil)
// ```
// will try to load files in this order:
// 1. `dir1/base.yaml`
// 2. `dir2/base.yaml`
// 3. `dir1/test.yaml`
// 4. `dir2/test.yaml`
// The function will return an error, if there are no providers to load.
func LoadFromFiles(dirs []string, files []FileInfo, lookUp LookUpFunc) (Provider, error) {
	var providers []Provider
	for _, info := range files {
		for _, dir := range dirs {
			name := filepath.Join(dir, info.Name)
			f, err := os.Open(name)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}

			if info.Interpolate {
				providers = append(providers, NewYAMLProviderFromReaderWithExpand(lookUp, f))
			} else {
				providers = append(providers, NewYAMLProviderFromReader(f))
			}
		}
	}

	if len(providers) == 0 {
		return nil, errors.New("no providers were loaded")
	}

	return NewProviderGroup("files", providers...), nil
}
