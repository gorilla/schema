// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"errors"
	"fmt"
	"reflect"
)

// NewDecoder returns a new Decoder.
func NewDecoder() *Decoder {
	return &Decoder{cache: newCache()}
}

// Decoder decodes values from a map[string][]string to a struct.
type Decoder struct {
	cache *cache
}

// RegisterConverter registers a converter function for a custom type.
func (d *Decoder) RegisterConverter(value interface{}, converterFunc Converter) {
	d.cache.conv[reflect.TypeOf(value)] = converterFunc
}

// Decode decodes a map[string][]string to a struct.
//
// The first parameter must be a pointer to a struct.
//
// The second parameter is a map, typically url.Values from an HTTP request.
// Keys are "paths" in dotted notation to the struct fields and nested structs.
//
// See the package documentation for a full explanation of the mechanics.
func (d *Decoder) Decode(dst interface{}, src map[string][]string) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("schema: interface must be a pointer to struct")
	}
	v = v.Elem()
	t := v.Type()
	errors := MultiError{}
	for path, values := range src {
		if parts, err := d.cache.parsePath(path, t); err == nil {
			if err = d.decode(v, path, parts, values); err != nil {
				errors[path] = err
			}
		} else {
			errors[path] = fmt.Errorf("schema: invalid path %q", path)
		}
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}

// decode fills a struct field using a parsed path.
func (d *Decoder) decode(v reflect.Value, path string, parts []pathPart,
	values []string) error {
	// Get the field walking the struct fields by index.
	for _, idx := range parts[0].path {
		if v.Type().Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(idx)
	}

	// Dereference if needed.
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if v.IsNil() {
			v.Set(reflect.New(t))
		}
		v = v.Elem()
	}

	// Slice of structs. Let's go recursive.
	if len(parts) > 1 {
		idx := parts[0].index
		if v.IsNil() || v.Len() < idx+1 {
			value := reflect.MakeSlice(t, idx+1, idx+1)
			if v.Len() < idx+1 {
				// Resize it.
				reflect.Copy(value, v)
			}
			v.Set(value)
		}
		return d.decode(v.Index(idx), path, parts[1:], values)
	}

	// Simple case.
	if t.Kind() == reflect.Slice {
		items := make([]reflect.Value, len(values))
		elemT := t.Elem()
		isPtrElem := elemT.Kind() == reflect.Ptr
		if isPtrElem {
			elemT = elemT.Elem()
		}
		conv := d.cache.conv[elemT]
		if conv == nil {
			return fmt.Errorf("schema: converter not found for %v", elemT)
		}
		for key, value := range values {
			if value == "" {
				// We are just ignoring empty values for now.
				continue
			} else if item := conv(value); item.IsValid() {
				if isPtrElem {
					ptr := reflect.New(elemT)
					ptr.Elem().Set(item)
					item = ptr
				}
				items[key] = item
			} else {
				// If a single value is invalid should we give up
				// or set a zero value?
				return ConversionError{path, key}
			}
		}
		value := reflect.Append(reflect.MakeSlice(t, 0, 0), items...)
		v.Set(value)
	} else {
		if values[0] == "" {
			// We are just ignoring empty values for now.
			return nil
		} else if conv := d.cache.conv[t]; conv != nil {
			if value := conv(values[0]); value.IsValid() {
				v.Set(value)
			} else {
				return ConversionError{path, -1}
			}
		} else {
			return fmt.Errorf("schema: converter not found for %v", t)
		}
	}
	return nil
}

// Errors ---------------------------------------------------------------------

// ConversionError stores information about a failed conversion.
type ConversionError struct {
	Key   string // key from the source map.
	Index int    // index for multi-value fields; -1 for single-value fields.
}

func (e ConversionError) Error() string {
	if e.Index < 0 {
		return fmt.Sprintf("schema: error converting value for %q", e.Key)
	}
	return fmt.Sprintf("schema: error converting value for index %d of %q",
		e.Index, e.Key)
}

// MultiError stores multiple decoding errors.
//
// Borrowed from the App Engine SDK.
type MultiError map[string]error

func (e MultiError) Error() string {
	s := ""
	for _, err := range e {
		s = err.Error()
		break
	}
	switch len(e) {
	case 0:
		return "(0 errors)"
	case 1:
		return s
	case 2:
		return s + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", s, len(e)-1)
}
