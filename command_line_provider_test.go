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
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandLineProvider_Roles(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	var s StringSlice
	assert.Equal(t, "StringSlice", s.Type())
	f.Var(&s, "roles", "")

	c, err := NewCommandLineProvider(f, []string{`--roles=a,b,c"d"`})
	require.NoError(t, err, "Can't create a command line provider")

	v := c.Get("roles")
	require.True(t, v.HasValue())
	var roles []string
	require.NoError(t, v.Populate(&roles))
	assert.Equal(t, []string{"a", "b", `c"d"`}, roles)
}

func TestCommandLineProvider_Default(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	f.String("killerFeature", "minesweeper", "Start->Games->Minesweeper")

	c, err := NewCommandLineProvider(f, nil)
	require.NoError(t, err, "Can't create a command line provider")

	v := c.Get("killerFeature")
	require.True(t, v.HasValue())
	assert.Equal(t, "minesweeper", v.String())
}

func TestCommandLineProvider_Conversion(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	f.String("dozen", "14", " that number of rolls being allowed to the purchaser of a dozen")

	c, err := NewCommandLineProvider(f, []string{"--dozen=13"})
	require.NoError(t, err, "Can't create a command line provider")

	v := c.Get("dozen")
	require.True(t, v.HasValue())
	assert.Equal(t, "13", v.Value())
}

func TestCommandLineProvider_ErrorsOnUnknownFlags(t *testing.T) {
	t.Parallel()

	_, err := NewCommandLineProvider(flag.NewFlagSet("", flag.ContinueOnError), []string{"--boom"})
	require.Error(t, err, "Shouldn't create a command line provider")
	assert.Contains(t, err.Error(), "unknown flag: --boom")
}

func TestCommandLineProvider_Name(t *testing.T) {
	t.Parallel()

	p, err := NewCommandLineProvider(flag.NewFlagSet("", flag.PanicOnError), nil)
	require.NoError(t, err, "Can't create a command line provider")
	assert.Equal(t, "cmd", p.Name())
}

func TestCommandLineProvider_RepeatingArguments(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	f.Int("count", 1, "If I had a million dollars")

	c, err := NewCommandLineProvider(f, []string{"--count=2", "--count=3"})
	require.NoError(t, err)

	v := c.Get("count")
	require.True(t, v.HasValue())
	assert.Equal(t, "3", v.String())
}

func TestCommandLineProvider_NestedValues(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	f.String("Name.Source", "default", "Data provider source")
	f.Var(&StringSlice{}, "Name.Array", "Example of a nested array")

	c, err := NewCommandLineProvider(f, []string{"--Name.Source=chocolateFactory", "--Name.Array=cookie, candy,brandy"})
	require.NoError(t, err)

	type Wonka struct {
		Source string
		Array  []string
	}

	type Willy struct {
		Name Wonka
	}

	s, err := NewStaticProvider(Willy{Name: Wonka{Source: "staticFactory"}})
	require.NoError(t, err, "Can't create a static provider")

	g := NewProviderGroup("group", s, c)
	require.NoError(t, err, "Can't group providers")

	var v Willy
	require.NoError(t, g.Get(Root).Populate(&v))
	assert.Equal(t, Willy{Name: Wonka{
		Source: "chocolateFactory",
		Array:  []string{"cookie", " candy", "brandy"},
	}}, v)
}

func TestCommandLineProvider_OverlappingFlags(t *testing.T) {
	t.Parallel()

	f := flag.NewFlagSet("", flag.PanicOnError)
	f.String("Sushi.Tools.1", "Saibashi", "Chopsticks are extremely helpful!")
	f.Var(&StringSlice{}, "Sushi.Tools", "yolo")

	c, err := NewCommandLineProvider(f, []string{"--Sushi.Tools.1=Fork", "--Sushi.Tools=Hocho, Hashi"})
	require.NoError(t, err, "Can't create a command line provider")

	type Sushi struct {
		Tools []string
	}

	var v Sushi
	require.NoError(t, c.Get("Sushi").Populate(&v))
	assert.Equal(t, Sushi{
		Tools: []string{"Hocho", "Fork"},
	}, v)
}
