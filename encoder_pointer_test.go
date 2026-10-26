package schema_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/gorilla/schema"
)

func TestEncoderUnsupportedPointers(t *testing.T) {
	type item struct{ N int }
	type form struct {
		Name   string               `schema:"name"`
		Map    *map[string]string   `schema:"map"`
		Items  *[]item              `schema:"items"`
		Maps   []*map[string]string `schema:"maps"`
		Nested *item                `schema:"nested"`
	}
	m := map[string]string{"key": "value"}
	items := []item{{N: 1}}
	tests := []struct {
		name    string
		input   form
		key     string
		want    []string
		wantErr bool
		custom  bool
	}{
		{"map pointer", form{Map: &m}, "map", []string{"existing"}, true, false},
		{"struct slice pointer", form{Items: &items}, "items", []string{"existing"}, true, false},
		{"slice of map pointers", form{Maps: []*map[string]string{nil, &m}}, "maps", []string{"existing"}, true, false},
		{"nil unsupported pointer", form{}, "map", []string{"null"}, false, false},
		{"slice of nil pointers", form{Maps: []*map[string]string{nil}}, "maps", []string{"null"}, false, false},
		{"registered pointer", form{Map: &m}, "map", []string{"custom"}, false, true},
		{"registered nil pointer", form{}, "map", []string{"custom"}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if caught := recover(); caught != nil {
					t.Errorf("Encode panicked: %v", caught)
				}
			}()
			encoder := schema.NewEncoder()
			if tt.custom {
				encoder.RegisterEncoder((*map[string]string)(nil), func(reflect.Value) string { return "custom" })
			}
			values := url.Values{}
			if tt.wantErr {
				values[tt.key] = []string{"existing"}
			}
			tt.input.Name = "keep"
			err := encoder.Encode(tt.input, values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode error = %v, want error: %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(values[tt.key], tt.want) {
				t.Errorf("values[%q] = %v, want %v", tt.key, values[tt.key], tt.want)
			}
			if !reflect.DeepEqual(values["name"], []string{"keep"}) {
				t.Errorf("supported field was not retained: %v", values)
			}
		})
	}
}
