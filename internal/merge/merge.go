// Copyright (c) 2019 Uber Technologies, Inc.
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

package merge

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"go.uber.org/config/internal/unreachable"

	yaml "go.yaml.in/yaml/v3"
)

type (
	// YAML has three fundamental types. When unmarshaled into any, they're
	// represented like this. Mappings always use string keys: Normalize renders
	// any non-string key produced by go.yaml.in/yaml/v3 in its canonical string
	// form.
	mapping  = map[string]any
	sequence = []any
	scalar   = any
)

// YAML deep-merges any number of YAML sources, with later sources taking
// priority over earlier ones.
//
// Maps are deep-merged. For example,
//
//	{"one": 1, "two": 2} + {"one": 42, "three": 3}
//	== {"one": 42, "two": 2, "three": 3}
//
// Sequences are replaced. For example,
//
//	{"foo": [1, 2, 3]} + {"foo": [4, 5, 6]}
//	== {"foo": [4, 5, 6]}
//
// Attempting to merge mismatched types (e.g., merging a sequence into a map)
// replaces the old value with the new. Enabling strict mode returns an error
// instead.
func YAML(sources [][]byte, strict bool) (*bytes.Buffer, error) {
	var merged any
	var hasContent bool
	for _, r := range sources {
		d := yaml.NewDecoder(bytes.NewReader(r))
		// Unlike gopkg.in/yaml.v2's SetStrict, go.yaml.in/yaml/v3 always
		// rejects duplicate mapping keys, even in permissive mode. The strict
		// flag now only governs type-mismatch handling during the merge below.

		var contents any
		if err := d.Decode(&contents); err == io.EOF {
			// Skip empty and comment-only sources, which we should handle
			// differently from explicit nils.
			continue
		} else if err != nil {
			return nil, fmt.Errorf("couldn't decode source: %v", err)
		}

		// go.yaml.in/yaml/v3 decodes mappings as map[string]any (or map[any]any
		// when a key isn't a string); Normalize collapses both to the
		// map[string]any form the merge logic below expects.
		contents = Normalize(contents)

		hasContent = true
		pair, err := merge(merged, contents, strict)
		if err != nil {
			return nil, err // error is already descriptive enough
		}
		merged = pair
	}

	buf := &bytes.Buffer{}
	if !hasContent {
		// No sources had any content. To distinguish this from a source with just
		// an explicit top-level null, return an empty buffer.
		return buf, nil
	}
	enc := yaml.NewEncoder(buf)
	// Match the two-space indentation historically produced by
	// gopkg.in/yaml.v2; go.yaml.in/yaml/v3 defaults to four.
	enc.SetIndent(2)
	if err := enc.Encode(merged); err != nil {
		return nil, unreachable.Wrap(fmt.Errorf("couldn't re-serialize merged YAML: %v", err))
	}
	if err := enc.Close(); err != nil {
		return nil, unreachable.Wrap(fmt.Errorf("couldn't re-serialize merged YAML: %v", err))
	}
	return buf, nil
}

func merge(into, from any, strict bool) (any, error) {
	// It's possible to handle this with a mass of reflection, but we only need
	// to merge whole YAML files. Since we're always unmarshaling into any, we
	// only need to handle a few types. This ends up being cleaner if we just
	// handle each case explicitly.
	if into == nil {
		return from, nil
	}
	if from == nil {
		// Allow higher-priority YAML to explicitly nil out lower-priority entries.
		return nil, nil
	}
	if IsScalar(into) && IsScalar(from) {
		return from, nil
	}
	if IsSequence(into) && IsSequence(from) {
		return from, nil
	}
	if IsMapping(into) && IsMapping(from) {
		return mergeMapping(into.(mapping), from.(mapping), strict)
	}
	// YAML types don't match, so no merge is possible. For backward
	// compatibility, ignore mismatches unless we're in strict mode and return
	// the higher-priority value.
	if !strict {
		return from, nil
	}
	return nil, fmt.Errorf("can't merge a %s into a %s", describe(from), describe(into))
}

func mergeMapping(into, from mapping, strict bool) (mapping, error) {
	merged := make(mapping, len(into))
	for k, v := range into {
		merged[k] = v
	}
	for k := range from {
		m, err := merge(merged[k], from[k], strict)
		if err != nil {
			return nil, err
		}
		merged[k] = m
	}
	return merged, nil
}

// IsMapping reports whether a value is a mapping in YAML, represented as a
// map[string]any.
func IsMapping(i any) bool {
	_, is := i.(mapping)
	return is
}

// IsSequence reports whether a value is a sequence in YAML, represented as a
// []any.
func IsSequence(i any) bool {
	_, is := i.(sequence)
	return is
}

// IsScalar reports whether a value is a scalar in YAML.
func IsScalar(i any) bool {
	return !IsMapping(i) && !IsSequence(i)
}

func describe(i any) string {
	if IsMapping(i) {
		return "mapping"
	}
	if IsSequence(i) {
		return "sequence"
	}
	return "scalar"
}

// Normalize recursively rewrites the values decoded by go.yaml.in/yaml/v3 into
// this package's canonical representation: every mapping becomes a
// map[string]any. go.yaml.in/yaml/v3 decodes a mapping as map[string]any when
// its keys are all strings and as map[any]any otherwise; Normalize converts the
// latter, rendering each non-string key via StringifyKey.
func Normalize(val any) any {
	switch v := val.(type) {
	case map[string]any:
		for k, e := range v {
			v[k] = Normalize(e)
		}
		return v
	case map[any]any:
		m := make(mapping, len(v))
		for k, e := range v {
			m[StringifyKey(k)] = Normalize(e)
		}
		return m
	case []any:
		for i, e := range v {
			v[i] = Normalize(e)
		}
		return v
	default:
		return v
	}
}

// StringifyKey renders a scalar mapping key as the canonical string used by
// Normalize. Strings are returned unchanged; other scalars (integers, nil,
// Booleans, ...) are rendered in their canonical YAML form, so 1 becomes "1"
// and a nil key becomes "null".
func StringifyKey(key any) string {
	if s, ok := key.(string); ok {
		return s
	}
	bs, err := yaml.Marshal(key)
	if err != nil {
		// Unreachable for scalar keys, which always marshal cleanly.
		return fmt.Sprint(key)
	}
	return strings.TrimSpace(string(bs))
}
