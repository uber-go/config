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
	"fmt"
	"strings"
	"testing"

	. "go.uber.org/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

const (
	_base = `
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
`
	_override = `
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
`
)

type testCase struct {
	Value     Value
	HasValue  bool
	Interface interface{}
	Populate  interface{}
}

func run(t testing.TB, tt testCase) {
	assert.Equal(t, tt.HasValue, tt.Value.HasValue(), "unexpected result from HasValue")
	assert.Equal(t, tt.Interface, tt.Value.Value(), "unexpected result from Value")
	require.NoError(t, tt.Value.Populate(tt.Populate), "error populating")
}

func runCommonTests(t *testing.T, provider Provider) {
	t.Run("missing", func(t *testing.T) {
		var s string

		t.Run("top level", func(t *testing.T) {
			run(t, testCase{
				Value:     provider.Get("not_a_key"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("subkey", func(t *testing.T) {
			run(t, testCase{
				Value:     provider.Get("practical.cadillac"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("subkey of sequence", func(t *testing.T) {
			// Accessing a subkey of a sequence is meaningless.
			run(t, testCase{
				Value:     provider.Get("fun.not_there"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})
	})

	t.Run("nil", func(t *testing.T) {
		var s string
		run(t, testCase{
			Value:     provider.Get("nothing"),
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after Populate")
	})

	t.Run("scalar", func(t *testing.T) {
		var s string
		run(t, testCase{
			Value:     provider.Get("practical.toyota"),
			HasValue:  true,
			Interface: "camry",
			Populate:  &s,
		})
		assert.Equal(t, "camry", s, "unexpected result after Populate")
	})

	t.Run("map", func(t *testing.T) {
		var m map[string]string
		run(t, testCase{
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
		run(t, testCase{
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
		run(t, testCase{
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
		var s string
		run(t, testCase{
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
		var s []string
		run(t, testCase{
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
		var m map[string]string
		run(t, testCase{
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
		run(t, testCase{
			Value:     provider.Get("antique_mapping_empty"),
			HasValue:  true,
			Interface: map[interface{}]interface{}{"ford": "model_t"},
			Populate:  &m,
		})
		assert.Equal(t, map[string]string{"ford": "model_t"}, m, "unexpected result after Populate")
	})

	t.Run("anchors and native merge", func(t *testing.T) {
		var m map[string]string
		run(t, testCase{
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
		run(t, testCase{
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
		run(t, testCase{
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
		run(t, testCase{
			Value:     val,
			HasValue:  true,
			Interface: "civic",
			Populate:  &s,
		})
		assert.Equal(t, "civic", s, "unexpected result after populate")
	})

	t.Run("default overriden by nil", func(t *testing.T) {
		var s string
		val, err := provider.Get("nothing").WithDefault("something")
		require.NoError(t, err, "couldn't set default")
		run(t, testCase{
			Value:     val,
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after populate")
	})

	t.Run("default merges maps", func(t *testing.T) {
		var m map[string]string
		val, err := provider.Get("practical").WithDefault(map[string]string{
			"ford":   "fiesta",  // new key
			"toyota": "corolla", // key present and set to "camry"
		})
		require.NoError(t, err, "couldn't set default")
		run(t, testCase{
			Value:    val,
			HasValue: true,
			Interface: map[interface{}]interface{}{
				"honda":  "civic",
				"toyota": "camry",
				"nissan": "altima",
				"ford":   "fiesta",
			},
			Populate: &m,
		})
		assert.Equal(t, map[string]string{
			"honda":  "civic",
			"toyota": "camry",
			"nissan": "altima",
			"ford":   "fiesta",
		}, m, "unexpected result after populate")
	})

	t.Run("default replaces sequences", func(t *testing.T) {
		var s []string
		val, err := provider.Get("fun").WithDefault([]string{"delorean"})
		require.NoError(t, err, "couldn't set default")
		run(t, testCase{
			Value:     val,
			HasValue:  true,
			Interface: []interface{}{"maserati", "lamborghini"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"maserati", "lamborghini"}, s, "unexpected result after populate")
	})

	t.Run("chained defaults", func(t *testing.T) {
		// Each call to WithDefault should merge all existing config into the
		// default, then use the result. This means that repeated calls to
		// WithDefault should deep-merge all the supplied defaults, with the last
		// call to WithDefault having the lowest priority.

		// First, set a default.
		val, err := provider.Get("top").WithDefault(map[string]string{"middle": "bottom"})
		require.NoError(t, err, "couldn't set first default")

		// Second, set another default with a different key. First default should
		// be deep-merged into this one.
		val, err = val.WithDefault(map[string]string{"other_middle": "other_bottom"})
		require.NoError(t, err, "couldn't set second default")

		// Last, set a default for the new key. The result of merging first and
		// second defaults should be merged into this value, overwriting it.
		val, err = val.WithDefault(map[string]string{"other_middle": "should be overwritten"})
		require.NoError(t, err, "couldn't set third default")

		var m map[string]string
		run(t, testCase{
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
		// Regression test for https://github.com/uber-go/config/issues/76.
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
		c := struct {
			Toyota string
			Honda  string
			Datsun string `yaml:"nissan"`
		}{}
		require.NoError(t, provider.Get("practical").Populate(&c))
		assert.Equal(t, "camry", c.Toyota, "wrong Toyota")
		assert.Equal(t, "altima", c.Datsun, "wrong Datsun (aka Nissan)")
	})

	t.Run("omitempty field", func(t *testing.T) {
		c := struct {
			Toyota string
			Honda  string
			Nissan string `yaml:",omitempty"`
		}{}
		require.NoError(t, provider.Get("practical").Populate(&c))
		assert.Equal(t, "altima", c.Nissan, "wrong Nissan")
	})

	t.Run("flow field", func(t *testing.T) {
		c := struct {
			Toyota string
			Honda  string
			Nissan string `yaml:",flow"`
		}{}
		require.NoError(t, provider.Get("practical").Populate(&c))
		assert.Equal(t, "altima", c.Nissan, "wrong Nissan")
	})

	t.Run("inline field", func(t *testing.T) {
		c := struct {
			Humans struct {
				Driver    string
				Passenger string
				Backseat  []string
			} `yaml:",inline"`
		}{}
		require.NoError(t, provider.Get("occupants.honda").Populate(&c))
		assert.Equal(t, 1, len(c.Humans.Backseat), "wrong number of backseat occupants")
	})
}

func TestPermissiveYAML(t *testing.T) {
	provider, err := NewYAML(
		Source(strings.NewReader(_base)),
		Source(strings.NewReader(_override)),
		Permissive(),
	)
	require.NoError(t, err, "couldn't create provider")
	assert.Equal(t, "YAML", provider.Get(Root).Source(), "wrong source")
	assert.Equal(t, "<nil>", provider.Get("nothing").String(), "wrong string representation")

	runCommonTests(t, provider)

	t.Run("ignore type mismatch during merge", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("mismatch: foo")),   // scalar
			Source(strings.NewReader("mismatch: [foo]")), // sequence
			Permissive(),
		)
		require.NoError(t, err, "couldn't create permissive provider with type mismatch")

		var s []string
		run(t, testCase{
			Value:     provider.Get("mismatch"),
			HasValue:  true,
			Interface: []interface{}{"foo"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"foo"}, s, "unexpected result after populate")
	})

	t.Run("ignores type mismatches during WithDefault", func(t *testing.T) {
		// Since defaults are effectively the lowest-priority config, their types
		// should be ignored when necessary.
		provider, err := NewYAML(
			Source(strings.NewReader("mismatch: foo")), // scalar
			Permissive(),
		)
		require.NoError(t, err, "couldn't create provider")
		val, err := provider.Get("mismatch").WithDefault([]string{"foo"}) // sequence
		assert.NoError(t, err, "error on type mismatch in permissive mode")
		assert.Equal(t, "foo", val.Value(), "wrong merged output")
	})

	t.Run("ignore duplicate keys", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("dupe: foo\ndupe: bar")),
			Permissive(),
		)
		require.NoError(t, err, "couldn't create provider")

		var s string
		run(t, testCase{
			Value:     provider.Get("dupe"),
			HasValue:  true,
			Interface: "bar",
			Populate:  &s,
		})
		assert.Equal(t, "bar", s, "unexpected result after populate")
	})

	t.Run("ignores extra data during Populate", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("foo: bar\nbaz: quux")),
			Permissive(),
		)
		require.NoError(t, err, "couldn't create provider")

		c := struct {
			Foo string
		}{}
		require.NoError(t, provider.Get(Root).Populate(&c), "populate failed")
		assert.Equal(t, "bar", c.Foo, "unexpected value")
	})

	t.Run("may provide ignored field", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("toyota: camry")),
			Permissive(),
		)
		require.NoError(t, err, "couldn't create provider")

		c := struct {
			Toyota string `yaml:"-"`
		}{}
		require.NoError(t, provider.Get(Root).Populate(&c))
		assert.Empty(t, c.Toyota, "should ignore Toyota")
	})
}

func TestStrictYAML(t *testing.T) {
	provider, err := NewYAML(
		Source(strings.NewReader(_base)),
		Source(strings.NewReader(_override)),
	)
	require.NoError(t, err, "couldn't create provider")
	assert.Equal(t, "YAML", provider.Get(Root).Source(), "wrong source")
	assert.Equal(t, "<nil>", provider.Get("nothing").String(), "wrong string representation")

	runCommonTests(t, provider)

	t.Run("fail on type mismatch during merge", func(t *testing.T) {
		_, err := NewYAML(
			Source(strings.NewReader("mismatch: foo")),   // scalar
			Source(strings.NewReader("mismatch: [foo]")), // sequence
		)
		require.Error(t, err, "couldn't create permissive provider with type mismatch")
		assert.Contains(t, err.Error(), "couldn't merge", "unexpected error message")
	})

	t.Run("fail on type mismatches during WithDefault", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("mismatch: foo")), // scalar
		)
		require.NoError(t, err, "couldn't create provider")

		_, err = provider.Get("mismatch").WithDefault([]string{"foo"}) // sequence
		assert.Error(t, err, "success on type mismatch in strict mode")
		assert.Contains(t, err.Error(), "can't merge", "unexpected error message")
	})

	t.Run("fail on duplicate keys", func(t *testing.T) {
		_, err := NewYAML(
			Source(strings.NewReader("dupe: foo\ndupe: bar")),
		)
		require.Error(t, err, "created strict provider with type mismatch")
		assert.Contains(t, err.Error(), `key "dupe" already set in map`, "unexpected error message")
	})

	t.Run("allow missing data during Populate", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("foo: bar")),
		)
		require.NoError(t, err, "couldn't create provider")

		c := struct {
			Foo   string
			Unset string
		}{}
		err = provider.Get(Root).Populate(&c)
		require.NoError(t, err, "populate failed")
		assert.Equal(t, c.Foo, "bar", "unexpected value for set field")
		assert.Zero(t, c.Unset, "expected unset field to be zero value")
	})

	t.Run("fail on extra data during Populate", func(t *testing.T) {
		provider, err := NewYAML(
			Source(strings.NewReader("foo: bar\nbaz: quux")),
		)
		require.NoError(t, err, "couldn't create provider")

		c := struct {
			Foo string
		}{}
		err = provider.Get(Root).Populate(&c)
		require.Error(t, err, "populate succeeded")
		assert.Contains(t, err.Error(), "field baz not found in type struct", "unexpected error message")
	})

	t.Run("must not provide ignored fields", func(t *testing.T) {
		provider, err := NewYAML(Source(strings.NewReader("toyota: camry")))
		require.NoError(t, err, "couldn't create provider")

		c := struct {
			Toyota string `yaml:"-"`
		}{}
		err = provider.Get(Root).Populate(&c)
		require.Error(t, err, "expected error")
		assert.Contains(t, err.Error(), "field toyota not found in type", "unexpected error message")
	})
}

func TestStaticFromYAML(t *testing.T) {
	// Since we have a common test suite, we may as well use it to exercise the
	// static provider too.
	var base, override interface{}
	require.NoError(t, yaml.Unmarshal([]byte(_base), &base), "couldn't unmarshal base YAML")
	require.NoError(t, yaml.Unmarshal([]byte(_override), &override), "couldn't unmarshal base YAML")

	p, err := NewYAML(Static(base), Static(override))
	require.NoError(t, err, "couldn't construct provider")
	runCommonTests(t, p)
}

func TestEmptySources(t *testing.T) {
	const (
		key = "foo"
		val = "bar"
	)
	full := fmt.Sprintf("%s: %s", key, val)
	empty := ""
	comment := "# just a comment"

	runTests := func(t *testing.T, newProvider func([]string) Provider) {
		tests := []struct {
			desc    string
			sources []string
			expect  interface{}
		}{
			{"no sources", []string{}, nil},
			{"empty base", []string{empty, full}, val},
			{"empty override", []string{full, empty}, val},
			{"empty base and override", []string{empty, empty}, nil},
			{"comment-only base", []string{comment, full}, val},
			{"comment-only override", []string{full, comment}, val},
			{"empty base and comment-only override", []string{empty, comment}, nil},
			{"comment-only base and empty override", []string{comment, empty}, nil},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				p := newProvider(tt.sources)
				assert.Equal(t, tt.expect, p.Get("foo").Value(), "incorrect merged result")

				d, err := p.Get("not_there").WithDefault(42)
				require.NoError(t, err, "failed to set default")
				assert.Equal(t, 42, d.Value(), "incorrect defaulted value")
			})
		}
	}

	t.Run("NewYAML", func(t *testing.T) {
		runTests(t, func(sources []string) Provider {
			opts := make([]YAMLOption, len(sources))
			for i := range sources {
				opts[i] = Source(strings.NewReader(sources[i]))
			}
			p, err := NewYAML(opts...)
			require.NoError(t, err, "failed to create provider")
			return p
		})
	})

	t.Run("NewProviderGroup", func(t *testing.T) {
		runTests(t, func(sources []string) Provider {
			providers := make([]Provider, len(sources))
			for i := range sources {
				var err error
				providers[i], err = NewYAML(Source(strings.NewReader(sources[i])))
				require.NoError(t, err, "failed to create provider")
			}
			p, err := NewProviderGroup("foo", providers...)
			require.NoError(t, err, "failed to create provider")
			return p
		})
	})
}

func TestNullSources(t *testing.T) {
	const (
		key = "foo"
		val = "bar"
	)
	full := fmt.Sprintf("%s: %s", key, val)
	empty := ""
	null := "~"

	runTests := func(t *testing.T, newProvider func([]string) Provider) {
		tests := []struct {
			desc    string
			sources []string
			expect  interface{}
		}{
			{"null base", []string{null, empty, full}, val},
			{"null override", []string{full, empty, null}, nil},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				p := newProvider(tt.sources)
				assert.Equal(t, tt.expect, p.Get("foo").Value(), "incorrect merged result")

				d, err := p.Get("not_there").WithDefault(42)
				require.NoError(t, err, "failed to set default")
				// Since there's an explicit nil in the initial sources, calls to
				// WithDefault should have no effect.
				assert.Equal(t, nil, d.Value(), "default should have no effect")
			})
		}
	}

	t.Run("NewYAML", func(t *testing.T) {
		runTests(t, func(sources []string) Provider {
			opts := make([]YAMLOption, len(sources))
			for i := range sources {
				opts[i] = Source(strings.NewReader(sources[i]))
			}
			p, err := NewYAML(opts...)
			require.NoError(t, err, "failed to create provider")
			return p
		})
	})

	t.Run("NewProviderGroup", func(t *testing.T) {
		runTests(t, func(sources []string) Provider {
			providers := make([]Provider, len(sources))
			for i := range sources {
				var err error
				providers[i], err = NewYAML(Source(strings.NewReader(sources[i])))
				require.NoError(t, err, "failed to create provider")
			}
			p, err := NewProviderGroup("foo", providers...)
			require.NoError(t, err, "failed to create provider")
			return p
		})
	})
}

func TestRawSources(t *testing.T) {
	env := map[string]string{
		"ZONE": "west1",
	}
	lookup := func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}

	base := `zone: $ZONE`
	secrets := `secret: abc$ZONE`

	cfg := struct {
		Zone   string
		Secret string
	}{}

	provider, err := NewYAML(
		Source(strings.NewReader(base)),
		RawSource(strings.NewReader(secrets)),
		Expand(lookup),
	)
	require.NoError(t, err, "failed to create provider")
	assert.NoError(t, provider.Get(Root).Populate(&cfg), "failed to populate config struct")
	assert.Equal(t, "west1", cfg.Zone, "unexpected zone")
	assert.Equal(t, "abc$ZONE", cfg.Secret, "unexpected secret")
}
