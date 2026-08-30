package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

// marshalSet renders r carrying only the fields parsing actually set, in
// struct declaration order.
//
// Encoding through a map would sort the keys, and tagging Result itself
// omitempty would change what the library emits — the golden corpus pins
// that — so walk the fields directly.
func marshalSet(r any, indent bool) ([]byte, error) {
	v := reflect.Indirect(reflect.ValueOf(r))
	t := v.Type()

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		if isUnset(v.Field(i)) {
			continue
		}
		val, err := json.Marshal(v.Field(i).Interface())
		if err != nil {
			return nil, err
		}
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(name)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(val)
	}
	buf.WriteByte('}')

	if !indent {
		return buf.Bytes(), nil
	}
	var out bytes.Buffer
	if err := json.Indent(&out, buf.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// isUnset reports whether a field still holds what parsing left untouched.
// An empty slice reads the same as an absent one here, and reflect.IsZero
// only catches the nil case.
func isUnset(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}
