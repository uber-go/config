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
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"go.uber.org/config/internal/merge"
	"go.uber.org/config/internal/unreachable"

	"golang.org/x/text/transform"
	yaml "gopkg.in/yaml.v2"
)

const _separator = "."

// YAML is a provider that reads from one or more YAML sources. Many aspects
// of the resulting provider's behavior can be altered by passing functional
// options.
//
// By default, the YAML provider attempts to proactively catch common mistakes
// by enabling gopkg.in/yaml.v2's strict mode. See the package-level
// documentation on strict unmarshalling for details.
//
// When populating Go structs, values produced by the YAML provider correctly
// handle all struct tags supported by gopkg.in/yaml.v2. See
// https://godoc.org/gopkg.in/yaml.v2#Marshal for details.
type YAML struct {
	name     string
	raw      [][]byte
	contents interface{}
	strict   bool
	empty    bool
}

// NewYAML constructs a YAML provider. See the various YAMLOptions for
// available tweaks to the default behavior.
func NewYAML(options ...YAMLOption) (*YAML, error) {
	cfg := &config{
		strict: true,
		name:   "YAML",
	}
	for _, o := range options {
		o.apply(cfg)
	}

	if cfg.err != nil {
		return nil, fmt.Errorf("error applying options: %v", cfg.err)
	}

	// On construction, go through a full merge-serialize-deserialize cycle to
	// catch any duplicated keys as early as possible (in strict mode). It also
	// strips comments, which stops us from attempting environment variable
	// expansion. (We'll expand environment variables next.)
	merged, err := merge.YAML(cfg.sources, cfg.strict)
	if err != nil {
		return nil, fmt.Errorf("couldn't merge YAML sources: %v", err)
	}

	if cfg.lookup != nil {
		exp, err := ioutil.ReadAll(transform.NewReader(merged, newExpandTransformer(cfg.lookup)))
		if err != nil {
			return nil, fmt.Errorf("couldn't expand environment: %v", err)
		}
		merged = bytes.NewBuffer(exp)
	}

	y := &YAML{
		name:   cfg.name,
		raw:    cfg.sources,
		strict: cfg.strict,
	}

	dec := yaml.NewDecoder(merged)
	dec.SetStrict(cfg.strict)
	if err := dec.Decode(&y.contents); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("couldn't decode merged YAML: %v", err)
		}
		y.empty = true
	}

	return y, nil
}

// Name returns the name of the provider. It defaults to "YAML".
func (y *YAML) Name() string {
	return y.name
}

// Get retrieves a value from the configuration. The supplied key is treated
// as a period-separated path, with each path segment used as a map key. For
// example, if the provider contains the YAML
//   foo:
//     bar:
//       baz: hello
// then Get("foo.bar") returns a value holding
//   baz: hello
//
// To get a value holding the entire configuration, use the Root constant as
// the key.
func (y *YAML) Get(key string) Value {
	return y.get(strings.Split(key, _separator))
}

func (y *YAML) get(path []string) Value {
	if len(path) == 1 && path[0] == Root {
		path = nil
	}
	return Value{
		path:     path,
		provider: y,
	}
}

// at returns the unmarshalled representation of the value at a given path,
// with a bool indicating whether the value was found.
//
// YAML mappings are unmarshalled as map[interface{}]interface{}, sequences as
// []interface{}, and scalars as interface{}.
func (y *YAML) at(path []string) (interface{}, bool) {
	if y.empty {
		return nil, false
	}

	cur := y.contents
	for _, segment := range path {
		m, ok := cur.(map[interface{}]interface{})
		if !ok {
			return nil, false
		}
		if _, ok := m[segment]; !ok {
			return nil, false
		}
		cur = m[segment]
	}
	return cur, true
}

func (y *YAML) populate(path []string, i interface{}) error {
	val, ok := y.at(path)
	if !ok {
		return nil
	}
	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(val); err != nil {
		// Provider contents were produced by unmarshaling YAML, this isn't
		// possible.
		err := fmt.Errorf(
			"couldn't marshal config at key %s to YAML: %v",
			strings.Join(path, _separator),
			err,
		)
		return unreachable.Wrap(err)
	}
	dec := yaml.NewDecoder(buf)
	dec.SetStrict(y.strict)
	// Decoding can't ever return EOF, since encoding any value is guaranteed to
	// produce non-empty YAML.
	return dec.Decode(i)
}

func (y *YAML) withDefault(d interface{}) (*YAML, error) {
	rawDefault := &bytes.Buffer{}
	if err := yaml.NewEncoder(rawDefault).Encode(d); err != nil {
		return nil, fmt.Errorf("can't marshal default to YAML: %v", err)
	}
	// It's possible that one of the sources used when initially configuring the
	// provider was nothing but a top-level null, but that a higher-priority
	// source included some additional data. In that case, the result of merging
	// all the sources is non-null. However, the explicitly-null source should
	// override all data provided by withDefault. To handle this correctly, we
	// must use the new defaults as the lowest-priority source and re-merge the
	// original sources.
	sources := make([][]byte, 0, len(y.raw)+1)
	sources = append(sources, rawDefault.Bytes())
	sources = append(sources, y.raw...)
	merged, err := merge.YAML(sources, y.strict)
	if err != nil {
		// The defaults and the existing config don't agree on the type of a value
		// (e.g., we're defaulting a mapping to a sequence, which doesn't make
		// sense).
		return nil, fmt.Errorf("merging default and existing YAML failed: %v", err)
	}

	newY := &YAML{
		name:   y.name,
		raw:    sources,
		strict: y.strict,
	}

	dec := yaml.NewDecoder(merged)
	dec.SetStrict(y.strict)
	if err := dec.Decode(&newY.contents); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("unmarshaling merged YAML failed: %v", err)
		}
		newY.empty = true
	}

	return newY, nil
}

// A Value is a subset of a provider's configuration.
type Value struct {
	path     []string
	provider *YAML
}

// NewValue is a highly error-prone constructor preserved only for backward
// compatibility. If value and found don't match the contents of the provider
// at the supplied key, it panics.
//
// Deprecated: this internal constructor was mistakenly exported in the
// initial release of this package, but its behavior was often very
// surprising. To guarantee sane behavior without changing the function
// signature, input validation and panics were added in version 1.2. In all
// cases, it's both safer and less verbose to use Provider.Get directly.
func NewValue(p Provider, key string, value interface{}, found bool) Value {
	actual := p.Get(key)
	if has := actual.HasValue(); has != found {
		var tmpl string
		if has {
			tmpl = "inconsistent parameters: provider %s has value at key %q but found parameter was false"
		} else {
			tmpl = "inconsistent parameters: provider %s has no value at key %q but found parameter was true"
		}
		panic(fmt.Sprintf(tmpl, p.Name(), key))
	}
	contents := actual.Value()
	same, err := areSameYAML(contents, value)
	if err != nil {
		panic(fmt.Sprintf("can't check NewValue parameter consistency: %v", err))
	}
	if !same {
		tmpl := "inconsistent parameters: provider %s has %#v at key %q but value was %#v"
		panic(fmt.Sprintf(tmpl, p.Name(), contents, key, value))
	}
	return actual
}

// Source returns the name of the value's provider.
func (v Value) Source() string {
	return v.provider.Name()
}

// Populate unmarshals the value into the target struct, much like
// json.Unmarshal or yaml.Unmarshal. When populating a struct with some fields
// already set, data is deep-merged as described in the package-level
// documentation.
func (v Value) Populate(target interface{}) error {
	return v.provider.populate(v.path, target)
}

// Get dives further into the configuration, pulling out more deeply nested
// values. The supplied path is split on periods, and each segment is treated
// as a nested map key. For example, if the current value holds the YAML
// configuration
//   foo:
//     bar:
//       baz: quux
// then a call to Get("foo.bar") will hold the YAML mapping
//   baz: quux
func (v Value) Get(path string) Value {
	if path == Root {
		return v
	}
	extended := make([]string, len(v.path))
	copy(extended, v.path)
	extended = append(extended, strings.Split(path, _separator)...)
	return v.provider.get(extended)
}

// HasValue checks whether any configuration is available at this key.
//
// It doesn't distinguish between configuration supplied during provider
// construction and configuration applied by WithDefault. If the value has
// explicitly been set to nil, HasValue is true.
//
// Deprecated: this function has little value and is often confusing. Rather
// than checking whether a value has any configuration available, Populate a
// struct with appropriate defaults and zero values.
func (v Value) HasValue() bool {
	_, ok := v.provider.at(v.path)
	return ok
}

func (v Value) String() string {
	return fmt.Sprint(v.Value())
}

// Value unmarshals the configuration into interface{}.
//
// Deprecated: in a strongly-typed language, unmarshaling configuration into
// interface{} isn't helpful. It's safer and easier to use Populate with a
// strongly-typed struct.
func (v Value) Value() interface{} {
	// Simplest way to ensure that the caller can't mutate the configuration is
	// to deep-copy with Populate.
	var i interface{}
	if err := v.Populate(&i); err != nil {
		// Unreachable, since we've already ensured that the underlying YAML is
		// valid. Can't alter this signature to include an error without breaking
		// backward compatibility.
		panic(unreachable.Wrap(err).Error())
	}
	return i
}

// WithDefault supplies a default configuration for the value. The default is
// serialized to YAML, and then the existing configuration sources are
// deep-merged into it using the merge logic described in the package-level
// documentation.
//
// Deprecated: the deep-merging behavior of WithDefault is complex, especially
// when applied multiple times. Instead, create a Go struct, set any defaults
// directly on the struct, then call Populate.
func (v Value) WithDefault(d interface{}) (Value, error) {
	fallback := d
	for i := len(v.path) - 1; i >= 0; i-- {
		fallback = map[string]interface{}{v.path[i]: fallback}
	}
	p, err := v.provider.withDefault(fallback)
	if err != nil {
		return Value{}, err
	}
	return Value{path: v.path, provider: p}, nil
}
