package schema

import (
	"reflect"
	"testing"
)

type E1 struct {
	F01 int     `schema:"f01"`
	F02 int     `schema:"-"`
	F03 string  `schema:"f03"`
	F04 string  `schema:"f04,omitempty"`
	F05 bool    `schema:"f05"`
	F06 bool    `schema:"f06"`
	F07 *string `schema:"f07"`
	F08 *int8   `schema:"f08"`
	F09 float64 `schema:"f09"`
	F10 func()  `schema:"f10"`
	F11 inner
}
type inner struct {
	F12 int
}

func TestFilled(t *testing.T) {
	f07 := "seven"
	var f08 int8 = 8
	s := &E1{
		F01: 1,
		F02: 2,
		F03: "three",
		F04: "four",
		F05: true,
		F06: false,
		F07: &f07,
		F08: &f08,
		F09: 1.618,
		F10: func() {},
		F11: inner{12},
	}

	vals := make(map[string][]string)
	errs := NewEncoder().Encode(s, vals)

	valExists(t, "f01", "1", vals)
	valNotExists(t, "f02", vals)
	valExists(t, "f03", "three", vals)
	valExists(t, "f05", "true", vals)
	valExists(t, "f06", "false", vals)
	valExists(t, "f07", "seven", vals)
	valExists(t, "f08", "8", vals)
	valExists(t, "f09", "1.618000", vals)
	valExists(t, "F12", "12", vals)

	emptyErr := MultiError{}
	if errs.Error() == emptyErr.Error() {
		t.Errorf("Expected error got %v", errs)
	}
}

type Aa int

type E3 struct {
	F01 bool    `schema:"f01"`
	F02 float32 `schema:"f02"`
	F03 float64 `schema:"f03"`
	F04 int     `schema:"f04"`
	F05 int8    `schema:"f05"`
	F06 int16   `schema:"f06"`
	F07 int32   `schema:"f07"`
	F08 int64   `schema:"f08"`
	F09 string  `schema:"f09"`
	F10 uint    `schema:"f10"`
	F11 uint8   `schema:"f11"`
	F12 uint16  `schema:"f12"`
	F13 uint32  `schema:"f13"`
	F14 uint64  `schema:"f14"`
	F15 Aa      `schema:"f15"`
}

// Test compatibility with default decoder types.
func TestCompat(t *testing.T) {
	src := &E3{
		F01: true,
		F02: 4.2,
		F03: 4.3,
		F04: -42,
		F05: -43,
		F06: -44,
		F07: -45,
		F08: -46,
		F09: "foo",
		F10: 42,
		F11: 43,
		F12: 44,
		F13: 45,
		F14: 46,
		F15: 1,
	}
	dst := &E3{}

	vals := make(map[string][]string)
	encoder := NewEncoder()
	decoder := NewDecoder()

	encoder.RegisterEncoder(src.F15, func(reflect.Value) string { return "1" })
	decoder.RegisterConverter(src.F15, func(string) reflect.Value { return reflect.ValueOf(1) })

	encoder.Encode(src, vals)
	decoder.Decode(dst, vals)

	if *src != *dst {
		t.Errorf("Decoder-Encoder compatibility: expected %v, got %v\n", src, dst)
	}
}

func TestEmpty(t *testing.T) {
	s := &E1{
		F01: 1,
		F02: 2,
		F03: "three",
	}

	vals := make(map[string][]string)
	_ = NewEncoder().Encode(s, vals)

	valExists(t, "f03", "three", vals)
	valNotExists(t, "f04", vals)
}

func TestStruct(t *testing.T) {
	estr := "schema: interface must be a struct"
	vals := make(map[string][]string)
	err := NewEncoder().Encode("hello world", vals)

	if err.Error() != estr {
		t.Errorf("Expected: %s, got %v", estr, err)
	}
}

func TestRegisterEncoder(t *testing.T) {
	type oneAsWord int
	type twoAsWord int

	s1 := &struct {
		oneAsWord
		twoAsWord
	}{1, 2}
	v1 := make(map[string][]string)

	encoder := NewEncoder()
	encoder.RegisterEncoder(s1.oneAsWord, func(v reflect.Value) string { return "one" })
	encoder.RegisterEncoder(s1.twoAsWord, func(v reflect.Value) string { return "two" })

	encoder.Encode(s1, v1)

	valExists(t, "oneAsWord", "one", v1)
	valExists(t, "twoAsWord", "two", v1)
}

func valExists(t *testing.T, key string, expect string, result map[string][]string) {
	if val, ok := result[key]; !ok {
		t.Error("Key not found. Expected: " + expect)
	} else if val[0] != expect {
		t.Error("Unexpected value. Expected: " + expect + "; got: " + val[0] + ".")
	}
}

func valNotExists(t *testing.T, key string, result map[string][]string) {
	if val, ok := result[key]; ok {
		t.Error("Key not ommited. Expected: empty; got: " + val[0] + ".")
	}
}
