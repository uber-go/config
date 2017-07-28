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
	"fmt"
	"reflect"
	"time"
)

const _separator = "."

var _typeOfString = reflect.TypeOf("string")

// A Value holds the value of a configuration
type Value struct {
	root         Provider
	provider     Provider
	key          string
	value        interface{}
	found        bool
	defaultValue interface{}
}

// NewValue creates a configuration value from a provider and a set
// of parameters describing the key.
func NewValue(
	provider Provider,
	key string,
	value interface{},
	found bool,
	timestamp *time.Time,
) Value {
	return Value{
		provider:     provider,
		key:          key,
		value:        value,
		defaultValue: nil,
		found:        found,
	}
}

// Source returns a configuration provider's name.
func (cv Value) Source() string {
	if cv.provider == nil {
		return ""
	}
	return cv.provider.Name()
}

// WithDefault sets the default value that can be overridden
// by providers with a highger priority.
func (cv Value) WithDefault(value interface{}) (Value, error) {
	cv.defaultValue = value
	p, err := NewStaticProvider(map[string]interface{}{cv.key: value})
	if err != nil {
		return cv, err
	}

	cv.root = NewProviderGroup("withDefault", p, cv.provider)
	return cv, nil
}

// String prints out underline value in Value with fmt.Sprint.
func (cv Value) String() string {
	return fmt.Sprint(cv.Value())
}

// IsDefault returns whether the return value is the default.
func (cv Value) IsDefault() bool {
	// TODO(ai) what should the semantics be if the provider has a value that's
	// the same as the default value?
	return !cv.found && cv.defaultValue != nil
}

// HasValue returns whether the configuration has a value that can be used.
func (cv Value) HasValue() bool {
	return cv.found || cv.IsDefault()
}

// Value returns the underlying configuration's value.
func (cv Value) Value() interface{} {
	if cv.found {
		return cv.value
	}
	return cv.defaultValue
}

// Get returns a value scoped in the current value.
func (cv Value) Get(key string) Value {
	return NewScopedProvider(cv.key, cv.provider).Get(key)
}

// this is a quick-and-dirty conversion method that only handles
// a couple of cases and complains if it finds one it doesn't like.
// needs a bunch more cases.
func convertValue(value interface{}, targetType reflect.Type) (interface{}, error) {
	valueType := reflect.TypeOf(value)
	if valueType.AssignableTo(targetType) {
		return value, nil
	} else if targetType == _typeOfString {
		return fmt.Sprint(value), nil
	}

	switch v := value.(type) {
	case string:
		target := reflect.New(targetType).Interface()
		switch target.(type) {
		case *time.Duration:
			return time.ParseDuration(v)
		}
	}

	return nil, fmt.Errorf("can't convert %v to %v", reflect.TypeOf(value).String(), targetType)
}

// Populate fills in an object from configuration.
func (cv Value) Populate(target interface{}) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("can't populate non pointer type %T", target)
	}

	d := decoder{Value: &cv, m: make(map[interface{}]struct{})}

	return d.unmarshal(cv.key, reflect.Indirect(reflect.ValueOf(target)), "")
}
