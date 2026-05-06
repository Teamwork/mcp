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
