// Copyright (c) 2018 Uber Technologies, Inc.
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
	"os"

	"go.uber.org/multierr"
	yaml "gopkg.in/yaml.v2"
)

// A YAMLOption alters the default configuration of the YAML configuration
// provider.
type YAMLOption interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) { f(c) }

// Expand enables variable expansion in all provided sources. The supplied
// function MUST behave like os.LookupEnv: it looks up a key and returns a
// value and whether the key was found.
//
// Expand allows variable references to take two forms: $VAR or
// ${VAR:default}. In the first form, variable names MUST adhere to shell
// naming rules:
//   ...a word consisting solely of underscores, digits, and alphabetics form
//   the portable character set. The first character of a name may not be a
//   digit.
// In this form, NewYAML returns an error if any referenced variables aren't
// found.
//
// In the second form, all characters between the opening curly brace and the
// first colon are used as the key, and all characters from the colon to the
// closing curly brace are used as the default value. Keys need not adhere to
// the shell naming rules above. If a variable isn't found, the default value
// is used.
//
// $$ is expanded to a literal $.
func Expand(lookup LookupFunc) YAMLOption {
	return optionFunc(func(c *config) {
		c.lookup = lookup
	})
}

// Permissive disables gopkg.in/yaml.v2's strict mode. It's provided for
// backward compatibility; to avoid a variety of common mistakes, most users
// should leave YAML providers in the default strict mode.
//
// In permissive mode, duplicate keys in the same source file are allowed.
// Later values override earlier ones (note that duplicates are NOT merged,
// unlike all other merges in this package). Calls to Populate that don't use
// all keys present in the YAML are allowed. Finally, type conflicts are
// allowed when merging source files, with later values replacing earlier
// ones.
func Permissive() YAMLOption {
	return optionFunc(func(c *config) {
		c.strict = false
	})
}

// Name customizes the name of the provider. The default name is "YAML".
func Name(name string) YAMLOption {
	return optionFunc(func(c *config) {
		c.name = name
	})
}

// Source adds a source of YAML configuration. Later sources override earlier
// ones using the merge logic described in the package-level documentation.
func Source(r io.Reader) YAMLOption {
	return optionFunc(func(c *config) {
		c.sources = append(c.sources, r)
	})
}

// File opens a file, uses it as a source of YAML configuration, and closes it
// once provider construction is complete. Priority and merge logic are
// identical to Source.
func File(name string) YAMLOption {
	f, err := os.Open(name)
	if err != nil {
		return failed(err)
	}
	return optionFunc(func(c *config) {
		c.sources = append(c.sources, f)
		c.closers = append(c.closers, f)
	})
}

// Static serializes a Go data structure to YAML and uses the result as a
// source. If serialization fails, provider construction will return an error.
// Priority and merge logic are identical to Source.
func Static(val interface{}) YAMLOption {
	bs, err := yaml.Marshal(val)
	if err != nil {
		return failed(err)
	}
	return Source(bytes.NewReader(bs))
}

func failed(err error) YAMLOption {
	return optionFunc(func(c *config) {
		c.err = multierr.Append(c.err, err)
	})
}

type config struct {
	name    string
	strict  bool
	sources []io.Reader
	closers []io.Closer
	lookup  LookupFunc
	err     error
}

func (c *config) close() error {
	var err error
	for _, c := range c.closers {
		err = multierr.Append(err, c.Close())
	}
	return err
}
