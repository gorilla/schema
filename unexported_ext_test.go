package schema_test

import (
	"testing"

	"github.com/gorilla/schema"
)

type srcWithPrivate struct {
	Public  string `schema:"public"`
	private string
	hidden  *int
}

func TestEncoderSkipsUnexportedFields(t *testing.T) {
	n := 3
	src := srcWithPrivate{Public: "ok", private: "secret", hidden: &n}
	dst := map[string][]string{}
	if err := schema.NewEncoder().Encode(src, dst); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if got := dst["public"]; len(got) != 1 || got[0] != "ok" {
		t.Fatalf("public: %v", dst)
	}
	if _, ok := dst["private"]; ok {
		t.Fatalf("encoded unexported field: %v", dst)
	}
	if _, ok := dst["hidden"]; ok {
		t.Fatalf("encoded unexported pointer: %v", dst)
	}
}

type innerWithPrivate struct {
	A string `schema:"a,omitempty"`
	b int
}

type outerOmitPrivate struct {
	Inner innerWithPrivate `schema:"inner,omitempty"`
}

func TestEncoderOmitemptyWithUnexportedFields(t *testing.T) {
	dst := map[string][]string{}
	if err := schema.NewEncoder().Encode(outerOmitPrivate{}, dst); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(dst) != 0 {
		t.Fatalf("expected empty encoding, got %v", dst)
	}
}

type destWithPrivate struct {
	Public  string `schema:"public"`
	private string `schema:"private"`
}

func TestDecoderSkipsUnexportedFields(t *testing.T) {
	var dst destWithPrivate
	src := map[string][]string{
		"public":  {"ok"},
		"private": {"secret"},
	}
	if err := schema.NewDecoder().Decode(&dst, src); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dst.Public != "ok" {
		t.Fatalf("public=%q", dst.Public)
	}
	if dst.private != "" {
		t.Fatalf("decoded unexported field: %q", dst.private)
	}
}

type privEmbed struct {
	X string `schema:"x"`
}

type destEmbedPrivate struct {
	*privEmbed
	Public string `schema:"public"`
}

func TestDecoderUnexportedEmbeddedPointer(t *testing.T) {
	var dst destEmbedPrivate
	src := map[string][]string{"public": {"ok"}}
	if err := schema.NewDecoder().Decode(&dst, src); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dst.Public != "ok" {
		t.Fatalf("public=%q", dst.Public)
	}
	if dst.privEmbed != nil {
		t.Fatalf("set unexported embedded pointer: %+v", dst.privEmbed)
	}
}
