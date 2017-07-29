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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func setEnv(key, value string) func() {
	res := func() { os.Unsetenv(key) }
	if oldVal, ok := os.LookupEnv(key); ok {
		res = func() { os.Setenv(key, oldVal) }
	}

	os.Setenv(key, value)
	return res
}

func TestDefaultLoad(t *testing.T) {
	assert := assert.New(t)
	require.NoError(t, os.Mkdir("config", os.ModePerm))
	defer func() { assert.NoError(os.Remove("config")) }()

	base, err := os.Create(filepath.Join("config", "base.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(base.Name())) }()

	defer setEnv("PORT", "80")()
	base.WriteString("source: base\ninterpolated: ${PORT:17}\n")
	base.Close()

	// Setup development.yaml
	dev, err := os.Create(filepath.Join("config", "development.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(dev.Name())) }()
	fmt.Fprint(dev, "development: loaded")

	// Setup secrets.yaml
	sec, err := os.Create(filepath.Join("config", "secrets.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(sec.Name())) }()
	fmt.Fprint(sec, "secret: ${password:1111}")

	p, err := LoadDefaults()
	require.NoError(t, err)

	var val map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&val))
	assert.Equal(4, len(val))
	assert.Equal("base", p.Get("source").String())
	assert.Equal("80", p.Get("interpolated").String())
	assert.Equal("loaded", p.Get("development").String())
	assert.Equal("${password:1111}", p.Get("secret").String())
}

func TestDefaultLoadMultipleTimes(t *testing.T) {
	assert := assert.New(t)
	require.NoError(t, os.Mkdir("config", os.ModePerm))
	defer func() { assert.NoError(os.Remove("config")) }()

	base, err := os.Create(filepath.Join("config", "base.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(base.Name())) }()

	defer setEnv("PORT", "80")()
	base.WriteString("source: base\ninterpolated: ${PORT:17}\n")
	base.Close()

	p, err := LoadDefaults()
	require.NoError(t, err)
	p, err = LoadDefaults()
	require.NoError(t, err)

	var val map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&val))
	assert.Equal(2, len(val))
	assert.Equal("base", p.Get("source").String())
	assert.Equal("80", p.Get("interpolated").String())
}

func TestLoadTestProvider(t *testing.T) {
	assert := assert.New(t)
	require.NoError(t, os.Mkdir("config", os.ModePerm))
	defer func() { assert.NoError(os.Remove("config")) }()

	base, err := os.Create(filepath.Join("config", "base.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(base.Name())) }()

	base.WriteString("source: base")
	base.Close()

	// Setup test.yaml
	tst, err := os.Create(filepath.Join("config", "test.yaml"))
	require.NoError(t, err)
	defer func() { assert.NoError(os.Remove(tst.Name())) }()
	fmt.Fprint(tst, "dir: ${CONFIG_DIR:test}")

	p, err := LoadTestProvider()
	require.NoError(t, err)
	assert.Equal("base", p.Get("source").String())
	assert.Equal("test", p.Get("dir").String())
}

func TestErrorWhenNoFilesLoaded(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestConfig")
	require.NoError(t, err)
	defer func() { assert.NoError(t, os.Remove(dir)) }()

	_, err = LoadFromFiles(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no providers were loaded")
}

func TestRootNodeConfig(t *testing.T) {
	t.Parallel()
	txt := []byte(`
one:
  two: hello
`)

	cfg, err := NewYAMLProviderFromBytes(txt)
	require.NoError(t, err, "Can't create a YAML provider")

	assert.Equal(t, "map[one:map[two:hello]]", cfg.Get(Root).String())
}

func TestDirectAccess(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(nestedYaml)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)

	v, err := provider.Get("n1.id1").WithDefault("xxx")
	require.NoError(t, err, "Can't create a default value")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())
}

func TestScopedAccess(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(nestedYaml)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)

	p1 := provider.Get("n1")

	v1 := p1.Get("id1")
	v2, err := p1.Get("idx").WithDefault("nope")
	require.NoError(t, err)

	assert.True(t, v1.HasValue())
	assert.Equal(t, 111, v1.Value())
	assert.True(t, v2.HasValue())
	assert.Equal(t, v2.String(), "nope")
}

func TestSimpleConfigValues(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(yamlConfig3)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)

	assert.Equal(t, 123, provider.Get("int").Value())
	assert.Equal(t, "test string", provider.Get("string").String())

	_, err = strconv.ParseBool(provider.Get("nonexisting").String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid syntax")

	assert.Equal(t, true, provider.Get("bool").Value())
	assert.Equal(t, 1.123, provider.Get("float").Value())

	nested := &nested{}
	v := provider.Get("nonexisting")
	assert.NoError(t, v.Populate(nested))
}

func TestNestedStructs(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(nestedYaml)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)

	str := &root{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	require.NoError(t, v.Populate(str), "Can't populate struct")

	assert.Equal(t, 1234, str.ID)
	assert.Equal(t, 1111, str.NestedPtr.ID1)
	assert.Equal(t, "2222", str.NestedPtr.ID2)
	assert.Equal(t, 111, str.Nested.ID1)
	assert.Equal(t, "aiden", str.Names[0])
	assert.Equal(t, "shawn", str.Names[1])
}

func TestArrayOfStructs(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(structArrayYaml)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)

	target := &arrayOfStructs{}

	v := provider.Get(Root)

	assert.True(t, v.HasValue())
	assert.NoError(t, v.Populate(target))
	assert.Equal(t, 0, target.Things[0].ID1)
	assert.Equal(t, 2, target.Things[2].ID1)
}

func TestDefault(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(nest1)
	require.NoError(t, err, "Can't create a YAML provider")

	provider := NewProviderGroup("test", p)
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
boolean: hooli
`)

	_, err := NewYAMLProviderFromBytes([]byte("bytes: \n\x010"))
	require.Error(t, err, "Can't parse invalid YAML")
	assert.Contains(t, err.Error(), "yaml: control characters are not allowed")

	_, err = NewYAMLProviderFromBytes(valueType)
	require.NoError(t, err, "Can't create a YAML provider")
}

func TestNopProvider_Get(t *testing.T) {
	t.Parallel()

	p := NopProvider{}
	assert.Equal(t, "NopProvider", p.Name())

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
	p, err := NewYAMLProviderFromBytes([]byte(ptrYaml))
	require.NoError(t, err, "Can't create a YAML provider")

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
	p, err := NewYAMLProviderFromBytes([]byte(ptrPort))
	require.NoError(t, err, "Can't create a YAML provider")

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

	p, err := NewYAMLProviderFromBytes([]byte(ptrChildPort))
	require.NoError(t, err, "Can't create a YAML provider")

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

	p, err := NewYAMLProviderFromReaderWithExpand(
		lookup,
		ioutil.NopCloser(bytes.NewBufferString(rpc)))

	require.NoError(t, err, "Can't create a YAML provider")

	cfg := &YARPCConfig{}
	v := p.Get("rpc")

	require.NoError(t, v.Populate(cfg))
	require.Equal(t, 4324, int(*cfg.Outbounds[0].TChannel.Port))
}

func TestLoadFromFiles(t *testing.T) {
	t.Parallel()

	withBase(t, func(dir string) {
		_, err := LoadFromFiles(
			[]string{dir},
			[]FileInfo{{Name: "base.yaml", Interpolate: true}},
			func(string) (string, bool) { return "", false },
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), `default is empty for "Email"`)
	}, "${Email}")
}

func withBase(t *testing.T, f func(dir string), contents string) {
	dir, err := ioutil.TempDir("", "TestConfig")
	require.NoError(t, err)

	defer func() { require.NoError(t, os.Remove(dir)) }()

	base, err := os.Create(fmt.Sprintf("%s/base.yaml", dir))
	require.NoError(t, err)
	defer os.Remove(base.Name())

	base.WriteString(contents)
	base.Close()

	f(dir)
}
