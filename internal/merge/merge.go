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

package merge

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"go.uber.org/config/internal/unreachable"

	yaml "gopkg.in/yaml.v2"
)

type (
	// YAML has three fundamental types. When unmarshaled into interface{},
	// they're represented like this.
	mapping  = map[interface{}]interface{}
	sequence = []interface{}
	scalar   = interface{}
)

// To distinguish between empty readers and explicit nils, we need to know
// whether a reader has any non-comment content.
//
// FIXME
func hasValues(content []byte) (bool, error) {
	var found bool
	scanner := bufio.NewScanner(bytes.NewBuffer(content))
	for scanner.Scan() && !found {
		line := strings.TrimSpace(scanner.Text())
		// http://yaml.org/spec/1.2/spec.html#comment: outside scalar content,
		// whitespace-only lines are treated as comments. Since starting a block
		// of scalar content requires some non-comment text, we can skip
		// whitespace-only lines here.
		if len(line) == 0 {
			continue
		}
		// http://yaml.org/spec/1.2/spec.html#comment: all full-line comments
		// start with # and there are no block comments.
		if strings.HasPrefix(line, "#") {
			continue
		}
		found = true
	}
	if err := scanner.Err(); err != nil {
		return false, unreachable.Wrap(err)
	}
	return found, nil
}

// YAML deep-merges any number of YAML sources, with later sources taking
// priority over earlier ones.
//
// Maps are deep-merged. For example,
//   {"one": 1, "two": 2} + {"one": 42, "three": 3}
//   == {"one": 42, "two": 2, "three": 3}
// Sequences are replaced. For example,
//   {"foo": [1, 2, 3]} + {"foo": [4, 5, 6]}
//   == {"foo": [4, 5, 6]}
//
// In non-strict mode, duplicate map keys are allowed within a single source,
// with later values overwriting previous ones. Attempting to merge
// mismatched types (e.g., merging a sequence into a map) replaces the old
// value with the new.
//
// Enabling strict mode returns errors in both of the above cases.
func YAML(sources []io.Reader, strict bool) (*bytes.Buffer, error) {
	var merged interface{}
	var hasContent bool
	for _, r := range sources {
		d := yaml.NewDecoder(r)
		d.SetStrict(strict)

		var contents interface{}
		if err := d.Decode(&contents); err == io.EOF {
			// Skip empty sources, which we should handle differently from explicit
			// nils.
			continue
		} else if err != nil {
			return nil, fmt.Errorf("couldn't decode source: %v", err)
		}

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
		// a top-level nil, return an empty buffer.
		return buf, nil
	}
	enc := yaml.NewEncoder(buf)
	if err := enc.Encode(merged); err != nil {
		return nil, unreachable.Wrap(fmt.Errorf("couldn't re-serialize merged YAML: %v", err))
	}
	return buf, nil
}

func merge(into, from interface{}, strict bool) (interface{}, error) {
	// It's possible to handle this with a mass of reflection, but we only need
	// to merge whole YAML files. Since we're always unmarshaling into
	// interface{}, we only need to handle a few types. This ends up being
	// cleaner if we just handle each case explicitly.
	if into == nil {
		return from, nil
	}
	if from == nil {
		// Allow higher-priority YAML to explicitly nil out lower-priority entries.
		return nil, nil
	}
	if isScalar(into) && isScalar(from) {
		return from, nil
	}
	if isSequence(into) && isSequence(from) {
		return from, nil
	}
	if isMapping(into) && isMapping(from) {
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

func isMapping(i interface{}) bool {
	_, is := i.(mapping)
	return is
}

func isSequence(i interface{}) bool {
	_, is := i.(sequence)
	return is
}

func isScalar(i interface{}) bool {
	return !isMapping(i) && !isSequence(i)
}

func describe(i interface{}) string {
	if isMapping(i) {
		return "mapping"
	}
	if isSequence(i) {
		return "sequence"
	}
	return "scalar"
}
