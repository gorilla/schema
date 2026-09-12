package schema

import (
	"reflect"
	"testing"
)

func TestRequiredIntegerEmptyValues(t *testing.T) {
	for _, sample := range []interface{}{int16(0), uint(0)} {
		typ := reflect.TypeOf(sample)
		t.Run(typ.String(), func(t *testing.T) {
			st := reflect.StructOf([]reflect.StructField{{
				Name: "Value", Type: typ, Tag: `schema:"value,required"`,
			}})
			target := reflect.New(st)
			decoder := NewDecoder()
			err := decoder.Decode(target.Interface(), map[string][]string{"value": {""}})
			errs, ok := err.(MultiError)
			if !ok || errs["value"] != (EmptyFieldError{Key: "value"}) {
				t.Fatalf("empty input error = %v, want EmptyFieldError for value", err)
			}
			if err := decoder.Decode(target.Interface(), map[string][]string{"value": {"0"}}); err != nil {
				t.Errorf("explicit zero should satisfy required: %v", err)
			}
		})
	}
}
