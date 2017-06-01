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
	"io/ioutil"
	"os"
	"strings"

	"github.com/ogier/pflag"
)

const (
	// ServiceNameKey is the config key of the service name.
	ServiceNameKey = "name"

	// ServiceDescriptionKey is the config key of the service
	// description.
	ServiceDescriptionKey = "description"

	// ServiceOwnerKey is the config key for a service owner.
	ServiceOwnerKey = "owner"

	// _defaultInit is used to load a default set of configuration files.
	_defaultInit = `
- path: ${CONFIG_DIR:config}/base.yaml
  interpolate: true
- path: ${CONFIG_DIR:config}/${ENVIRONMENT:development}.yaml
  interpolate: true
- path: ${CONFIG_DIR:config}/secrets.yaml
  optional: true
`
	// _testInit is used to specify files for the TestLoader.
	_testInit = `
- path: ${CONFIG_DIR:config}/base.yaml
  interpolate: true
- path: ${CONFIG_DIR:config}/test.yaml
  interpolate: true
`
)

// LookupFunc is a type alias for a function to look for environment variables,
type LookUpFunc func(string) (string, bool)

// ProviderComposition is a collection of functions to fold the configs.
type ProviderComposition []func(LookUpFunc, Provider) (Provider, error)

// Loader is responsible for loading config providers.
type Loader struct {
	// LookUp is a function to look for interpolated/environment values
	LookUp LookUpFunc

	// Initial config to apply fold functions
	Init Provider

	// Functions to apply on initial provider.
	Apply ProviderComposition
}

// DefaultLoader is going to be used by a service if config is not specified.
// First values are going to be looked in dynamic providers, then in command line provider
// and YAML provider is going to be the last.
var DefaultLoader = Loader{
	LookUp: os.LookupEnv,
	Init:   NewYAMLProviderFromReaderWithExpand(os.LookupEnv, ioutil.NopCloser(strings.NewReader(_defaultInit))),
	Apply: []func(LookUpFunc, Provider) (Provider, error){
		loadFilesFromConfig,
		loadCommandLineProvider,
	},
}

// TestLoader reads configuration from base.yaml and test.yaml files, but skips command line parameters.
var TestLoader = Loader{
	LookUp: os.LookupEnv,
	Init:   NewYAMLProviderFromReaderWithExpand(os.LookupEnv, ioutil.NopCloser(strings.NewReader(_testInit))),
	Apply: []func(LookUpFunc, Provider) (Provider, error){
		loadFilesFromConfig,
	},
}

// Load creates a Provider for use in a service.
func (l Loader) Load() (Provider, error) {
	cfg := l.Init
	for _, f := range l.Apply {
		if f == nil {
			continue
		}

		var err error
		if cfg, err = f(l.LookUp, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func loadCommandLineProvider(lookup LookUpFunc, provider Provider) (Provider, error) {
	var s StringSlice
	pflag.CommandLine.Var(&s, "roles", "")
	return NewProviderGroup("global", provider, NewCommandLineProvider(pflag.CommandLine, os.Args[1:])), nil
}

func loadFilesFromConfig(lookup LookUpFunc, provider Provider) (Provider, error) {
	if provider == nil {
		return provider, nil
	}

	type cfg struct {
		Path        string
		Interpolate bool
		Optional    bool
	}

	var list []cfg
	if err := provider.Get(Root).Populate(&list); err != nil {
		return nil, err
	}

	var res []Provider
	for _, f := range list {
		if _, err := os.Stat(f.Path); os.IsNotExist(err) && f.Optional {
			continue
		}

		reader, err := os.Open(f.Path)
		if err != nil {
			return nil, err
		}

		var p Provider
		if f.Interpolate {
			p = NewYAMLProviderFromReaderWithExpand(lookup, reader)
		} else {
			p = NewYAMLProviderFromReader(reader)
		}

		res = append(res, p)
	}

	return NewProviderGroup("global", res...), nil
}
