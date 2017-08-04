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
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"gopkg.in/validator.v2"
	"gopkg.in/yaml.v2"
)

type fieldInfo struct {
	FieldName    string
	DefaultValue string
	Required     bool
}

func getFieldInfo(field reflect.StructField) fieldInfo {
	return fieldInfo{
		FieldName:    field.Tag.Get("yaml"),
		DefaultValue: field.Tag.Get("default"),
	}
}

func derefType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}

func convertSignedInts(src interface{}, dst *reflect.Value) error {
	switch t := src.(type) {
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64:
		i, err := strconv.ParseInt(fmt.Sprint(t), 10, 64)
		if err != nil {
			return err
		}

		if !dst.OverflowInt(i) {
			dst.SetInt(i)
			return nil
		}
	case uint64:
		if t <= math.MaxInt64 {
			dst.SetInt(int64(t))
			return nil
		}
	case uintptr:
		if t <= math.MaxInt64 && !dst.OverflowInt(int64(t)) {
			dst.SetInt(int64(t))
			return nil
		}
	case float32:
		if t >= math.MinInt64 && t <= math.MaxInt64 && !dst.OverflowInt(int64(t)) {
			dst.SetInt(int64(t))
			return nil
		}
	case float64:
		if t >= math.MinInt64 && t <= math.MaxInt64 && !dst.OverflowInt(int64(t)) {
			dst.SetInt(int64(t))
			return nil
		}
	case string:
		i, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return err
		}

		if !dst.OverflowInt(i) {
			dst.SetInt(i)
			return nil
		}
	}

	return fmt.Errorf("can't convert %q to integer type %q", fmt.Sprint(src), dst.Type())
}

func convertUnsignedInts(src interface{}, dst *reflect.Value) error {
	switch t := src.(type) {
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64:
		i, err := strconv.ParseInt(fmt.Sprint(t), 10, 64)
		if err != nil {
			return err
		}
		if i >= 0 && !dst.OverflowUint(uint64(i)) {
			dst.SetUint(uint64(i))
			return nil
		}
	case uint64:
		if !dst.OverflowUint(t) {
			dst.SetUint(t)
			return nil
		}
	case uintptr:
		if t <= math.MaxUint64 && !dst.OverflowUint(uint64(t)) {
			dst.SetUint(uint64(t))
			return nil
		}
	case float32:
		if t >= 0 && t <= math.MaxUint64 && !dst.OverflowUint(uint64(t)) {
			dst.SetUint(uint64(t))
			return nil
		}
	case float64:
		if t >= 0 && t <= math.MaxUint64 && !dst.OverflowUint(uint64(t)) {
			dst.SetUint(uint64(t))
			return nil
		}
	case string:
		i, err := strconv.ParseUint(t, 10, 64)
		if err != nil {
			return err
		}

		if !dst.OverflowUint(i) {
			dst.SetUint(i)
			return nil
		}
	}

	return fmt.Errorf("can't convert %q to unsigned integer type %q", fmt.Sprint(src), dst.Type())
}

func convertFloats(src interface{}, dst *reflect.Value) error {
	switch t := src.(type) {
	case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64, uintptr, float32:
		f, err := strconv.ParseFloat(fmt.Sprint(t), 64)
		dst.SetFloat(f)
		return err
	case float64:
		v := float64(t)
		if !dst.OverflowFloat(v) {
			dst.SetFloat(v)
			return nil
		}
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return err
		}

		if !dst.OverflowFloat(f) {
			dst.SetFloat(f)
			return nil
		}
	}

	return fmt.Errorf("can't convert %q to float type %q", fmt.Sprint(src), dst.Type())
}

func convertValueFromStruct(src interface{}, dst *reflect.Value) error {
	// The fieldType is probably a custom type here. We will try and set the fieldValue by
	// the custom type
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return convertSignedInts(src, dst)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return convertUnsignedInts(src, dst)

	case reflect.Float32, reflect.Float64:
		return convertFloats(src, dst)

	case reflect.Bool:
		v, err := strconv.ParseBool(fmt.Sprint(src))
		if err != nil {
			return err
		}

		dst.SetBool(v)

	case reflect.String:
		dst.SetString(fmt.Sprint(src))

	default:
		return fmt.Errorf("can't convert %q to %q", fmt.Sprint(src), dst.Type())
	}
	return nil
}

func convert(childKey string, value *reflect.Value, val interface{}) error {
	if val != nil {
		// First try to convert primitive type values, if convertValue wasn't able
		// to convert to primitive,try converting the value as a struct value
		if ret, err := convertValue(val, value.Type()); ret != nil {
			if err != nil {
				return errorWithKey(err, childKey)
			}

			value.Set(reflect.ValueOf(ret))
		} else {
			return errorWithKey(convertValueFromStruct(val, value), childKey)
		}
	}

	return nil
}

func addSeparator(key string) string {
	if key != "" {
		key += _separator
	}
	return key
}

func errorWithKey(err error, key string) error {
	return errors.Wrap(err, fmt.Sprintf("for key %q", key))
}

type decoder struct {
	*Value
	m map[interface{}]struct{}
}

func (d *decoder) getGlobalProvider() Provider {
	if d.root == nil {
		return d.provider
	}

	return d.root
}

// Sets value to a primitive type.
func (d *decoder) scalar(childKey string, value reflect.Value, def string) error {
	global := d.getGlobalProvider()
	var val interface{}

	// For primitive values, just get the value and set it into the field
	if v2 := global.Get(childKey); v2.HasValue() {
		val = v2.Value()
	} else if def != "" {
		val = def
	}

	return convert(childKey, &value, val)
}

// Collections of objects can be converted in the following way:
// 1. Maps can be decoded only to maps
// 2. Slices with arrays can be decoded one into another.
func checkCollections(src, dst reflect.Kind) error {
	err := errors.Errorf("can't convert %q to %q", src, dst)
	switch dst {
	case reflect.Map:
		if src != reflect.Map {
			return err
		}
	case reflect.Array, reflect.Slice:
		if src != reflect.Array && src != reflect.Slice {
			return err
		}
	}

	return nil
}

// Set value for a sequence type. Length of the collection is determined by the
// length of the underlying collection in a provider. It can be augmented
// further, by overriding values after the end, e.g.
//
// 	var x []int
// 	NewStaticProvider(map[string]interface{}{"a":[]int{0,1}, "a.2":2}.Populate(&x)
// 	fmt.Println(x)
//
// will print:
// [0 1 2]
func (d *decoder) sequence(childKey string, value reflect.Value) error {
	global := d.getGlobalProvider()
	slice := global.Get(childKey)

	s := slice.Value()
	sv := reflect.ValueOf(s)
	if err := checkCollections(sv.Kind(), value.Kind()); err != nil && slice.Value() != nil {
		return err
	}

	valueType := value.Type()
	destSlice := reflect.MakeSlice(valueType, 0, 4)

	// start looking for child values.
	elementType := derefType(valueType).Elem()
	childKey = addSeparator(childKey)

	for ai := 0; ; ai++ {
		arrayKey := childKey + strconv.Itoa(ai)

		// Iterate until we find first missing value.
		if v2 := global.Get(arrayKey); v2.Value() != nil {
			if err := checkCollections(reflect.TypeOf(v2.value).Kind(), elementType.Kind()); err != nil {
				return err
			}

			val := reflect.New(elementType).Elem()

			// Unmarshal current element.
			if err := d.unmarshal(arrayKey, val, ""); err != nil {
				return err
			}

			// Append element to the slice
			destSlice = reflect.Append(destSlice, reflect.Zero(elementType))
			destSlice.Index(ai).Set(val)
		} else {
			// Value in the middle was overridden,
			// but is missing, set it to the zero initialized value.
			if sv.IsValid() && sv.Len() > ai {

				// Append element to the slice
				destSlice = reflect.Append(destSlice, reflect.Zero(elementType))
				continue
			}

			break
		}
	}

	if destSlice.Len() > 0 {
		value.Set(destSlice)
	}

	return nil
}

// Set value for the array type. If a value for array item is not found,
// decoder will use a default one if present, or a zero constructed.
func (d *decoder) array(childKey string, value reflect.Value) error {
	global := d.getGlobalProvider()
	ar := global.Get(childKey)
	a := ar.Value()
	av := reflect.ValueOf(a)
	if err := checkCollections(av.Kind(), value.Kind()); err != nil && ar.Value() != nil {
		return err
	}

	valueType := value.Type()

	// start looking for child values.
	elementType := derefType(valueType).Elem()
	childKey = addSeparator(childKey)

	for ai := 0; ai < value.Len(); ai++ {
		arrayKey := childKey + strconv.Itoa(ai)

		// Iterate until we find first missing value.
		if v2 := global.Get(arrayKey); v2.Value() != nil {
			if err := checkCollections(reflect.TypeOf(v2.value).Kind(), elementType.Kind()); err != nil {
				return err
			}

			val := reflect.New(elementType).Elem()

			// Unmarshal current element.
			if err := d.unmarshal(arrayKey, val, ""); err != nil {
				return err
			}

			value.Index(ai).Set(val)
		} else if value.Index(ai).Kind() == reflect.Struct {
			if err := d.valueStruct(arrayKey, value.Index(ai).Addr().Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

// Sets value to a map type.
func (d *decoder) mapping(childKey string, value reflect.Value, def string) error {
	valueType := value.Type()
	global := d.getGlobalProvider()

	v := global.Get(childKey)
	if !v.HasValue() || v.Value() == nil {
		return nil
	}

	val := v.Value()
	if err := checkCollections(reflect.TypeOf(v.Value()).Kind(), valueType.Kind()); err != nil {
		return fmt.Errorf("expected map for key %q. actual type: %q", childKey, reflect.TypeOf(val))
	}

	destMap := reflect.ValueOf(reflect.MakeMap(valueType).Interface())

	childKey = addSeparator(childKey)

	rVal := reflect.ValueOf(val)
	for _, key := range rVal.MapKeys() {
		subKey := fmt.Sprintf("%v", key.Interface())
		if subKey == "" {
			// We can confuse an empty map key with a root element.
			return errorWithKey(errors.New("empty map key is ambiguous"), childKey)
		}

		itemValue := reflect.New(valueType.Elem()).Elem()

		// Try to unmarshal value and save it in the map.
		if err := d.unmarshal(childKey+subKey, itemValue, def); err != nil {
			return err
		}

		//TODO(alsam) do we need non scalar key types?
		keyVal := reflect.New(value.Type().Key()).Elem()
		if err := convert(childKey, &keyVal, key.Interface()); err != nil {
			return errors.Wrap(err, "key types conversion")
		}

		destMap.SetMapIndex(keyVal, itemValue)
	}

	value.Set(destMap)
	return nil
}

// Sets value to an interface type.
func (d *decoder) iface(key string, value reflect.Value, def string) error {
	v := d.getGlobalProvider().Get(key)

	if !v.HasValue() || v.Value() == nil {
		return nil
	}

	src := reflect.ValueOf(v.Value())
	if src.Type().Implements(value.Type()) {
		value.Set(src)
		return nil
	}

	return errorWithKey(fmt.Errorf("%q doesn't implement %q", src.Type(), value.Type()), key)
}

// Sets value to an object type.
func (d *decoder) object(childKey string, value reflect.Value) error {
	return d.valueStruct(childKey, value.Addr().Interface())
}

// Walk through the struct and start asking the providers for values at each key.
//
// - for individual values, we terminate
// - for array values, we start asking for indexes
// - for object values, we recurse.
func (d *decoder) valueStruct(key string, target interface{}) error {
	tarGet := reflect.Indirect(reflect.ValueOf(target))
	targetType := tarGet.Type()
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)

		// Check for the private field
		if field.PkgPath != "" || field.Anonymous {
			continue
		}

		fieldName := field.Name
		fieldInfo := getFieldInfo(field)
		if fieldInfo.FieldName != "" {
			fieldName = fieldInfo.FieldName
		}

		if key != "" {
			fieldName = key + _separator + fieldName
		}

		fieldValue := tarGet.Field(i)
		if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type()).Elem())
		}

		if err := d.unmarshal(fieldName, fieldValue, getFieldInfo(field).DefaultValue); err != nil {
			return err
		}
	}

	return errorWithKey(validator.Validate(target), key)
}

// If there is no value with name - leave it nil, otherwise allocate memory and set the value.
func (d *decoder) pointer(name string, value reflect.Value, def string) error {
	if !d.getGlobalProvider().Get(name).HasValue() {
		return nil
	}

	if value.IsNil() {
		value.Set(reflect.New(value.Type().Elem()))
	}

	return d.unmarshal(name, value.Elem(), def)
}

// JSON encoder will fail to serialize maps that don't have strings as keys
// so we are going to stringify them manually.
func jsonMap(v interface{}) interface{} {
	if reflect.TypeOf(v).Kind() == reflect.Map {
		rv := reflect.ValueOf(v)
		tmp := make(map[string]interface{}, len(rv.MapKeys()))
		for _, key := range rv.MapKeys() {
			tmp[fmt.Sprint(key.Interface())] = jsonMap(rv.MapIndex(key).Interface())
		}

		return tmp
	}

	return v
}

// Skip unmarshaling if there is no value in provider and default is empty.
func shouldSkip(value *Value, def string) bool {
	return !value.HasValue() && def == ""
}

// tryUnmarshallers checks if the value's type implements either one of standard
// interfaces in order:
// 1. `json.Unmarshaler`
// 2. `encoding.TextUnmarshaler`
// 3. `yaml.Unmarshaler`
// and tries it to populate the value.
func (d *decoder) tryUnmarshalers(key string, value reflect.Value, def string) (bool, error) {
	switch value.Kind() {
	// value.IsNil panics if value.Kind() is not equal to one of these constants.
	case reflect.Chan, reflect.Interface, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice:
		if value.IsNil() {
			value.Set(reflect.New(value.Type()).Elem())
		}
	}

	v := d.getGlobalProvider().Get(key)
	switch t := value.Addr().Interface().(type) {
	case json.Unmarshaler:
		// Skip unmarshaling if there is no value.
		if shouldSkip(&v, def) {
			return true, nil
		}

		// Use default if a value wasn't found.
		if !v.HasValue() {
			return true, errorWithKey(t.UnmarshalJSON([]byte(def)), key)
		}

		// Marshal the value first.
		b, err := json.Marshal(jsonMap(v.Value()))
		if err != nil {
			return true, errorWithKey(err, key)
		}

		// Unmarshal corresponding json.
		return true, errorWithKey(t.UnmarshalJSON(b), key)
	case encoding.TextUnmarshaler:
		if shouldSkip(&v, def) {
			return true, nil
		}

		// Use default if a value wasn't found.
		if v.HasValue() {
			def = v.String()
		}

		return true, errorWithKey(t.UnmarshalText([]byte(def)), key)
	case yaml.Unmarshaler:
		if shouldSkip(&v, def) {
			return true, nil
		}

		// Use default if a value wasn't found.
		if !v.HasValue() {
			return true, errorWithKey(yaml.Unmarshal([]byte(def), value.Addr().Interface()), key)
		}

		b, err := yaml.Marshal(v.Value())
		if err != nil {
			return true, err
		}

		return true, errorWithKey(yaml.Unmarshal(b, value.Addr().Interface()), key)
	}

	return false, nil
}

// Check if a value is a pointer and decoder set it before.
// TODO(alsam) print only elements in the loop, not all elements.
func (d *decoder) checkCycles(value reflect.Value) error {
	if value.Type().Comparable() {
		val := value.Interface()
		kind := value.Kind()
		if _, ok := d.m[val]; ok {
			if kind == reflect.Ptr && !value.IsNil() {
				buf := &bytes.Buffer{}
				for k := range d.m {
					fmt.Fprintf(buf, "%+v -> ", k)
				}

				fmt.Fprintf(buf, "%+v", value.Interface())
				return fmt.Errorf("cycles detected: %s", buf.String())
			}
		}

		d.m[val] = struct{}{}
	}

	return nil
}

// Dispatch un-marshalling functions based on the value type.
func (d *decoder) unmarshal(name string, value reflect.Value, def string) error {
	if err := d.checkCycles(value); err != nil {
		return errorWithKey(err, name)
	}

	// Check if a type can be unmarshaled directly.
	if ok, err := d.tryUnmarshalers(name, value, def); ok {
		return err
	}

	// Try to unmarshal each type separately.
	switch value.Kind() {
	case reflect.Invalid:
		return fmt.Errorf("invalid value type for key %s", name)

	// Permissive decoding: skip Chan/Func fields, instead of returning an
	// error. We don't have schemas and the caller probably has a big
	// aggregate struct and these fields will be set up separately later.
	case reflect.Chan, reflect.Func:
		return nil
	case reflect.Ptr:
		return d.pointer(name, value, def)
	case reflect.Struct:
		return d.object(name, value)
	case reflect.Array:
		return d.array(name, value)
	case reflect.Slice:
		return d.sequence(name, value)
	case reflect.Map:
		return d.mapping(name, value, def)
	case reflect.Interface:
		return d.iface(name, value, def)
	default:
		return d.scalar(name, value, def)
	}
}
