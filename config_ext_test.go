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

package config_test

import (
	"strings"
	"testing"

	. "go.uber.org/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	first := strings.NewReader(`
nothing: ~
fun:
  - maserati
  - porsche
practical: &ptr
  toyota: camry
  honda: accord
antique_scalar: model_t
antique_sequence:
  - model_t
antique_mapping_empty:  # will override with empty map
  ford: model_t
antique_mapping_nil:  # will override with nil
  ford: model_t
occupants:
  honda:
    driver: jane
    backseat: [nate]
extra_practical:
  <<: *ptr
  volkswagon: jetta
mismatch: scalar
`)
	second := strings.NewReader(`
fun:
  - maserati
  - lamborghini
practical:
  honda: civic
  nissan: altima
antique_scalar: ~
antique_sequence: ~
antique_mapping_empty: {}
antique_mapping_nil: ~
occupants:
  honda:
    passenger: arthur
    backseat: [nora]
mismatch: [scalar]
`)

	provider, err := NewYAMLProviderFromReader(first, second)
	require.NoError(t, err, "couldn't create provider")
	assert.Equal(t, `cached "yaml"`, provider.Get(Root).Source(), "wrong source")
	assert.Equal(t, "<nil>", provider.Get("nothing").String(), "wrong string representation")

	type TestCase struct {
		Value     Value
		HasValue  bool
		Interface interface{}
		Populate  interface{}
	}

	run := func(t testing.TB, tt TestCase) {
		assert.Equal(t, tt.HasValue, tt.Value.HasValue(), "unexpected result from HasValue")
		assert.Equal(t, tt.Interface, tt.Value.Value(), "unexpected result from Value")
		require.NoError(t, tt.Value.Populate(tt.Populate), "error populating")
	}

	t.Run("missing", func(t *testing.T) {
		var s string

		t.Run("top level", func(t *testing.T) {
			run(t, TestCase{
				Value:     provider.Get("not_a_key"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("subkey", func(t *testing.T) {
			run(t, TestCase{
				Value:     provider.Get("practical.cadillac"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("descend into slice", func(t *testing.T) {
			run(t, TestCase{
				Value:     provider.Get("fun.extra"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})
	})

	t.Run("nil", func(t *testing.T) {
		var s string
		run(t, TestCase{
			Value:     provider.Get("nothing"),
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after Populate")
	})

	t.Run("scalar", func(t *testing.T) {
		var s string
		run(t, TestCase{
			Value:     provider.Get("practical.toyota"),
			HasValue:  true,
			Interface: "camry",
			Populate:  &s,
		})
		assert.Equal(t, "camry", s, "unexpected result after Populate")
	})

	t.Run("map", func(t *testing.T) {
		var m map[string]string
		run(t, TestCase{
			Value:    provider.Get("practical"),
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"toyota": "camry",
				"honda":  "civic",
				"nissan": "altima",
			},
			Populate: &m,
		})
		assert.Equal(t, map[string]string{
			"toyota": "camry",
			"honda":  "civic",
			"nissan": "altima",
		}, m, "unexpected result after populate")
	})

	t.Run("slice", func(t *testing.T) {
		var s []string
		run(t, TestCase{
			Value:     provider.Get("fun"),
			HasValue:  true,
			Interface: []interface{}{"maserati", "lamborghini"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"maserati", "lamborghini"}, s, "unexpected result after populate")
	})

	t.Run("struct", func(t *testing.T) {
		type Occupants struct {
			Driver    string
			Passenger string
			Backseat  []string
		}
		var o Occupants
		run(t, TestCase{
			Value:    provider.Get("occupants.honda"),
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"driver":    "jane",
				"passenger": "arthur",
				"backseat":  []interface{}{"nora"},
			},
			Populate: &o,
		})
		assert.Equal(t, Occupants{
			Driver:    "jane",
			Passenger: "arthur",
			Backseat:  []string{"nora"},
		}, o, "unexpected result after populate")
	})

	t.Run("nil merged into scalar", func(t *testing.T) {
		// Merging replaces scalars, so an explicit nil should replace
		// lower-priority values. (This matches gopkg.in/yaml.v2.)
		t.Skip("TODO: ignores explicit nil")
		var s string
		run(t, TestCase{
			Value:     provider.Get("antique_scalar"),
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Empty(t, s, "unexpected result after Populate")
	})

	t.Run("nil merged into sequence", func(t *testing.T) {
		// Merging replaces sequences, so an explicit nil should replace
		// lower-priority values. (This matches gopkg.in/yaml.v2.)
		t.Skip("TODO: ignores explicit nil")
		var s []string
		run(t, TestCase{
			Value:     provider.Get("antique_sequence"),
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Empty(t, s, "unexpected result after Populate")
	})

	t.Run("nil merged into mapping", func(t *testing.T) {
		// Merging deep-merges mappings, but we should honor explicitly-configured
		// nils and replace all existing configuration. (This matches
		// gopkg.in/yaml.v2.)
		t.Skip("TODO: ignores explicit nil")
		var m map[string]string
		run(t, TestCase{
			Value:     provider.Get("antique_mapping_nil"),
			HasValue:  true,
			Interface: nil,
			Populate:  &m,
		})
		assert.Nil(t, m, "unexpected result after Populate")
	})

	t.Run("empty map merged into mapping", func(t *testing.T) {
		// Merging deep-merges mappings, so merging in an empty map is a no-op.
		// (This matches gopkg.in/yaml.v2.)
		var m map[string]string
		run(t, TestCase{
			Value:     provider.Get("antique_mapping_empty"),
			HasValue:  true,
			Interface: map[interface{}]interface{}{"ford": "model_t"},
			Populate:  &m,
		})
		assert.Equal(t, map[string]string{"ford": "model_t"}, m, "unexpected result after Populate")
	})

	t.Run("sequence merged into scalar", func(t *testing.T) {
		var s []string
		run(t, TestCase{
			Value:     provider.Get("mismatch"),
			HasValue:  true,
			Interface: []interface{}{"scalar"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"scalar"}, s, "unexpected result after populate")
	})

	t.Run("anchors and native merge", func(t *testing.T) {
		var m map[string]string
		run(t, TestCase{
			Value:    provider.Get("extra_practical"),
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"toyota":     "camry",
				"honda":      "accord",
				"volkswagon": "jetta",
			},
			Populate: &m,
		})
		assert.Equal(t, map[string]string{
			"toyota":     "camry",
			"honda":      "accord",
			"volkswagon": "jetta",
		}, m, "unexpected result after populate")
	})

	t.Run("multiple gets", func(t *testing.T) {
		var s string
		run(t, TestCase{
			Value:     provider.Get("occupants").Get("honda").Get("driver"),
			HasValue:  true,
			Interface: "jane",
			Populate:  &s,
		})
		assert.Equal(t, "jane", s, "unexpected result after populate")
	})

	t.Run("default missing value", func(t *testing.T) {
		var s string
		val, err := provider.Get("not_there").WithDefault("something")
		require.NoError(t, err, "couldn't set default")
		run(t, TestCase{
			Value:     val,
			HasValue:  true,
			Interface: "something",
			Populate:  &s,
		})
		assert.Equal(t, "something", s, "unexpected result after populate")
	})

	t.Run("default overriden by scalar", func(t *testing.T) {
		var s string
		val, err := provider.Get("practical.honda").WithDefault("CRV")
		require.NoError(t, err, "couldn't set default")
		run(t, TestCase{
			Value:     val,
			HasValue:  true,
			Interface: "civic",
			Populate:  &s,
		})
		assert.Equal(t, "civic", s, "unexpected result after populate")
	})

	t.Run("default overriden by nil", func(t *testing.T) {
		t.Skip("TODO: panics")
		var s string
		val, err := provider.Get("nothing").WithDefault("something")
		require.NoError(t, err, "couldn't set default")
		run(t, TestCase{
			Value:     val,
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after populate")
	})

	t.Run("default merges maps", func(t *testing.T) {
		t.Skip("TODO: defaults aren't deep-merged")
		var m map[string]string
		val, err := provider.Get("practical").WithDefault(map[string]string{
			"ford":   "fiesta",
			"toyota": "corolla",
		})
		require.NoError(t, err, "couldn't set default")
		run(t, TestCase{
			Value:    val,
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"honda":  "civic",
				"toyota": "corolla",
				"nissan": "altima",
				"ford":   "fiesta",
			},
			Populate: &m,
		})
		assert.Equal(t, map[string]string{
			"honda":  "civic",
			"toyota": "corolla",
			"nissan": "altima",
			"ford":   "fiesta",
		}, m, "unexpected result after populate")
	})

	t.Run("default doesn't merge sequences", func(t *testing.T) {
		var s []string
		val, err := provider.Get("fun").WithDefault([]string{"delorean"})
		require.NoError(t, err, "couldn't set default")
		run(t, TestCase{
			Value:     val,
			HasValue:  true,
			Interface: []interface{}{"maserati", "lamborghini"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"maserati", "lamborghini"}, s, "unexpected result after populate")
	})

	t.Run("default ignores type mismatches", func(t *testing.T) {
		// TODO: We should return an error in strict mode.
		// Since defaults are effectively the lowest-priority config, they should
		// control the types expected in users' config. Type mismatches are
		// ignored.
		t.Skip("TODO: default isn't treated as lowest-priority config source")
		d := map[string]string{"dodge": "viper"}
		val, err := provider.Get("fun").WithDefault(d)
		assert.NoError(t, err, "error on type mismatch in permissive mode")
		assert.Equal(t, d, val.Value(), "wrong merged output")
	})

	t.Run("chained defaults", func(t *testing.T) {
		// Each call to WithDefault should merge all existing config into the
		// default, then use the result. This means that repeated calls to
		// WithDefault should deep-merge all the supplied defaults, with the last
		// call to WithDefault having the lowest priority.
		t.Skip("TODO: defaults aren't deep-merged")

		// First, set a default.
		val, err := provider.Get("top").WithDefault(map[string]string{"middle": "bottom"})
		require.NoError(t, err, "couldn't set first default")

		// Second, set another default with a different key. First default should
		// be deep-merged into this one.
		val, err = val.WithDefault(map[string]string{"other_middle": "other_bottom"})
		require.NoError(t, err, "couldn't set second default")

		// Last, set a default for one of the inner keys. The result of merging
		// first and second defaults should be merged into this value, overwriting
		// it.
		val, err = val.Get("other_middle").WithDefault("should be overwritten")
		require.NoError(t, err, "couldn't set third, nested default")

		var m map[string]string
		run(t, TestCase{
			Value:    val,
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"middle":       "bottom",
				"other_middle": "other_bottom",
			},
			Populate: &m,
		})
		assert.Equal(t, map[string]string{
			"middle":       "bottom",
			"other_middle": "other_bottom",
		}, m, "unexpected result after populate")
	})

	t.Run("deep copy", func(t *testing.T) {
		t.Skip("TODO: mutations are visible to other callers")

		const key = "foobar"
		unmarshal := func() map[interface{}]interface{} {
			var m map[string]interface{}
			require.NoError(t, provider.Get(Root).Populate(&m), "Populate failed")
			return m["practical"].(map[interface{}]interface{})
		}

		before := unmarshal()
		require.NotContains(t, before, key, "precondition failed: key %q already in map", key)
		before[key] = "bazbing"

		after := unmarshal()
		assert.NotContains(t, after, key, "didn't deep copy config")
	})

	t.Run("named field", func(t *testing.T) {
		type cfg struct {
			Boring []string `yaml:"fun"`
		}

		var c cfg
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Equal(t, 2, len(c.Boring), "wrong number of fun cars")
	})

	t.Run("ignored field", func(t *testing.T) {
		type cfg struct {
			Boring []string `yaml:"-"`
		}

		var c cfg
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Empty(t, c.Boring, "wrong number of fun cars")
	})

	t.Run("omitempty field", func(t *testing.T) {
		t.Skip("TODO: doesn't handle omitempty fields")
		type cfg struct {
			Fun []string `yaml:",omitempty"`
		}

		var c cfg
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Equal(t, 2, len(c.Fun), "wrong number of fun cars")
	})

	t.Run("flow field", func(t *testing.T) {
		t.Skip("TODO: doesn't handle flow fields")
		type cfg struct {
			Fun []string `yaml:",flow"`
		}

		var c cfg
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Equal(t, 2, len(c.Fun), "wrong number of fun cars")
	})

	t.Run("inline field", func(t *testing.T) {
		t.Skip("TODO: doesn't handle inline fields")
		type cfg struct {
			Things struct {
				Fun []string
			} `yaml:",inline"`
		}

		var c cfg
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Equal(t, 2, len(c.Things.Fun), "wrong number of fun cars")
	})
}
