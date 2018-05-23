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
practical:
  toyota: camry
  honda: accord
occupants:
  honda:
    driver: jane
    backseat: [nate]
`)
	second := strings.NewReader(`
fun:
  - maserati
  - lamborghini
practical:
  honda: civic
  nissan: altima
occupants:
  honda:
    passenger: arthur
    backseat: [nora]
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

	run := func(tt TestCase) {
		assert.Equal(t, tt.HasValue, tt.Value.HasValue(), "unexpected result from HasValue")
		assert.Equal(t, tt.Interface, tt.Value.Value(), "unexpected result from Value")
		require.NoError(t, tt.Value.Populate(tt.Populate), "error populating")
	}

	t.Run("missing", func(t *testing.T) {
		var s string

		t.Run("top level", func(t *testing.T) {
			run(TestCase{
				Value:     provider.Get("not_a_key"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("subkey", func(t *testing.T) {
			run(TestCase{
				Value:     provider.Get("practical.cadillac"),
				HasValue:  false,
				Interface: nil,
				Populate:  &s,
			})
			assert.Equal(t, "", s, "unexpected result after Populate")
		})

		t.Run("descend into slice", func(t *testing.T) {
			run(TestCase{
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
		run(TestCase{
			Value:     provider.Get("nothing"),
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after Populate")
	})

	t.Run("scalar", func(t *testing.T) {
		var s string
		run(TestCase{
			Value:     provider.Get("practical.toyota"),
			HasValue:  true,
			Interface: "camry",
			Populate:  &s,
		})
		assert.Equal(t, "camry", s, "unexpected result after Populate")
	})

	t.Run("map", func(t *testing.T) {
		var m map[string]string
		run(TestCase{
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
		run(TestCase{
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
		run(TestCase{
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

	t.Run("multiple gets", func(t *testing.T) {
		var s string
		run(TestCase{
			Value:     provider.Get("occupants").Get("honda").Get("driver"),
			HasValue:  true,
			Interface: "jane",
			Populate:  &s,
		})
		assert.Equal(t, "jane", s, "unexpected result after populate")
	})

	t.Run("default missing value", func(t *testing.T) {
		var s string
		run(TestCase{
			Value:     provider.Get("occupants").Get("honda").Get("driver"),
			HasValue:  true,
			Interface: "jane",
			Populate:  &s,
		})
		assert.Equal(t, "jane", s, "unexpected result after populate")
	})

	t.Run("default overriden by scalar", func(t *testing.T) {
		var s string
		val, err := provider.Get("practical.honda").WithDefault("CRV")
		require.NoError(t, err, "couldn't set default")
		run(TestCase{
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
		run(TestCase{
			Value:     val,
			HasValue:  true,
			Interface: nil,
			Populate:  &s,
		})
		assert.Equal(t, "", s, "unexpected result after populate")
	})

	t.Run("default merges maps", func(t *testing.T) {
		var m map[string]string
		val, err := provider.Get("practical").WithDefault(map[string]string{"ford": "fiesta"})
		require.NoError(t, err, "couldn't set default")
		run(TestCase{
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
		run(TestCase{
			Value:     val,
			HasValue:  true,
			Interface: []interface{}{"maserati", "lamborghini"},
			Populate:  &s,
		})
		assert.Equal(t, []string{"maserati", "lamborghini"}, s, "unexpected result after populate")
	})

	t.Run("default ignores type mismatches", func(t *testing.T) {
		// TODO: We should return an error in strict mode.
		// fun is a sequence, so we shouldn't be able to merge in a mapping
		val, err := provider.Get("fun").WithDefault(map[string]string{"dodge": "viper"})
		assert.NoError(t, err, "error on type mismatch in permissive mode")
		assert.Equal(t, []interface{}{"maserati", "lamborghini"}, val.Value(), "wrong merged output")
	})

	t.Run("chained defaults", func(t *testing.T) {
		// Defaults should be FIFO: once we've set a default value, we can't later
		// change our minds and set a different default.
		val, err := provider.Get("top").WithDefault(map[string]string{"middle": "bottom"})
		require.NoError(t, err, "couldn't set first default")
		val, err = val.Get("middle").WithDefault("boom")
		require.NoError(t, err, "couldn't set second default")

		var s string
		run(TestCase{
			Value:     val,
			HasValue:  true,
			Interface: "bottom",
			Populate:  &s,
		})
		assert.Equal(t, "bottom", s, "unexpected result after populate")
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
		assert.Equal(t, 0, len(c.Boring), "wrong number of fun cars")
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
