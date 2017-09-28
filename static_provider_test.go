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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticProvider_Name(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(nil)
	require.NoError(t, err, "Can't create a static provider")

	assert.Equal(t, "static", p.Name())
}

func TestNewStaticProvider_NilData(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(nil)
	require.NoError(t, err, "Can't create a static provider")

	val := p.Get("something")
	assert.False(t, val.HasValue())
}

func TestStaticProvider_WithData(t *testing.T) {
	t.Parallel()

	data := map[string]interface{}{
		"hello": "world",
	}

	p, err := NewStaticProvider(data)
	require.NoError(t, err, "Can't create a static provider")

	val := p.Get("hello")
	assert.True(t, val.HasValue())
	assert.Equal(t, "world", val.String())
}

func TestStaticProvider_WithGet(t *testing.T) {
	t.Parallel()

	data := map[string]interface{}{
		"hello": map[string]int{"world": 42},
	}

	p, err := NewStaticProvider(data)
	require.NoError(t, err, "can't create a static provider")

	val := p.Get("hello")
	assert.True(t, val.HasValue())

	sub := p.Get("hello")
	val = sub.Get("world")
	assert.True(t, val.HasValue())
	assert.Equal(t, 42, val.Value())
}

func TestStaticProviderFmtPrintOnValueNoPanic(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(nil)
	require.NoError(t, err, "Can't create a static provider")

	val := p.Get("something")

	f := func() {
		assert.Contains(t, fmt.Sprintf("%v", val), "")
	}
	assert.NotPanics(t, f)
}

func TestNilStaticProviderSetDefaultTagValue(t *testing.T) {
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

	p, err := NewStaticProvider(nil)
	require.NoError(t, err, "Can't create a static provider")
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

func TestPopulateForSimpleMap(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[string]int{"one": 1, "b": -1})
	require.NoError(t, err, "Can't create a static provider")

	var m map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Equal(t, 1, m["one"])
}

func TestPopulateForNestedMap(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[string]interface{}{
		"top":    map[string]int{"one": 1, "": -1},
		"bottom": "value"})

	require.NoError(t, err, "Can't create a static provider")

	var m map[string]interface{}
	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Equal(t, 2, len(m["top"].(map[interface{}]interface{})))
	assert.Equal(t, 1, m["top"].(map[interface{}]interface{})["one"])
	assert.Equal(t, "value", m["bottom"])
}

func TestPopulateForSimpleSlice(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider([]string{"Eeny", "meeny", "miny", "moe"})
	require.NoError(t, err, "Can't create a static provider")

	var s []string
	require.NoError(t, p.Get(Root).Populate(&s))
	assert.Equal(t, []string{"Eeny", "meeny", "miny", "moe"}, s)

	var str string
	require.NoError(t, p.Get("1").Populate(&str))
	assert.Equal(t, "meeny", str)
	assert.Equal(t, "miny", p.Get("2").String())
}

func TestPopulateForNestedSlices(t *testing.T) {
	t.Parallel()
	p, err := NewStaticProvider([][]string{{}, {"Catch", "a", "tiger", "by", "the", "toe"}, nil, {""}})
	require.NoError(t, err, "can't create a static provider")

	var s [][]string
	require.NoError(t, p.Get(Root).Populate(&s))
	require.Equal(t, 4, len(s))
	assert.Equal(t, [][]string{nil, {"Catch", "a", "tiger", "by", "the", "toe"}, nil, {""}}, s)
	assert.Equal(t, "Catch", p.Get("1.0").String())
}

func TestPopulateForBuiltins(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		p, err := NewStaticProvider(1)
		require.NoError(t, err, "Can't create a static provider")

		var i int
		require.NoError(t, p.Get(Root).Populate(&i))
		assert.Equal(t, 1, i)
		assert.Equal(t, 1, p.Get(Root).Value())
	})
	t.Run("float", func(t *testing.T) {
		p, err := NewStaticProvider(1.23)
		require.NoError(t, err, "Can't create a static provider")

		var f float64
		require.NoError(t, p.Get(Root).Populate(&f))
		assert.Equal(t, 1.23, f)
		assert.Equal(t, 1.23, p.Get(Root).Value())
	})
	t.Run("string", func(t *testing.T) {
		p, err := NewStaticProvider("pie")
		require.NoError(t, err, "Can't create a static provider")

		var s string
		require.NoError(t, p.Get(Root).Populate(&s))
		assert.Equal(t, "pie", s)
		assert.Equal(t, "pie", p.Get(Root).String())
	})
	t.Run("bool", func(t *testing.T) {
		p, err := NewStaticProvider(true)
		require.NoError(t, err, "Can't create a static provider")

		var b bool
		require.NoError(t, p.Get(Root).Populate(&b))
		assert.True(t, b)
		assert.True(t, p.Get(Root).Value().(bool))
	})
}

func TestPopulateForNestedMaps(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[string]map[string]string{
		"a": {"one": "1", "": ""}})

	require.NoError(t, err, "Can't create a static provider")

	var m map[string]string
	err = p.Get("a").Populate(&m)
	require.Error(t, err, "Shouldn't be able to populate")
	assert.Contains(t, err.Error(), `empty map key is ambiguous`)
	assert.Contains(t, err.Error(), `a.`)
}

func TestPopulateNonPointerType(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(42)
	require.NoError(t, err, "Can't create a static provider")

	x := 13
	err = p.Get(Root).Populate(x)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't populate non pointer type")
}

func TestStaticProviderWithExpand(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProviderWithExpand(map[string]interface{}{
		"slice": []interface{}{"one", "${iTwo:2}"},
		"value": `${iValue:""}`,
		"empty": `${iEmpty:""}`,
		"two":   `${iTwo:2}`,
		"map": map[string]interface{}{
			"drink?": "${iMap:tea?}",
			"tea?":   "with cream",
		},
	}, func(key string) (string, bool) {
		switch key {
		case "iEmpty":
			return "\"\"", true
		case "iValue":
			return "null", true
		case "iTwo":
			return "3", true
		case "iMap":
			return "rum please!", true
		}

		return "", false
	})

	require.NoError(t, err, "can't create a static provider")

	assert.Equal(t, "one", p.Get("slice.0").String())
	assert.Equal(t, "3", p.Get("slice.1").String())
	assert.Equal(t, nil, p.Get("value").Value())
	assert.Equal(t, "", p.Get("empty").Value())
	assert.Equal(t, int(3), p.Get("two").Value())

	assert.Equal(t, "rum please!", p.Get("map.drink?").String())
	assert.Equal(t, "with cream", p.Get("map.tea?").String())
}

func TestStaticProviderWithExpandEscapeHandling(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProviderWithExpand(map[string]interface{}{
		"nil":   "a",
		"one":   "$a",
		"two":   "$$a",
		"three": "$$$a",
		"four":  "$$$$a",
		"five":  "$$$$$a",
	}, func(key string) (string, bool) {
		return fmt.Sprint(len(key)), true
	})

	require.NoError(t, err, "can't create a static provider")
	assert.Equal(t, "a", p.Get("nil").String())
	assert.Equal(t, "1", p.Get("one").String())
	assert.Equal(t, "$a", p.Get("two").String())
	assert.Equal(t, "$1", p.Get("three").String())
	assert.Equal(t, "$$a", p.Get("four").String())
	assert.Equal(t, "$$1", p.Get("five").String())
}

func TestPopulateForMapOfDifferentKeyTypes(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[int]string{1: "a"})
	require.NoError(t, err, "Can't create a static provider")

	var m map[string]string
	require.NoError(t, p.Get(Root).Populate(&m))
	assert.Equal(t, "a", m["1"])
}

func TestPopulateForMapsWithNotAssignableKeyTypes(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(map[int]string{1: "a"})
	require.NoError(t, err, "Can't create a static provider")

	type secretType struct{ password string }
	var m map[secretType]string
	err = p.Get(Root).Populate(&m)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, `can't convert "1" to "config.secretType"`)
	assert.Contains(t, msg, "key types conversion")
}

func TestInterpolatedBool(t *testing.T) {
	t.Parallel()

	f := func(key string) (string, bool) {
		if key == "interpolate" {
			return "true", true
		}
		return "", false
	}

	p, err := NewYAMLProviderFromReaderWithExpand(
		f,
		ioutil.NopCloser(strings.NewReader("val: ${interpolate:false}")))

	require.NoError(t, err, "Can't create a static provider")

	assert.Equal(t, "true", p.Get("val").String())
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	type cfg struct{ N int }

	var c cfg
	p, err := NewStaticProvider(nil)
	require.NoError(t, err, "Can't create a static provider")

	v, err := p.Get("metrics").WithDefault(cfg{42})
	require.NoError(t, err, "Can't set a default value")

	require.NoError(t, v.Populate(&c), "Failed to populate config.")

	assert.Equal(t, cfg{N: 42}, c)
}

func TestConfigDefaultsAreOverriddenByHigherPriorityProviders(t *testing.T) {
	t.Parallel()

	type book struct {
		Title  string
		Author string
		Year   int
	}

	p, err := NewStaticProvider(map[string]string{
		"library.author": "Dreiser",
		"library.title":  "The Financier"},
	)

	require.NoError(t, err, "Can't create a static provider")

	v, err := p.Get("library").WithDefault(book{
		Title: "An American Tragedy",
		Year:  1925,
	})

	require.NoError(t, err, "Can't setup a default value")

	var novel book
	require.NoError(t, v.Populate(&novel), "Failed to write a novel.")

	assert.Equal(t, book{Title: "The Financier", Author: "Dreiser", Year: 1925}, novel)
}

type grumpyMarshalYAML struct{}

func (grumpyMarshalYAML) MarshalYAML() (interface{}, error) {
	return nil, errors.New("grumpy")
}

func TestStaticProviderConstructorErrors(t *testing.T) {
	t.Parallel()

	var a grumpyMarshalYAML
	t.Run("Regular static provider closer error", func(t *testing.T) {
		_, err := NewStaticProvider(a)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "grumpy")
	})

	t.Run("Static provider with expand closer error", func(t *testing.T) {
		_, err := NewStaticProviderWithExpand(a, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "grumpy")
	})

	t.Run("Static provider with expand error", func(t *testing.T) {
		_, err := NewStaticProviderWithExpand(`val: ${Email}`, func(string) (string, bool) { return "", false })
		require.Error(t, err)
		assert.Contains(t, err.Error(), `default is empty for "Email"`)
	})
}

func TestTextUnmarshalerOnMissingValue(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(nil)
	require.NoError(t, err)
	ds := duckTales{}
	require.NoError(t, p.Get(Root).Populate(&ds))
}

func TestPopulateNilPointer(t *testing.T) {
	t.Parallel()

	p, err := NewStaticProvider(13)
	require.NoError(t, err)

	var i *int
	err = p.Get(Root).Populate(i)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `can't populate nil *int`)
}
