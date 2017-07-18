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
	"strings"

	flag "github.com/spf13/pflag"
)

// StringSlice is an alias to string slice, that is used to read
// comma separated flag values.
type StringSlice []string

var _ flag.Value = (*StringSlice)(nil)

// String returns slice elements separated by comma.
func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

// Set splits val using comma as separators.
func (s *StringSlice) Set(val string) error {
	*s = StringSlice(strings.Split(val, ","))
	return nil
}

// Type implements pflag.Value interface.
func (s *StringSlice) Type() string {
	return "StringSlice"
}

type commandLineProvider struct {
	Provider
}

// NewCommandLineProvider returns a Provider that is using command line
// parameters as config values. In order to address nested elements one
// can use dots in flag names which are considered separators.
// One can use StringSlice type to work with a list of comma separated strings.
func NewCommandLineProvider(flags *flag.FlagSet, args []string) Provider {
	if err := flags.Parse(args); err != nil {
		panic(err)
	}

	m := make(map[string]interface{})
	flags.VisitAll(func(f *flag.Flag) {
		if ss, ok := f.Value.(*StringSlice); ok {
			slice := []string(*ss)
			tmp := make([]interface{}, len(slice))
			for i, str := range slice {
				tmp[i] = str
			}

			m[f.Name] = tmp
			return
		}

		m[f.Name] = f.Value.String()
	})

	return commandLineProvider{Provider: NewStaticProvider(m)}
}

// Name implements the Provider interface.
func (commandLineProvider) Name() string {
	return "cmd"
}
