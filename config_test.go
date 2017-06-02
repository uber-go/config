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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Skip command line parsing
	DefaultLoader.Apply = ProviderComposition{loadFilesFromConfig}
	os.Exit(m.Run())
}

type nested struct {
	Name string `yaml:"name" default:"default_name"`
	ID1  int    `yaml:"id1"`
	ID2  string `yaml:"id2"`
}

var nest1 = []byte(`
id1: 1
id2: 2
`)

type root struct {
	ID        int      `yaml:"id"`
	Names     []string `yaml:"names"`
	Nested    nested   `yaml:"n1"`
	NestedPtr *nested  `yaml:"nptr"`
}

var nestedYaml = []byte(`
id: 1234
names:
  - aiden
  - shawn
  - glib
  - madhu
  - anup
n1:
  name: struct
  id1:	111
  id2:  222
nptr:
  name: ptr
  id1: 	1111
  id2:  2222
`)

var structArrayYaml = []byte(`
things:
  - id1: 0
  - id1: 1
  - id1: 2
`)

var yamlConfig3 = []byte(`
float: 1.123
bool: true
int: 123
string: test string
`)

type arrayOfStructs struct {
	Things []nested `yaml:"things"`
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	withBase(t, func(dir string) {
		lookup := lookUpWithConfig(dir, func(string) (string, bool) { return "", false })

		l := Loader{
			LookUp: lookup,
			Init:   NewStaticProviderWithExpand(_defaultFileList, lookup),
			Apply:  DefaultLoader.Apply,
		}

		f, err := os.Create(filepath.Join(dir, "development.yaml"))
		require.NoError(t, err)
		defer func() { assert.NoError(t, os.Remove(f.Name())) }()

		f.WriteString("override: dev")
		require.NoError(t, f.Close())

		cfg, err := l.Load()
		require.NoError(t, err)

		assert.Equal(t, "global", cfg.Name())
		val := cfg.Get("name")
		assert.Equal(t, "test", val.AsString())

		val = cfg.Get("override")
		assert.Equal(t, "dev", val.AsString())
	}, "name: test\noverride: base")

}

func TestRootNodeConfig(t *testing.T) {
	t.Parallel()
	txt := []byte(`
one:
  two: hello
`)

	cfg := NewYAMLProviderFromBytes(txt).Get(Root).AsString()
	assert.Equal(t, "map[one:map[two:hello]]", cfg)
}

func TestDirectAccess(t *testing.T) {
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
	)

	v := provider.Get("n1.id1").WithDefault("xxx")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())
}

func TestScopedAccess(t *testing.T) {
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
	)

	p1 := provider.Get("n1")

	v1 := p1.Get("id1")
	v2 := p1.Get("idx").WithDefault("nope")

	assert.True(t, v1.HasValue())
	assert.Equal(t, 111, v1.AsInt())
	assert.True(t, v2.IsDefault())
	assert.True(t, v2.HasValue())
	assert.Equal(t, v2.AsString(), "nope")
}

func TestSimpleConfigValues(t *testing.T) {
	t.Parallel()
	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(yamlConfig3),
	)

	assert.Equal(t, 123, provider.Get("int").AsInt())
	assert.Equal(t, "test string", provider.Get("string").AsString())
	_, ok := provider.Get("nonexisting").TryAsString()
	assert.False(t, ok)
	assert.Equal(t, true, provider.Get("bool").AsBool())
	assert.Equal(t, 1.123, provider.Get("float").AsFloat())

	nested := &nested{}
	v := provider.Get("nonexisting")
	assert.NoError(t, v.Populate(nested))
}

func TestGetAsIntegerValue(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		value interface{}
	}{
		{float32(2)},
		{float64(2)},
		{int(2)},
		{int32(2)},
		{int64(2)},
	}
	for _, tc := range testCases {
		cv := NewValue(NewStaticProvider(nil), "key", tc.value, true, Integer, nil)
		assert.Equal(t, 2, cv.AsInt())
	}
}

func TestGetAsFloatValue(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		value interface{}
	}{
		{float32(2)},
		{float64(2)},
		{int(2)},
		{int32(2)},
		{int64(2)},
	}
	for _, tc := range testCases {
		cv := NewValue(NewStaticProvider(nil), "key", tc.value, true, Float, nil)
		assert.Equal(t, float64(2), cv.AsFloat())
	}
}

func TestNestedStructs(t *testing.T) {
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nestedYaml),
	)

	str := &root{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	err := v.Populate(str)
	assert.Nil(t, err)

	assert.Equal(t, 1234, str.ID)
	assert.Equal(t, 1111, str.NestedPtr.ID1)
	assert.Equal(t, "2222", str.NestedPtr.ID2)
	assert.Equal(t, 111, str.Nested.ID1)
	assert.Equal(t, "aiden", str.Names[0])
	assert.Equal(t, "shawn", str.Names[1])
}

func TestArrayOfStructs(t *testing.T) {
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(structArrayYaml),
	)

	target := &arrayOfStructs{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	assert.NoError(t, v.Populate(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, 2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	t.Parallel()

	provider := NewProviderGroup(
		"test",
		NewYAMLProviderFromBytes(nest1),
	)
	target := &nested{}
	v := provider.Get(Root)
	assert.True(t, v.HasValue())
	assert.NoError(t, v.Populate(target))
	assert.Equal(t, "default_name", target.Name)
}

func TestInvalidConfigFailures(t *testing.T) {
	t.Parallel()
	valueType := []byte(`
id: xyz
boolean:
`)
	provider := NewYAMLProviderFromBytes(valueType)
	assert.Panics(t, func() { NewYAMLProviderFromBytes([]byte("bytes: \n\x010")) }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.Get("id").AsInt() }, "Can't parse as int")
	assert.Panics(t, func() { provider.Get("boolean").AsBool() }, "Can't parse empty boolean")
	assert.Panics(t, func() { provider.Get("id").AsFloat() }, "Can't parse as float")
}

func TestNilProvider(t *testing.T) {
	t.Parallel()

	cfg, err := Loader{}.Load()
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestNopProvider_Get(t *testing.T) {
	t.Parallel()

	p := NopProvider{}
	assert.Equal(t, "NopProvider", p.Name())
	assert.NoError(t, p.RegisterChangeCallback("key", nil))
	assert.NoError(t, p.UnregisterChangeCallback("token"))

	v := p.Get("randomKey")
	assert.Equal(t, "NopProvider", v.Source())
	assert.True(t, v.HasValue())
	assert.Nil(t, v.Value())
}

func TestPointerIntField(t *testing.T) {
	t.Parallel()

	type pointerFieldStruct struct {
		Name  string
		Value *int
	}

	var ptrYaml = `
ps:
  name: Hello
  value: 123
`
	p := NewYAMLProviderFromBytes([]byte(ptrYaml))

	cfg := &pointerFieldStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
}

func TestPointerTypedField(t *testing.T) {
	t.Parallel()

	type pointerFieldStruct struct {
		Name  string
		Value *int
	}

	var ptrPort = `
ps:
  name: Hello
  port: 123
`
	p := NewYAMLProviderFromBytes([]byte(ptrPort))

	cfg := &pointerFieldStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
}

func TestPointerChildTypedField(t *testing.T) {
	t.Parallel()

	type Port int
	type childPort struct {
		Port *Port
	}

	type portChildStruct struct {
		Name     string
		Child    *childPort
		Children []childPort
	}

	var ptrChildPort = `
ps:
  name: Hello
  child:
    port: 123
  children:
    - port: 321
`

	p := NewYAMLProviderFromBytes([]byte(ptrChildPort))

	cfg := &portChildStruct{Name: "xxx"}
	v := p.Get("ps")

	require.NoError(t, v.Populate(cfg))
	require.Equal(t, 123, int(*cfg.Child.Port))
}

func TestRPCPortField(t *testing.T) {
	t.Parallel()

	type Port int
	type TChannelOutbound struct {
		Port *Port `yaml:"port"`
	}

	type Outbound struct {
		// Only one of the following must be set.
		TChannel *TChannelOutbound `yaml:"tchannel"`
	}

	type Outbounds []Outbound

	// Config is the YARPC YAML configuration.
	type YARPCConfig struct {
		// Name of the service.
		Name  string `yaml:"name"`
		Stuff int    `yaml:"stuff"`
		// Outbounds specifies how this service sends requests to other services.
		Outbounds Outbounds `yaml:"outbounds"`
	}

	var rpc = `
rpc:
  name: my-cool-service
  stuff: 999
  outbounds:
    - services:
        - buffetpushgateway
      tchannel:
        host: 127.0.0.1
        port: ${COMPANY_TCHANNEL_PORT:321}
`
	lookup := func(key string) (string, bool) {
		require.Equal(t, "COMPANY_TCHANNEL_PORT", key)
		return "4324", true
	}

	p := NewYAMLProviderFromReaderWithExpand(lookup, ioutil.NopCloser(bytes.NewBufferString(rpc)))

	cfg := &YARPCConfig{}
	v := p.Get("rpc")

	require.NoError(t, v.Populate(cfg))
	require.Equal(t, 4324, int(*cfg.Outbounds[0].TChannel.Port))
}

func TestLoader_Environment(t *testing.T) {
	t.Parallel()

	l := Loader{
		LookUp: func(key string) (string, bool) {
			require.Equal(t, "ENVIRONMENT", key)
			return "KGBeast", true
		},
		Apply: []func(LookUpFunc, Provider) (Provider, error){
			func(f LookUpFunc, p Provider) (Provider, error) {
				require.Nil(t, p)
				return NewStaticProviderWithExpand(map[string]string{"hero": `${ENVIRONMENT:"joker"}`}, f), nil
			},
		},
	}

	cfg, err := l.Load()
	require.NoError(t, err)

	assert.Equal(t, "KGBeast", cfg.Get("hero").AsString())
}

func withBase(t *testing.T, f func(dir string), contents string) {
	dir, err := ioutil.TempDir("", "TestLoader_Dirs")
	require.NoError(t, err)

	defer func() { require.NoError(t, os.Remove(dir)) }()

	base, err := os.Create(fmt.Sprintf("%s/base.yaml", dir))
	require.NoError(t, err)
	defer os.Remove(base.Name())

	base.WriteString(contents)
	base.Close()

	f(dir)
}

func lookUpWithConfig(dir string, lookUp LookUpFunc) LookUpFunc {
	return func(key string) (string, bool) {
		if key == "CONFIG_DIR" {
			return dir, true
		}

		return lookUp(key)
	}
}

func TestParallelLoad(t *testing.T) {
	// TODO(alsam) make Load parallel
	t.Skip()
	t.Parallel()

	l := DefaultLoader
	count := 100
	wg := sync.WaitGroup{}
	wg.Add(count)

	withBase(t, func(dir string) {
		l.LookUp = lookUpWithConfig(dir, l.LookUp)
		l.Init = NewYAMLProviderFromReader(ioutil.NopCloser(strings.NewReader(fmt.Sprintf(`- path: %s/base.yaml`, dir))))

		for i := 0; i < count; i++ {
			go func() {
				p, err := l.Load()
				assert.NoError(t, err)
				assert.Equal(t, "bane", p.Get("villain").String())
				wg.Done()
			}()
		}

		wg.Wait()
	}, "villain: bane")

}

func TestZeroInitializeLoader(t *testing.T) {
	t.Parallel()
	var l Loader
	_, err := l.Load()
	require.NoError(t, err)
	l.Apply = []func(LookUpFunc, Provider) (Provider, error){nil}

	_, err = l.Load()
	require.NoError(t, err)
}

func TestLoaderApplyFunctionsErrors(t *testing.T) {
	t.Parallel()

	alwaysBlue := func(LookUpFunc, Provider) (Provider, error) { return nil, errors.New("Jump!") }
	l := Loader{
		Apply: ProviderComposition{alwaysBlue},
	}

	_, err := l.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Jump!")
}

func TestLoaderFailsToFindConfigFile(t *testing.T) {
	t.Parallel()

	withBase(t, func(dir string) {
		l := TestLoader
		l.LookUp = lookUpWithConfig(dir, l.LookUp)

		_, err := l.Load()
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	}, "")
}

func TestDefaultLoaderWhenInitIsNil(t *testing.T) {
	t.Parallel()

	l := DefaultLoader
	l.Init = nil
	p, err := l.Load()
	require.NoError(t, err)
	assert.Nil(t, p)
}
