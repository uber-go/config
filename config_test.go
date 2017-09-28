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
	"io/ioutil"
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

	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

	v, err := provider.Get("n1.id1").WithDefault("xxx")
	require.NoError(t, err, "Can't create a default value")

	assert.True(t, v.HasValue())
	assert.Equal(t, 111, v.Value())
}

func TestScopedAccess(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(nestedYaml)
	require.NoError(t, err, "Can't create a YAML provider")
	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

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

	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

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

	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

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

	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

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

	provider, err := NewProviderGroup("test", p)
	require.NoError(t, err)

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
