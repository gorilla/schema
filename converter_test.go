// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schema

import (
	"reflect"
	"testing"
)

func TestBuiltinConverters(t *testing.T) {
	// Test convertBool
	if v := convertBool("on"); !v.IsValid() || !v.Bool() {
		t.Errorf("convertBool('on') expected true, got %v", v)
	}
	if v := convertBool("true"); !v.IsValid() || !v.Bool() {
		t.Errorf("convertBool('true') expected true, got %v", v)
	}
	if v := convertBool("1"); !v.IsValid() || !v.Bool() {
		t.Errorf("convertBool('1') expected true, got %v", v)
	}
	if v := convertBool("false"); !v.IsValid() || v.Bool() {
		t.Errorf("convertBool('false') expected false, got %v", v)
	}
	if v := convertBool("invalid"); v.IsValid() {
		t.Errorf("convertBool('invalid') expected invalid, got %v", v)
	}

	// Test convertFloat32 & convertFloat64
	if v := convertFloat32("3.14"); !v.IsValid() || v.Float() < 3.13 || v.Float() > 3.15 {
		t.Errorf("convertFloat32('3.14') got %v", v)
	}
	if v := convertFloat32("invalid"); v.IsValid() {
		t.Errorf("convertFloat32('invalid') expected invalid, got %v", v)
	}
	if v := convertFloat64("3.14159265"); !v.IsValid() || v.Float() < 3.14 {
		t.Errorf("convertFloat64('3.14159265') got %v", v)
	}
	if v := convertFloat64("invalid"); v.IsValid() {
		t.Errorf("convertFloat64('invalid') expected invalid, got %v", v)
	}

	// Test convertInt variants
	if v := convertInt("123"); !v.IsValid() || v.Int() != 123 {
		t.Errorf("convertInt('123') got %v", v)
	}
	if v := convertInt8("127"); !v.IsValid() || v.Int() != 127 {
		t.Errorf("convertInt8('127') got %v", v)
	}
	if v := convertInt16("32767"); !v.IsValid() || v.Int() != 32767 {
		t.Errorf("convertInt16('32767') got %v", v)
	}
	if v := convertInt32("2147483647"); !v.IsValid() || v.Int() != 2147483647 {
		t.Errorf("convertInt32('2147483647') got %v", v)
	}
	if v := convertInt64("9223372036854775807"); !v.IsValid() || v.Int() != 9223372036854775807 {
		t.Errorf("convertInt64 got %v", v)
	}
	if v := convertInt("invalid"); v.IsValid() {
		t.Errorf("convertInt('invalid') expected invalid, got %v", v)
	}

	// Test convertUint variants
	if v := convertUint("123"); !v.IsValid() || v.Uint() != 123 {
		t.Errorf("convertUint('123') got %v", v)
	}
	if v := convertUint8("255"); !v.IsValid() || v.Uint() != 255 {
		t.Errorf("convertUint8('255') got %v", v)
	}
	if v := convertUint16("65535"); !v.IsValid() || v.Uint() != 65535 {
		t.Errorf("convertUint16('65535') got %v", v)
	}
	if v := convertUint32("4294967295"); !v.IsValid() || v.Uint() != 4294967295 {
		t.Errorf("convertUint32('4294967295') got %v", v)
	}
	if v := convertUint64("18446744073709551615"); !v.IsValid() || v.Uint() != 18446744073709551615 {
		t.Errorf("convertUint64 got %v", v)
	}
	if v := convertUint("invalid"); v.IsValid() {
		t.Errorf("convertUint('invalid') expected invalid, got %v", v)
	}

	// Test convertString
	if v := convertString("hello"); !v.IsValid() || v.String() != "hello" {
		t.Errorf("convertString('hello') got %v", v)
	}
}

func TestConvertPointer(t *testing.T) {
	kinds := []struct {
		kind  reflect.Kind
		value string
		valid bool
	}{
		{boolType, "on", true},
		{boolType, "invalid", false},
		{float32Type, "1.5", true},
		{float32Type, "invalid", false},
		{float64Type, "2.5", true},
		{float64Type, "invalid", false},
		{intType, "10", true},
		{int8Type, "8", true},
		{int16Type, "16", true},
		{int32Type, "32", true},
		{int64Type, "64", true},
		{intType, "invalid", false},
		{stringType, "str", true},
		{uintType, "10", true},
		{uint8Type, "8", true},
		{uint16Type, "16", true},
		{uint32Type, "32", true},
		{uint64Type, "64", true},
		{uintType, "invalid", false},
		{reflect.Array, "arr", false},
	}

	for _, k := range kinds {
		v := convertPointer(k.kind, k.value)
		if k.valid && !v.IsValid() {
			t.Errorf("convertPointer(%v, %q) expected valid pointer, got invalid", k.kind, k.value)
		} else if !k.valid && v.IsValid() {
			t.Errorf("convertPointer(%v, %q) expected invalid, got %v", k.kind, k.value, v)
		}
	}
}
