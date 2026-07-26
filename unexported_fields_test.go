package schema_test

import (
	"reflect"
	"testing"

	"github.com/gorilla/schema"
)

func TestDecoderSkipsUnexportedFields(t *testing.T) {
	dst := struct {
		Exported   string `schema:"exported"`
		unexported string `schema:"unexported,required"`
	}{}

	err := schema.NewDecoder().Decode(&dst, map[string][]string{
		"exported": {"value"},
	})
	if err != nil {
		t.Fatalf("Decode returned an error for an unexported required field: %v", err)
	}
	if dst.Exported != "value" {
		t.Fatalf("Exported = %q, want %q", dst.Exported, "value")
	}
}

func TestEncoderSkipsUnexportedFields(t *testing.T) {
	src := struct {
		Exported   string `schema:"exported"`
		unexported string `schema:"unexported,omitempty"`
	}{
		Exported:   "value",
		unexported: "private",
	}
	dst := map[string][]string{}

	if err := schema.NewEncoder().Encode(src, dst); err != nil {
		t.Fatalf("Encode returned an error: %v", err)
	}
	want := map[string][]string{"exported": {"value"}}
	if !reflect.DeepEqual(dst, want) {
		t.Fatalf("Encode = %#v, want %#v", dst, want)
	}
}
