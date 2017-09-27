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
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _yamlConfig1 = []byte(`
appid: keyvalue
desc: A simple keyvalue service
appowner: owner@service.com
modules:
  rpc:
    bind: :28941
`)

func TestYAMLSimple(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(_yamlConfig1)
	require.NoError(t, err, "Can't create a YAML provider")

	c := p.Get("modules.rpc.bind")
	assert.True(t, c.HasValue())
	assert.NotNil(t, c.Value())

	assert.Equal(t, ":28941", c.String())
}

func TestYAMLEnvInterpolation(t *testing.T) {
	t.Parallel()
	f := func(key string) (string, bool) {
		if key == "OWNER_EMAIL" {
			return "hello@there.yasss", true
		}

		return "", false
	}

	cfg := strings.NewReader(`
name: some $$name here
owner: ${OWNER_EMAIL}
module:
  fake:
    number: ${FAKE_NUMBER:321}`)

	p, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.NoError(t, err, "Can't create a YAML provider")
	require.Equal(t, "321", p.Get("module.fake.number").String())

	owner := p.Get("owner").String()
	require.Equal(t, "hello@there.yasss", owner)

	name := p.Get("name").String()
	require.Equal(t, "some $name here", name)
}

func TestYAMLEnvInterpolationMissing(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`
name: some name here
email: ${EMAIL_ADDRESS}`)

	f := func(string) (string, bool) { return "", false }
	_, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `default is empty for "EMAIL_ADDRESS"`)
}

func TestYAMLEnvInterpolationIncomplete(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`
name: some name here
telephone: ${SUPPORT_TEL:}`)

	f := func(string) (string, bool) { return "", false }
	_, err := NewYAMLProviderFromReaderWithExpand(f, cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `default is empty for "SUPPORT_TEL" (use "" for empty string)`)
}

func TestYAMLEnvInterpolationWithColon(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`fullValue: ${MISSING_ENV:this:is:my:value}`)
	f := func(string) (string, bool) {
		return "", false
	}

	p, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.NoError(t, err, "Can't create a YAML provider")

	require.Equal(t, "this:is:my:value", p.Get("fullValue").String())
}

func TestYAMLEnvInterpolationEmptyString(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`
name: ${APP_NAME:my shiny app}
fullTel: 1-800-LOLZ${TELEPHONE_EXTENSION:""}`)

	f := func(string) (string, bool) { return "", false }
	p, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.NoError(t, err, "Can't create a YAML provider")

	require.Equal(t, "my shiny app", p.Get("name").String())
	require.Equal(t, "1-800-LOLZ", p.Get("fullTel").String())
}

type configStruct struct {
	AppID string
	Desc  string
	Owner string `yaml:"appowner"`
}

func TestYamlStructRoot(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(_yamlConfig1)
	require.NoError(t, err, "Can't create a YAML provider")

	cs := &configStruct{}
	require.NoError(t, p.Get(Root).Populate(cs))

	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "owner@service.com", cs.Owner)
}

type rpcStruct struct {
	Bind string `yaml:"bind"`
}

func TestYamlStructChild(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(_yamlConfig1)
	require.NoError(t, err, "Can't create a YAML provider")

	cs := &rpcStruct{}
	assert.NoError(t, p.Get("modules.rpc").Populate(cs))
	assert.Equal(t, ":28941", cs.Bind)
}

func TestExtends(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromFiles(
		"./testdata/base.yaml",
		"./testdata/dev.yaml",
		"./testdata/secrets.yaml")

	require.NoError(t, err, "Can't create a YAML provider")

	baseValue := p.Get("value").String()
	assert.Equal(t, "base_only", baseValue)

	devValue := p.Get("value_override").String()
	assert.Equal(t, "dev_setting", devValue)

	secretValue := p.Get("secret").String()
	assert.Equal(t, "my_${secret}", secretValue)
}

func TestAppRoot(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromFiles(
		"./testdata/base.yaml",
		"./testdata/dev.yaml",
		"./testdata/secrets.yaml")

	require.NoError(t, err, "Can't create a combined YAML provider")

	baseValue := p.Get("value").String()
	assert.Equal(t, "base_only", baseValue)

	devValue := p.Get("value_override").String()
	assert.Equal(t, "dev_setting", devValue)

	secretValue := p.Get("secret").String()
	assert.Equal(t, "my_${secret}", secretValue)
}

func TestNewYAMLProviderFromReader(t *testing.T) {
	t.Parallel()

	buff := bytes.NewBuffer([]byte(_yamlConfig1))
	provider, err := NewYAMLProviderFromReader(buff)
	require.NoError(t, err, "Can't create a YAML provider")

	cs := &configStruct{}
	assert.NoError(t, provider.Get(Root).Populate(cs))
	assert.Equal(t, "keyvalue", cs.AppID)
	assert.Equal(t, "owner@service.com", cs.Owner)
}

func TestYAMLNode(t *testing.T) {
	t.Parallel()

	buff := bytes.NewBuffer([]byte("a: b"))
	node := &yamlNode{value: make(map[interface{}]interface{})}
	require.NoError(t, unmarshalYAMLValue(buff, &node.value))

	assert.Equal(t, "map[a:b]", node.String())
	assert.Equal(t, "map[interface {}]interface {}", node.Type().String())
}

func TestYamlNodeWithNil(t *testing.T) {
	t.Parallel()

	provider, err := NewYAMLProviderFromFiles()
	require.NoError(t, err, "Can't create a YAML provider")

	assert.NotNil(t, provider)
	assert.Panics(t, func() {
		_ = unmarshalYAMLValue(nil, nil)
	}, "Expected panic with nil inpout.")
}

func withYamlBytes(yamlBytes []byte, f func(Provider), t *testing.T) {
	y, err := NewYAMLProviderFromBytes(yamlBytes)
	require.NoError(t, err, "Can't create a YAML provider")

	provider, err := NewProviderGroup("global", y)
	require.NoError(t, err)

	f(provider)
}

func TestMatchEmptyStruct(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		[]byte(``),
		func(provider Provider) {
			es := emptystruct{}
			provider.Get("emptystruct").Populate(&es)
			empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
			assert.True(t, reflect.DeepEqual(empty, es))
		},
		t,
	)
}

func TestMatchPopulatedEmptyStruct(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_emptyyaml,
		func(provider Provider) {
			es := emptystruct{}
			provider.Get("emptystruct").Populate(&es)
			empty := reflect.New(reflect.TypeOf(es)).Elem().Interface()
			assert.True(t, reflect.DeepEqual(empty, es))
		},
		t,
	)
}

func TestPopulateWithPointers(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_pointerYaml,
		func(provider Provider) {
			ps := pointerStruct{}
			provider.Get("pointerStruct").Populate(&ps)
			assert.True(t, *ps.MyTrueBool)
			assert.False(t, *ps.MyFalseBool)
			assert.Equal(t, "hello", *ps.MyString)
		},
		t,
	)
}

func TestNonExistingPopulateWithPointers(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		[]byte(``),
		func(provider Provider) {
			ps := pointerStruct{}
			provider.Get("pointerStruct").Populate(&ps)
			assert.Nil(t, ps.MyTrueBool)
			assert.Nil(t, ps.MyFalseBool)
			assert.Nil(t, ps.MyString)
		},
		t,
	)
}

func TestMapParsing(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_complexMapYaml,
		func(provider Provider) {
			ms := mapStruct{}
			provider.Get("mapStruct").Populate(&ms)

			assert.NotNil(t, ms.MyMap)
			assert.NotZero(t, len(ms.MyMap))

			p, ok := ms.MyMap["policy"].(map[interface{}]interface{})
			assert.True(t, ok)

			for key, val := range p {
				assert.Equal(t, "makeway", key)
				assert.Equal(t, "notanoption", val)
			}

			assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
		},
		t,
	)
}

func TestMapParsingSimpleMap(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_simpleMapYaml,
		func(provider Provider) {
			ms := mapStruct{}
			provider.Get("mapStruct").Populate(&ms)
			assert.Equal(t, 1, ms.MyMap["one"])
			assert.Equal(t, 2, ms.MyMap["two"])
			assert.Equal(t, 3, ms.MyMap["three"])
			assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
		},
		t,
	)
}

func TestMapParsingMapWithNonStringKeys(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_intKeyMapYaml,
		func(provider Provider) {
			ik := intKeyMapStruct{}
			err := provider.Get("intKeyMapStruct").Populate(&ik)
			assert.NoError(t, err)
			assert.Equal(t, "onetwothree", ik.IntKeyMap[123])
		},
		t,
	)
}

func TestDurationParsing(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_durationYaml,
		func(provider Provider) {
			ds := durationStruct{}
			err := provider.Get("durationStruct").Populate(&ds)
			assert.NoError(t, err)
			assert.Equal(t, 10*time.Second, ds.Seconds)
			assert.Equal(t, 20*time.Minute, ds.Minutes)
			assert.Equal(t, 30*time.Hour, ds.Hours)
		},
		t,
	)
}

func TestParsingUnparsableDuration(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_unparsableDurationYaml,
		func(provider Provider) {
			ds := durationStruct{}
			err := provider.Get("durationStruct").Populate(&ds)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "time: unknown unit thhour in duration 25thhour")
		},
		t,
	)
}

func TestTypeOfTypes(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_typeStructYaml,
		func(provider Provider) {
			tts := typeStructStruct{}
			err := provider.Get(Root).Populate(&tts)
			assert.NoError(t, err)
			assert.Equal(t, userDefinedTypeInt(123), tts.TypeStruct.TestInt)
			assert.Equal(t, userDefinedTypeUInt(456), tts.TypeStruct.TestUInt)
			assert.Equal(t, userDefinedTypeFloat(123.456), tts.TypeStruct.TestFloat)
			assert.Equal(t, userDefinedTypeBool(true), tts.TypeStruct.TestBool)
			assert.Equal(t, userDefinedTypeString("hello"), tts.TypeStruct.TestString)
			assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
			assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
			assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
		},
		t,
	)
}

func TestTypeOfTypesPtr(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_typeStructYaml,
		func(provider Provider) {
			tts := typeStructStructPtr{}
			err := provider.Get(Root).Populate(&tts)
			assert.NoError(t, err)
			assert.Equal(t, userDefinedTypeInt(123), *tts.TypeStruct.TestInt)
			assert.Equal(t, userDefinedTypeUInt(456), *tts.TypeStruct.TestUInt)
			assert.Equal(t, userDefinedTypeFloat(123.456), *tts.TypeStruct.TestFloat)
			assert.Equal(t, userDefinedTypeBool(true), *tts.TypeStruct.TestBool)
			assert.Equal(t, userDefinedTypeString("hello"), *tts.TypeStruct.TestString)
			assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
			assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
			assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
		},
		t)
}

func TestTypeOfTypesPtrPtr(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_typeStructYaml,
		func(provider Provider) {
			tts := typeStructStructPtrPtr{}
			err := provider.Get(Root).Populate(&tts)
			assert.NoError(t, err)
			assert.Equal(t, userDefinedTypeInt(123), *tts.TypeStruct.TestInt)
			assert.Equal(t, userDefinedTypeUInt(456), *tts.TypeStruct.TestUInt)
			assert.Equal(t, userDefinedTypeFloat(123.456), *tts.TypeStruct.TestFloat)
			assert.Equal(t, userDefinedTypeBool(true), *tts.TypeStruct.TestBool)
			assert.Equal(t, userDefinedTypeString("hello"), *tts.TypeStruct.TestString)
			assert.Equal(t, 10*time.Second, tts.TypeStruct.TestDuration.Seconds)
			assert.Equal(t, 20*time.Minute, tts.TypeStruct.TestDuration.Minutes)
			assert.Equal(t, 30*time.Hour, tts.TypeStruct.TestDuration.Hours)
		},
		t,
	)
}

func TestHappyTextUnMarshallerParsing(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_happyTextUnmarshallerYaml,
		func(p Provider) {
			ds := duckTales{}

			require.NoError(t, p.Get("duckTales").Populate(&ds))
			assert.Equal(t, scrooge, ds.Protagonist)
			assert.Equal(t, launchpadMcQuack, ds.Pilot)
		},
		t,
	)
}

type atomicDuckTale struct {
	hero duckTaleCharacter
}

func (a *atomicDuckTale) UnmarshalText(b []byte) error {
	return a.hero.UnmarshalText(b)
}

func TestHappyStructTextUnMarshallerParsing(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		[]byte(`hero: LaunchpadMcQuack`),
		func(p Provider) {
			a := &atomicDuckTale{}
			require.NoError(t, p.Get("hero").Populate(a))
			assert.Equal(t, launchpadMcQuack, a.hero)
		},
		t,
	)
}

func TestGrumpyTextUnMarshallerParsing(t *testing.T) {
	t.Parallel()

	withYamlBytes(
		_grumpyTextUnmarshallerYaml,
		func(p Provider) {
			ds := duckTales{}
			err := p.Get("darkwingDuck").Populate(&ds)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Unknown character: DarkwingDuck")
		},
		t,
	)
}

func TestMergeUnmarshaller(t *testing.T) {
	t.Parallel()

	provider, err := NewYAMLProviderFromBytes(_complexMapYaml, _complexMapYamlV2)
	require.NoError(t, err, "Can't create a YAML provider")

	ms := mapStruct{}
	assert.NoError(t, provider.Get("mapStruct").Populate(&ms))
	assert.NotNil(t, ms.MyMap)
	assert.NotZero(t, len(ms.MyMap))

	p, ok := ms.MyMap["policy"].(map[interface{}]interface{})
	assert.True(t, ok)
	for key, val := range p {
		assert.Equal(t, "makeway", key)
		assert.Equal(t, "notanoption", val)
	}

	s, ok := ms.MyMap["pools"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, []interface{}{"very", "funny"}, s)
	assert.Equal(t, "nesteddata", ms.NestedStruct.AdditionalData)
}

func TestMerge(t *testing.T) {
	t.Parallel()

	for _, v := range _mergeTest {
		t.Run(v.description, func(t *testing.T) {
			p, err := NewYAMLProviderFromBytes(v.yaml...)
			require.NoError(t, err, "Can't create a YAML provider")

			for path, exp := range v.expected {
				res := reflect.New(reflect.ValueOf(exp).Type()).Interface()
				assert.NoError(t, p.Get(path).Populate(res))
				assert.Equal(t, exp, reflect.ValueOf(res).Elem().Interface(), "For path: %s", path)
			}
		})
	}
}

func TestMergeError(t *testing.T) {
	t.Parallel()

	src := []byte(`
map:
  key: value
`)
	dst := []byte(`
map:
  - array
`)

	_, err := NewYAMLProviderFromBytes(dst, src)
	require.Error(t, err, "Merge should return an error")
	assert.Contains(t, err.Error(), `can't merge map[interface{}]interface{} and []interface {}. Source: map["key":"value"]. Destination: ["array"]`)
}

func TestYamlProviderFmtPrintOnValueNoPanic(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes(_yamlConfig1)
	require.NoError(t, err, "Can't create a YAML provider")

	c := p.Get("modules.rpc.bind")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", c), "")
	}
	assert.NotPanics(t, f)
}

func TestArrayTypeNoPanic(t *testing.T) {
	t.Parallel()

	// This test will panic if we treat array the same as slice.
	p, err := NewYAMLProviderFromBytes(_yamlConfig1)
	require.NoError(t, err, "Can't create a YAML provider")

	cs := struct {
		ID [6]int `yaml:"id"`
	}{}

	assert.NoError(t, p.Get(Root).Populate(&cs))
}

func TestNilYAMLProviderSetDefaultTagValue(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Set bool `yaml:"set" default:"true"`
	}

	data := struct {
		ID0 int             `yaml:"id0" default:"10"`
		ID1 string          `yaml:"id1" default:"string"`
		ID2 Inner           `yaml:"id2"`
		ID3 []Inner         `yaml:"id3"`
		ID4 map[Inner]Inner `yaml:"id4"`
		ID5 *Inner          `yaml:"id5"`
		ID6 [6]Inner        `yaml:"id6"`
		ID7 [7]*Inner       `yaml:"id7"`
	}{}

	p, err := NewYAMLProviderFromBytes(nil)
	require.NoError(t, err, "Can't create a YAML provider")

	require.NoError(t, p.Get("hello").Populate(&data))

	assert.Equal(t, 10, data.ID0)
	assert.Equal(t, "string", data.ID1)
	assert.True(t, data.ID2.Set)
	assert.Nil(t, data.ID3)
	assert.Nil(t, data.ID4)
	assert.Nil(t, data.ID5)
	assert.True(t, data.ID6[0].Set)
	assert.Nil(t, data.ID7[0])
}

func TestDefaultWithMergeConfig(t *testing.T) {
	t.Parallel()

	base := []byte(`
abc:
  str: "base"
  int: 1
`)

	prod := []byte(`
abc:
  str: "prod"
`)
	cfg := struct {
		Str     string `yaml:"str" default:"nope"`
		Int     int    `yaml:"int" default:"0"`
		Bool    bool   `yaml:"bool" default:"true"`
		BoolPtr *bool  `yaml:"bool_ptr"`
	}{}

	p, err := NewYAMLProviderFromBytes(base, prod)
	require.NoError(t, err, "Can't create a YAML provider")

	p.Get("abc").Populate(&cfg)

	assert.Equal(t, "prod", cfg.Str)
	assert.Equal(t, 1, cfg.Int)
	assert.Equal(t, true, cfg.Bool)
	assert.Nil(t, cfg.BoolPtr)
}

func TestMapOfStructs(t *testing.T) {
	t.Parallel()

	type Bag struct {
		S string
		I int
		P *string
	}
	type Map struct {
		M map[string]Bag
	}

	b := []byte(`
m:
  first:
    s: one
    i: 1
  second:
    s: two
    i: 2
    p: Pointer
`)

	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Map
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, Bag{S: "one", I: 1, P: nil}, r.M["first"])

	snd := r.M["second"]
	assert.Equal(t, 2, snd.I)
	assert.Equal(t, "two", snd.S)
	assert.Equal(t, "Pointer", *snd.P)
}

func TestMapOfSlices(t *testing.T) {
	t.Parallel()

	type Map struct {
		S map[string][]time.Duration
	}

	b := []byte(`
s:
  first:
    - 1s
  second:
    - 2m
    - 3h
`)
	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Map
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, []time.Duration{time.Second}, r.S["first"])
	assert.Equal(t, []time.Duration{2 * time.Minute, 3 * time.Hour}, r.S["second"])
}

func TestMapOfArrays(t *testing.T) {
	t.Parallel()

	type Map struct {
		S map[string][2]time.Duration
	}

	b := []byte(`
s:
  first:
    - 1s
    - 4m
  second:
    - 2m
    - 3h
`)

	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Map
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, [2]time.Duration{time.Second, 4 * time.Minute}, r.S["first"])
	assert.Equal(t, [2]time.Duration{2 * time.Minute, 3 * time.Hour}, r.S["second"])
}

type cycle struct {
	A *cycle
}

type testProvider struct {
	staticProvider
	a cycle
}

func (s *testProvider) Get(key string) Value {
	val, found := s.a, true
	return NewValue(s, key, val, found)
}

func TestLoops(t *testing.T) {
	t.Parallel()

	a := cycle{}
	a.A = &a

	b := cycle{&a}
	require.Equal(t, b, a)

	p := testProvider{}
	err := p.Get(Root).Populate(&b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycles")
	assert.Contains(t, err.Error(), `for key "A.A"`)
}

func TestInternalFieldsAreNotSet(t *testing.T) {
	t.Parallel()

	type External struct {
		internal string
	}

	b := []byte(`
internal: set
`)
	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r External
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, "", r.internal)
}

func TestEmbeddedStructs(t *testing.T) {
	t.Skip("TODO(alsam) GFM(415)")
	t.Parallel()

	type Config struct {
		Foo string
	}

	type Sentry struct {
		DSN string
	}

	type Logging struct {
		Config
		Sentry
	}

	b := []byte(`
logging:
   foo: bar
   sentry:
      dsn: asdf
`)

	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Config
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, "bar", r.Foo)
}

func TestEmptyValuesSetForMaps(t *testing.T) {
	t.Parallel()

	type Hello interface {
		Hello()
	}

	type Foo struct {
		M map[string]Hello
	}

	b := []byte(`
M:
   sayHello:
`)

	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Foo
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Equal(t, r.M, map[string]Hello{"sayHello": Hello(nil)})
}

func TestEmptyValuesSetForStructs(t *testing.T) {
	t.Parallel()

	type Hello interface {
		Hello()
	}

	type Foo struct {
		Say Hello
	}

	b := []byte(`
Say:
`)

	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var r Foo
	require.NoError(t, p.Get(Root).Populate(&r))
	assert.Nil(t, r.Say)
}

type unmarshallerChan chan string

func (m *unmarshallerChan) UnmarshalText(text []byte) error {
	name := string(text)
	if name == "error" {
		return errors.New("unmarshaller channel error")
	}
	*m = make(chan string, 1)
	*m <- "Hello " + name
	return nil
}

type unmarshallerFunc func(string) error

func (m *unmarshallerFunc) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "error" {
		return errors.New("unmarshaller function error")
	}
	*m = func(message string) error {
		return errors.New(message + str)
	}

	return nil
}

func TestHappyUnmarshallerChannelFunction(t *testing.T) {
	t.Parallel()

	type Chart struct {
		Band unmarshallerChan `default:"Beatles"`
		Song unmarshallerFunc `default:"back"`
	}

	f := func(src []byte, band, song string) {
		var r Chart
		p, err := NewYAMLProviderFromBytes(src)
		require.NoError(t, err, "Can't create a YAML provider")

		require.NoError(t, p.Get(Root).Populate(&r))
		require.Equal(t, band, <-r.Band)
		assert.EqualError(t, r.Song("Get "), song)
	}

	b := []byte(`
Band: Rolling Stones
Song: off my cloud
`)

	tests := map[string]func(){
		"defaults":      func() { f([]byte(``), "Hello Beatles", "Get back") },
		"custom values": func() { f(b, "Hello Rolling Stones", "Get off my cloud") },
	}

	for k, v := range tests {
		t.Run(k, func(*testing.T) { v() })
	}
}

func TestGrumpyUnmarshallerChannelFunction(t *testing.T) {
	t.Parallel()

	type S struct {
		C unmarshallerChan
		F unmarshallerFunc
	}

	f := func(src []byte, message string) {
		var r S
		p, err := NewYAMLProviderFromBytes(src)
		require.NoError(t, err, "Can't create a YAML provider")

		e := p.Get(Root).Populate(&r)
		require.Contains(t, e.Error(), message)
	}

	chanError := []byte(`
C: error
F: something
`)

	funcError := []byte(`
C: something
F: error
`)

	tests := map[string]func(){
		"channel error":  func() { f(chanError, "unmarshaller channel error") },
		"function error": func() { f(funcError, "unmarshaller function error") },
	}

	for k, v := range tests {
		t.Run(k, func(*testing.T) { v() })
	}
}

func TestFileNameInPanic(t *testing.T) {
	t.Parallel()

	f, err := ioutil.TempFile("", "testYaml")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.Write([]byte("\t"))
	require.NoError(t, f.Close())

	_, err = NewYAMLProviderFromFiles(f.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), f.Name())
}

func TestYAMLName(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(``))
	require.NoError(t, err, "Can't create a YAML provider")
	require.Contains(t, p.Name(), "yaml")
}

func TestAbsolutePaths(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "TestAbsolutePaths")
	require.NoError(t, err)
	file.WriteString("")
	require.NoError(t, file.Close())
	defer func() { assert.NoError(t, os.Remove(file.Name())) }()

	p, err := NewYAMLProviderFromFiles(file.Name())
	require.NoError(t, err, "Can't create a YAML provider")

	val := p.Get("Imaginary")
	assert.False(t, val.HasValue())
}

func TestPrivateAnonymousField(t *testing.T) {
	t.Parallel()

	type x struct {
		field string
	}

	type y struct {
		x
	}

	b := []byte(`
x:
  field: something
`)
	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	var z y
	require.NoError(t, p.Get(Root).Populate(&z))
	assert.Empty(t, z.field)
}

func TestFlatMapWithDots(t *testing.T) {
	t.Parallel()

	type b struct {
		S string
		I int
	}

	type a struct {
		B b
	}

	bytes := []byte(`
a.b.s: Beethoven
a.b.i: 1770
`)

	p, err := NewYAMLProviderFromBytes(bytes)
	require.NoError(t, err, "Can't create a YAML provider")

	var A a
	require.NoError(t, p.Get("a").Populate(&A))
	assert.Equal(t, 1770, A.B.I)
	assert.Equal(t, "Beethoven", A.B.S)
}

func TestOverridingLongestPath(t *testing.T) {
	t.Parallel()

	type b struct {
		S string
		I int
	}

	type a struct {
		B b
	}

	bytes := []byte(`
a:
  b:
    s: Mozart
    i: 1756
a.b:
  i: 1791
`)

	p, err := NewYAMLProviderFromBytes(bytes)
	require.NoError(t, err, "Can't create a YAML provider")

	var A a
	require.NoError(t, p.Get("a").Populate(&A))
	assert.Equal(t, 1791, A.B.I)
	assert.Equal(t, "Mozart", A.B.S)
}

func TestFlatSingleDots(t *testing.T) {
	t.Parallel()

	type b struct {
		S string
		I int
	}

	type a struct {
		B b
	}

	bytes := []byte(`
.: .
..: ..
...: 3
.................................................: 50
`)
	p, err := NewYAMLProviderFromBytes(bytes)
	require.NoError(t, err, "Can't create a YAML provider")

	require.Equal(t, ".", p.Get(".").String())
	require.Equal(t, "..", p.Get("..").String())
	require.Equal(t, "3", p.Get("...").String())
	require.Equal(t, 50, p.Get(".................................................").Value())
}

func TestDotsFromMultipleSources(t *testing.T) {
	t.Parallel()

	type b struct {
		S string
		I int
	}

	type a struct {
		B b
	}

	base := []byte(`
a:
  b:
    s: Chopin
    i: 1810
`)

	development := []byte(`
a.b:
  s: List
a.b.i: 1811
`)
	p, err := NewYAMLProviderFromBytes(base, development)
	require.NoError(t, err, "Can't create a YAML provider")

	var A a
	require.NoError(t, p.Get("a").Populate(&A))
	assert.Equal(t, 1811, A.B.I)
	assert.Equal(t, "List", A.B.S)
}

func TestMapsWithDottedKeys(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(`
a: b
a.b: c
a.b.c: d
a.b.c.d : e
`))

	require.NoError(t, err, "Can't create a YAML provider")

	var m map[string]string
	require.NoError(t, p.Get(Root).Populate(&m))
	expected := map[string]string{
		"a":       "b",
		"a.b":     "c",
		"a.b.c":   "d",
		"a.b.c.d": "e",
	}

	assert.Equal(t, expected, m)
}

func TestYAMLEnvInterpolationValueMissing(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`name:`)

	f := func(string) (string, bool) { return "", false }
	p, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.NoError(t, err, "Can't create a YAML provider")
	assert.Equal(t, nil, p.Get("name").Value())
}

func TestYAMLEnvInterpolationValueConversion(t *testing.T) {
	t.Parallel()

	cfg := strings.NewReader(`number: ${TWO:3}`)

	f := func(key string) (string, bool) {
		assert.Equal(t, "TWO", key)
		return "3", true
	}

	p, err := NewYAMLProviderFromReaderWithExpand(f, cfg)
	require.NoError(t, err, "Can't create a YAML provider")

	assert.Equal(t, "3", p.Get("number").String())
}

type cartoon struct {
	title string
	year  int
}

func (c *cartoon) UnmarshalText(b []byte) error {
	year := regexp.MustCompile("year:([\\d]+)")
	title := regexp.MustCompile("title:([\\w]+)")
	s := year.FindAllStringSubmatch(string(b), -1)
	c.year = len(s[0][1])

	s = title.FindAllStringSubmatch(string(b), -1)
	c.title = s[0][1]
	return nil
}

func TestUnmarshalTextOnComplexStruct(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(`cartoon:
  year: 1994
  title: FreeWilly`))

	require.NoError(t, err, "Can't create a YAML provider")

	c := &cartoon{}
	require.NoError(t, p.Get("cartoon").Populate(c))
	assert.Equal(t, 4, c.year)
	assert.Equal(t, "FreeWilly", c.title)
}

type jsonUnmarshaller struct {
	Size int
	Name string
}

func (j *jsonUnmarshaller) UnmarshalJSON(b []byte) error {
	if string(b) != `{"name":"maxInt","size":2147483647}` {
		return errors.New("boom")
	}

	j.Name = "mega"
	j.Size = 1000000
	return nil
}

func (j *jsonUnmarshaller) UnmarshalText(b []byte) error {
	panic("should never be called")
}

func TestPopulateOfJSONUnmarshal(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[string]jsonUnmarshaller{
		// Test that big integers are not going to be encoded as floats.
		"pass": {Size: math.MaxInt32, Name: "maxInt"},
		"fail": {Size: 0, Name: "zero"},
	})

	require.NoError(t, err, "Can't create a YAML provider")

	j := jsonUnmarshaller{}
	require.NoError(t, p.Get("pass").Populate(&j))
	assert.Equal(t, j, jsonUnmarshaller{Size: 1000000, Name: "mega"})

	assert.NoError(t, p.Get("empty").Populate(&j), "Empty value shouldn't cause errors.")
	assert.Equal(t, j, jsonUnmarshaller{Size: 1000000, Name: "mega"}, "Empty value shouldn't change actual variable")

	err = p.Get("fail").Populate(&j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

type jsonMarshalError struct{}

func (j *jsonMarshalError) UnmarshalJSON(b []byte) error { return nil }
func (j jsonMarshalError) MarshalJSON() ([]byte, error)  { return nil, errors.New("never give up") }

func TestPopulateOfFailedJSONMarshal(t *testing.T) {
	t.Parallel()

	j := jsonMarshalError{}
	p := newValueProvider(jsonMarshalError{})

	err := p.Get("fail").Populate(&j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "never give up")
}

type yamlUnmarshal struct {
	Size int
	Name string
}

func (y *yamlUnmarshal) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type fakeYAMLUnmarshal struct {
		Size int
		Name string
	}

	var f fakeYAMLUnmarshal

	if err := unmarshal(&f); err == nil {
		y.Name = f.Name + "Fake"
		y.Size = f.Size
		return nil
	}

	m := make(map[string]string)
	if err := unmarshal(&m); err != nil {
		return err
	}

	stringToInt := map[string]int{"one": 1, "two": 2}
	y.Size = stringToInt[m["size"]]
	y.Name = m["name"]

	return nil
}

func TestPopulateUnmarshalersWithDefaultsFromTags(t *testing.T) {
	t.Parallel()

	t.Run("JSON", func(t *testing.T) {
		var j struct {
			J jsonUnmarshaller `default:"explode"`
		}

		p, err := NewStaticProvider(nil)
		require.NoError(t, err)

		err = p.Get(Root).Populate(&j)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("YAML", func(t *testing.T) {
		var y struct {
			Y yamlUnmarshal `default:"Fake: fake"`
		}

		p, err := NewStaticProvider(nil)
		require.NoError(t, err)

		err = p.Get(Root).Populate(&y)
		require.NoError(t, err)
		assert.Equal(t, yamlUnmarshal{Name: "Fake"}, y.Y)
	})
}
func TestPopulateNotAppropriateTypes(t *testing.T) {
	t.Parallel()

	p := newValueProvider(nil)
	t.Run("channel", func(t *testing.T) {
		v := make(chan int)
		require.NoError(t, p.Get(Root).Populate(&v))
	})

	t.Run("func", func(t *testing.T) {
		var f func()
		require.NoError(t, p.Get(Root).Populate(&f))
	})
}

type alwaysBlueYAML struct{}

func (a alwaysBlueYAML) MarshalYAML() (interface{}, error) {
	return nil, errors.New("always blue")
}

func (a *alwaysBlueYAML) UnmarshalYAML(func(interface{}) error) error {
	return nil
}

func TestYAMLMarshallerErrors(t *testing.T) {
	t.Parallel()

	p := newValueProvider(alwaysBlueYAML{})
	var v alwaysBlueYAML
	err := p.Get(Root).Populate(&v)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "always blue")
}

func TestYAMLFailOnMalformedData(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(`foo: ["a", "b", "c"]`))
	require.NoError(t, err, "Can't create a YAML provider")

	var (
		intMap           map[int]int
		intList          []int
		intArray         [2]int
		stringListList   [][]string
		stringArrayArray [2][2]string
	)

	assert := assert.New(t)
	err = p.Get("foo").Populate(&intMap)
	require.Error(t, err, "can't convert []string to")
	assert.Contains(err.Error(), `expected map for key "foo". actual type: "[]interface {}"`)

	err = p.Get("foo").Populate(&intList)
	require.Error(t, err)
	assert.Contains(err.Error(), `parsing "a": invalid syntax`)

	err = p.Get("foo").Populate(&intArray)
	require.Error(t, err)
	assert.Contains(err.Error(), `parsing "a": invalid syntax`)

	err = p.Get("foo").Populate(&stringListList)
	require.Error(t, err)
	assert.Contains(err.Error(), `can't convert "string" to "slice"`)

	err = p.Get("foo").Populate(&stringArrayArray)
	require.Error(t, err)
	assert.Contains(err.Error(), `can't convert "string" to "array"`)
}

func TestJSONDecode(t *testing.T) {
	t.Parallel()

	b := []byte(`- a: b
- c: d`)
	var stringListList [][]string
	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	err = p.Get("").Populate(&stringListList)
	require.Error(t, err)
	assert.Len(t, stringListList, 0)
}

func TestNilsOnMaps(t *testing.T) {
	t.Parallel()

	b := []byte(``)
	var m map[string]string
	p, err := NewYAMLProviderFromBytes(b)
	require.NoError(t, err, "Can't create a YAML provider")

	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Nil(t, m)
}

func TestPopulateOfYAMLUnmarshal(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(`
pass:
  name: deci
  size: 10
fail:
  name: first
  size: one
`))

	require.NoError(t, err, "Can't create a YAML provider")

	y := yamlUnmarshal{}
	require.NoError(t, p.Get("pass").Populate(&y))
	assert.Equal(t, y, yamlUnmarshal{Size: 10, Name: "deciFake"})

	assert.NoError(t, p.Get("empty").Populate(&y), "Empty value shouldn't cause errors.")
	assert.Equal(t, y, yamlUnmarshal{Size: 10, Name: "deciFake"}, "Empty value shouldn't change actual variable")

	assert.NoError(t, p.Get("fail").Populate(&y))
	assert.Equal(t, y, yamlUnmarshal{Size: 1, Name: "first"})
}

func TestFailConversionFromMapsToSlices(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromBytes([]byte(`
foo:
  0: "a"
  1: "b"
`))

	require.NoError(t, err, "Can't create a YAML provider")

	t.Run("map of ints", func(t *testing.T) {
		var intMap map[int]string
		require.NoError(t, p.Get("foo").Populate(&intMap))
		assert.Equal(t, map[int]string{0: "a", 1: "b"}, intMap)
	})

	t.Run("list of strings", func(t *testing.T) {
		var stringList []string
		err = p.Get("foo").Populate(&stringList)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `can't convert "map" to "slice"`)
		require.Len(t, stringList, 0)
	})

	t.Run("list of strings with first overridden value", func(t *testing.T) {
		var stringList []string
		p, err := NewStaticProvider(map[string]int{"foo.0": 1})
		require.NoError(t, err, "can't create a static provider")
		require.NoError(t, p.Get("foo").Populate(&stringList))
		require.Len(t, stringList, 1)
	})

	t.Run("array of strings", func(t *testing.T) {
		var stringArray [2]string
		err := p.Get("foo").Populate(&stringArray)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `can't convert "map" to "array"`)
		require.Equal(t, stringArray, [2]string{"", ""})
	})

	t.Run("list of strings", func(t *testing.T) {
		var stringListList [][]string
		err := p.Get("foo").Populate(&stringListList)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `can't convert "map" to "slice"`)
		require.Len(t, stringListList, 0)
	})

	t.Run("nested map of ints of strings failure", func(t *testing.T) {
		var mapIntMap map[string]map[int]string
		err := p.Get("foo").Populate(&mapIntMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `expected map for key "foo`)
		assert.Contains(t, err.Error(), `actual type: "string"`)
		assert.Nil(t, mapIntMap)
	})
}

func TestSliceElementInDifferentPositions(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		p, err := NewStaticProvider([]int{})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get(Root).Populate(&s))
		assert.Nil(t, s)
	})

	t.Run("first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{1}, s)
	})

	t.Run("nil slice with first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": nil, "a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{1}, s)
	})

	t.Run("empty slice with first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{}, "a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{1}, s)
	})

	t.Run("slice with second element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.1": 3})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{0, 3, 2}, s)
	})

	t.Run("slice with an extra element added", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.3": 3})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{0, 1, 2, 3}, s)
	})

	t.Run("slice with a nil element inside", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.1": nil})
		require.NoError(t, err, "can't create a static provider")

		var s []int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, []int{0, 0, 2}, s)
	})

	t.Run("default value in the middle", func(t *testing.T) {
		type Inner struct {
			Set bool `yaml:"set" default:"true"`
		}

		p, err := NewYAMLProviderFromBytes([]byte(`
a:
- set: true
- get: something
- set: false`))

		require.NoError(t, err, "Can't create a YAML provider")

		var a []Inner
		require.NoError(t, p.Get("a").Populate(&a))
		assert.Equal(t, []Inner{{Set: true}, {Set: true}, {Set: false}}, a)
	})
}

func TestArrayElementInDifferentPositions(t *testing.T) {
	t.Parallel()

	t.Run("empty array", func(t *testing.T) {
		p, err := NewStaticProvider([]int{})
		require.NoError(t, err, "can't create a static provider")

		var s [2]int
		require.NoError(t, p.Get(Root).Populate(&s))
		assert.Equal(t, [2]int{0, 0}, s)
	})

	t.Run("first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s [2]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [2]int{1, 0}, s)
	})

	t.Run("nil collection with first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": nil, "a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s [2]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [2]int{1, 0}, s)
	})

	t.Run("empty collection with first element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{}, "a.0": 1})
		require.NoError(t, err, "can't create a static provider")

		var s [2]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [2]int{1, 0}, s)
	})

	t.Run("collection with second element overridden", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.1": 3})
		require.NoError(t, err, "can't create a static provider")

		var s [2]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [2]int{0, 3}, s)
	})

	t.Run("collection with an extra element added", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.3": 3})
		require.NoError(t, err, "can't create a static provider")

		var s [4]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [4]int{0, 1, 2, 3}, s)
	})

	t.Run("collection with a nil element inside", func(t *testing.T) {
		p, err := NewStaticProvider(map[string]interface{}{"a": []int{0, 1, 2}, "a.1": nil})
		require.NoError(t, err, "can't create a static provider")

		var s [3]int
		require.NoError(t, p.Get("a").Populate(&s))
		assert.Equal(t, [3]int{0, 0, 2}, s)
	})

	t.Run("collection error unmarshalable elements", func(t *testing.T) {
		p, err := NewYAMLProviderFromBytes([]byte(`
a:
- protagonist: Scrooge
  pilot: LaunchpadMcQuack
- protagonist:
`))

		require.NoError(t, err, "Can't create a YAML provider")

		var s [2]duckTales
		err = p.Get("a").Populate(&s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `for key "a.1.Protagonist": Unknown character:`)
	})

	t.Run("default value in the middle", func(t *testing.T) {
		type Inner struct {
			Set bool `yaml:"set" default:"true"`
		}

		p, err := NewYAMLProviderFromBytes([]byte(`
a:
- set: true
- get: something
- get: something
- set: false
a.2.set: false`))

		require.NoError(t, err, "Can't create a YAML provider")

		var a [4]Inner
		require.NoError(t, p.Get("a").Populate(&a))
		assert.Equal(t, [4]Inner{{Set: true}, {Set: true}, {Set: false}, {Set: false}}, a)
	})
}

func TestNewYamlProviderWithExpand(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderWithExpand(nil, "./testdata/base.yaml")
	require.NoError(t, err, "Can't create a YAML provider")

	baseValue := p.Get("value").String()
	assert.Equal(t, "base_only", baseValue)
}

func TestYamlProvidersProduceSameResults(t *testing.T) {
	t.Parallel()

	p, err := NewYAMLProviderFromFiles("./testdata/base.yaml")
	require.NoError(t, err, "Can't create a YAML provider")

	pp, err := NewYAMLProviderWithExpand(nil, "./testdata/base.yaml")
	require.NoError(t, err, "Can't create a YAML provider with expand")

	assert.IsType(t, true, p.Get("a-bool").Value())
	assert.Exactly(t, p.Get("a-bool").Value(), pp.Get("a-bool").Value())
	assert.IsType(t, "empty", p.Get("a-empty").Value())
	assert.Exactly(t, p.Get("a-empty").Value(), pp.Get("a-empty").Value())
	assert.IsType(t, float64(1.2), p.Get("a-float").Value())
	assert.Exactly(t, p.Get("a-float").Value(), pp.Get("a-float").Value())
	assert.IsType(t, int(12), p.Get("a-int").Value())
	assert.Exactly(t, p.Get("a-int").Value(), pp.Get("a-int").Value())
	assert.IsType(t, nil, p.Get("a-nil").Value())
	assert.Exactly(t, p.Get("a-nil").Value(), pp.Get("a-nil").Value())
	assert.IsType(t, nil, p.Get("a-null").Value())
	assert.Exactly(t, p.Get("a-null").Value(), pp.Get("a-null").Value())
	assert.IsType(t, "string", p.Get("a-string").Value())
	assert.Exactly(t, p.Get("a-string").Value(), pp.Get("a-string").Value())
}

func TestMergeErrorsFromReaders(t *testing.T) {
	t.Parallel()

	t.Run("regular", func(t *testing.T) {
		base := strings.NewReader(`a:
  - b`)
		dev := strings.NewReader(`a:
  b: c`)

		_, err := NewYAMLProviderFromReader(base, dev)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})

	t.Run("expand", func(t *testing.T) {
		expand := func(string) (string, bool) { return "", false }

		base := strings.NewReader(`a:
  - b`)
		dev := strings.NewReader(`a:
  b: c`)

		_, err := NewYAMLProviderFromReaderWithExpand(expand, base, dev)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})
}

func TestMergeErrorsFromFiles(t *testing.T) {
	t.Parallel()

	base, err := ioutil.TempFile(".", "test")
	require.NoError(t, err, "Can't create a temp base file")
	fmt.Fprint(base, `a:
  - b`)
	require.NoError(t, base.Close(), "Close error for base")
	defer func() { assert.NoError(t, os.Remove(base.Name())) }()

	dev, err := ioutil.TempFile(".", "test")
	require.NoError(t, err, "Can't create a temp dev file")
	fmt.Fprint(dev, `a:
  b: c`)
	require.NoError(t, dev.Close(), "Close error for dev")
	defer func() { assert.NoError(t, os.Remove(dev.Name())) }()

	t.Run("regular files", func(t *testing.T) {
		_, err := NewYAMLProviderFromFiles(base.Name(), dev.Name())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})

	t.Run("regular readers", func(t *testing.T) {
		b, err := ioutil.ReadFile(base.Name())
		require.NoError(t, err, "Can't read base file")

		d, err := ioutil.ReadFile(dev.Name())
		require.NoError(t, err, "Can't read dev file")

		_, err = NewYAMLProviderFromReader(
			bytes.NewBuffer(b),
			bytes.NewBuffer(d))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})

	t.Run("from files with expand", func(t *testing.T) {
		expand := func(string) (string, bool) { return "", false }

		_, err := NewYAMLProviderWithExpand(expand, base.Name(), dev.Name())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})

	t.Run("reader with expand", func(t *testing.T) {
		b, err := ioutil.ReadFile(base.Name())
		require.NoError(t, err, "Can't read base file")

		d, err := ioutil.ReadFile(dev.Name())
		require.NoError(t, err, "Can't read dev file")

		expand := func(string) (string, bool) { return "", false }

		_, err = NewYAMLProviderFromReaderWithExpand(
			expand,
			bytes.NewBuffer(b),
			bytes.NewBuffer(d))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't merge map")
	})
}

func TestYAMLProviderWithGarbledPath(t *testing.T) {
	t.Parallel()

	t.Run("regular", func(t *testing.T) {
		_, err := NewYAMLProviderFromFiles("./testdata/base.yaml",
			"/some/nonexisting/config")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("expand", func(t *testing.T) {
		_, err := NewYAMLProviderWithExpand(nil, "./testdata/base.yaml",
			"/some/nonexisting/config")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}
