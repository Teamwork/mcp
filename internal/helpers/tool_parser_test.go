package helpers_test

import (
	"testing"
	"time"

	"github.com/teamwork/mcp/internal/helpers"
)

// TestOptionalParamsAcceptNull verifies that optional parameter parsers treat
// JSON null (nil) as "not provided" rather than returning a type error. This
// matters because the input schemas declare optional params as AnyOf([T, null]),
// so strict-mode clients and some LLMs may explicitly pass null.
func TestOptionalParamsAcceptNull(t *testing.T) {
	t.Run("OptionalParam nil", func(t *testing.T) {
		var s string
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalParam(&s, "k")); err != nil {
			t.Errorf("expected nil error for null optional string, got: %v", err)
		}
	})

	t.Run("OptionalNumericParam nil", func(t *testing.T) {
		var n int64
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalNumericParam(&n, "k")); err != nil {
			t.Errorf("expected nil error for null optional int64, got: %v", err)
		}
	})

	t.Run("OptionalNumericListParam nil", func(t *testing.T) {
		var l []int64
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalNumericListParam(&l, "k")); err != nil {
			t.Errorf("expected nil error for null optional int64 list, got: %v", err)
		}
	})

	t.Run("OptionalListParam nil", func(t *testing.T) {
		var l []string
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalListParam(&l, "k")); err != nil {
			t.Errorf("expected nil error for null optional string list, got: %v", err)
		}
	})

	t.Run("OptionalPointerParam nil leaves pointer unset", func(t *testing.T) {
		var b *bool
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalPointerParam(&b, "k")); err != nil {
			t.Errorf("expected nil error for null optional bool pointer, got: %v", err)
		}
		if b != nil {
			t.Errorf("expected pointer to remain nil for null input, got: %v", *b)
		}
	})

	t.Run("OptionalNumericPointerParam nil leaves pointer unset", func(t *testing.T) {
		var n *int64
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalNumericPointerParam(&n, "k")); err != nil {
			t.Errorf("expected nil error for null optional int64 pointer, got: %v", err)
		}
		if n != nil {
			t.Errorf("expected pointer to remain nil for null input, got: %v", *n)
		}
	})

	t.Run("OptionalTimePointerParam nil leaves pointer unset", func(t *testing.T) {
		var tp *time.Time
		if err := helpers.ParamGroup(map[string]any{"k": nil}, helpers.OptionalTimePointerParam(&tp, "k")); err != nil {
			t.Errorf("expected nil error for null optional time pointer, got: %v", err)
		}
		if tp != nil {
			t.Errorf("expected pointer to remain nil for null input, got: %v", *tp)
		}
	})
}

// TestParamsAcceptDefinedTypes verifies that parameter parsers accept values
// whose target type is a defined type with a matching underlying type (e.g.
// `type Status string`). JSON decoding produces base types (string, float64),
// so a plain type assertion to the defined type would otherwise fail.
func TestParamsAcceptDefinedTypes(t *testing.T) {
	type stringAlias string

	t.Run("RequiredParam with defined string type", func(t *testing.T) {
		var s stringAlias
		err := helpers.ParamGroup(map[string]any{"k": "hello"}, helpers.RequiredParam(&s, "k"))
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if s != "hello" {
			t.Errorf("expected %q, got %q", "hello", s)
		}
	})

	t.Run("OptionalParam with defined string type", func(t *testing.T) {
		var s stringAlias
		err := helpers.ParamGroup(map[string]any{"k": "world"}, helpers.OptionalParam(&s, "k"))
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if s != "world" {
			t.Errorf("expected %q, got %q", "world", s)
		}
	})

	t.Run("OptionalPointerParam with defined string type", func(t *testing.T) {
		var s *stringAlias
		err := helpers.ParamGroup(map[string]any{"k": "ptr"}, helpers.OptionalPointerParam(&s, "k"))
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if s == nil || *s != "ptr" {
			t.Errorf("expected pointer to %q, got %v", "ptr", s)
		}
	})

	t.Run("OptionalListParam with defined string type", func(t *testing.T) {
		var l []stringAlias
		err := helpers.ParamGroup(
			map[string]any{"k": []any{"a", "b", "c"}},
			helpers.OptionalListParam(&l, "k"),
		)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		want := []stringAlias{"a", "b", "c"}
		if len(l) != len(want) {
			t.Fatalf("expected %d items, got %d", len(want), len(l))
		}
		for i := range want {
			if l[i] != want[i] {
				t.Errorf("item %d: expected %q, got %q", i, want[i], l[i])
			}
		}
	})

	t.Run("OptionalListParam with base string type still works", func(t *testing.T) {
		var l []string
		err := helpers.ParamGroup(
			map[string]any{"k": []any{"a", "b"}},
			helpers.OptionalListParam(&l, "k"),
		)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if len(l) != 2 || l[0] != "a" || l[1] != "b" {
			t.Errorf("expected [a b], got %v", l)
		}
	})

	t.Run("OptionalListParam rejects mismatched underlying type", func(t *testing.T) {
		var l []stringAlias
		err := helpers.ParamGroup(
			map[string]any{"k": []any{"a", 42}},
			helpers.OptionalListParam(&l, "k"),
		)
		if err == nil {
			t.Fatalf("expected error for int item in string-aliased list, got nil")
		}
	})

}

// TestParamCoercesStringBool verifies that a boolean parameter supplied as a
// string (as some MCP clients serialize scalars) is parsed rather than causing a
// panic. reflect.Convert(string -> bool) is illegal, so the parser must handle
// bool targets explicitly. Invalid strings must return an error, not panic.
func TestParamCoercesStringBool(t *testing.T) {
	cases := map[string]bool{"true": true, "false": false, "1": true, "0": false}
	for in, want := range cases {
		t.Run("valid "+in, func(t *testing.T) {
			var b bool
			if err := helpers.ParamGroup(map[string]any{"k": in}, helpers.OptionalParam(&b, "k")); err != nil {
				t.Fatalf("expected nil error for %q, got: %v", in, err)
			}
			if b != want {
				t.Errorf("for %q expected %v, got %v", in, want, b)
			}
		})
	}

	t.Run("native bool still works", func(t *testing.T) {
		var b bool
		if err := helpers.ParamGroup(map[string]any{"k": true}, helpers.OptionalParam(&b, "k")); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if !b {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("non-boolean string returns error without panic", func(t *testing.T) {
		var b bool
		err := helpers.ParamGroup(map[string]any{"k": "maybe"}, helpers.OptionalParam(&b, "k"))
		if err == nil {
			t.Fatalf("expected error for non-boolean string, got nil")
		}
	})
}

// TestParamCoercesStringToPointer verifies that a string value is coerced when
// the target is a pointer to a supported type. No current caller uses param with
// a pointer target (pointer optionals go through OptionalPointerParam), but the
// generic parser must handle it safely rather than erroring or panicking.
func TestParamCoercesStringToPointer(t *testing.T) {
	t.Run("string to *string", func(t *testing.T) {
		var sp *string
		if err := helpers.ParamGroup(map[string]any{"k": "hi"}, helpers.OptionalParam(&sp, "k")); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if sp == nil || *sp != "hi" {
			t.Errorf("expected pointer to %q, got %v", "hi", sp)
		}
	})

	t.Run("string to *bool", func(t *testing.T) {
		var bp *bool
		if err := helpers.ParamGroup(map[string]any{"k": "true"}, helpers.OptionalParam(&bp, "k")); err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if bp == nil || !*bp {
			t.Errorf("expected pointer to true, got %v", bp)
		}
	})

	t.Run("invalid bool string to *bool returns error", func(t *testing.T) {
		var bp *bool
		err := helpers.ParamGroup(map[string]any{"k": "maybe"}, helpers.OptionalParam(&bp, "k"))
		if err == nil {
			t.Fatalf("expected error for non-boolean string to *bool, got nil")
		}
	})
}
