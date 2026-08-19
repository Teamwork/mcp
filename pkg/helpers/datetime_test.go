package helpers_test

import (
	"strings"
	"testing"
	"time"

	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"

	"github.com/teamwork/mcp/pkg/helpers"
)

// TestTimeParamAcceptsLooseLayouts covers the layouts a model emits instead of
// the RFC 3339 the schema advertises. A bare YYYY-MM-DD used to fail outright,
// costing a visible retry on the first call of a time-reporting conversation.
func TestTimeParamAcceptsLooseLayouts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{{
		name:  "RFC 3339 UTC",
		input: "2026-06-24T09:30:15Z",
		want:  time.Date(2026, 6, 24, 9, 30, 15, 0, time.UTC),
	}, {
		name:  "RFC 3339 with offset",
		input: "2026-06-24T09:30:15+01:00",
		want:  time.Date(2026, 6, 24, 9, 30, 15, 0, time.FixedZone("", 3600)),
	}, {
		name:  "RFC 3339 with fractional seconds",
		input: "2026-06-24T09:30:15.500Z",
		want:  time.Date(2026, 6, 24, 9, 30, 15, 500000000, time.UTC),
	}, {
		name:  "offset omitted",
		input: "2026-06-24T09:30:15",
		want:  time.Date(2026, 6, 24, 9, 30, 15, 0, time.UTC),
	}, {
		name:  "seconds and offset omitted",
		input: "2026-06-24T09:30",
		want:  time.Date(2026, 6, 24, 9, 30, 0, 0, time.UTC),
	}, {
		name:  "seconds omitted with offset",
		input: "2026-06-24T09:30Z",
		want:  time.Date(2026, 6, 24, 9, 30, 0, 0, time.UTC),
	}, {
		name:  "space separator",
		input: "2026-06-24 09:30:15",
		want:  time.Date(2026, 6, 24, 9, 30, 15, 0, time.UTC),
	}, {
		name:  "date only",
		input: "2026-06-24",
		want:  time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
	}, {
		name:  "surrounding whitespace",
		input: "  2026-06-24  ",
		want:  time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got time.Time
			err := helpers.ParamGroup(map[string]any{"start_date": tt.input},
				helpers.OptionalTimeParam(&got, "start_date"))
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %s, want %s", got.Format(time.RFC3339Nano), tt.want.Format(time.RFC3339Nano))
			}
		})
	}
}

// TestTimeParamRejectsUnknownLayouts keeps the widening honest: values that are
// not dates at all must still fail, and the error must name the layouts that
// would have worked so the retry is informed.
func TestTimeParamRejectsUnknownLayouts(t *testing.T) {
	for _, input := range []string{"24/06/2026", "June 24 2026", "last tuesday", "20260624"} {
		t.Run(input, func(t *testing.T) {
			var got time.Time
			err := helpers.ParamGroup(map[string]any{"start_date": input},
				helpers.OptionalTimeParam(&got, "start_date"))
			if err == nil {
				t.Fatalf("expected an error for %q, got %s", input, got.Format(time.RFC3339))
			}
			if !strings.Contains(err.Error(), "YYYY-MM-DD") || !strings.Contains(err.Error(), "RFC 3339") {
				t.Errorf("error should name the accepted layouts, got: %v", err)
			}
		})
	}
}

// TestEndOfDayCoversWholeDay verifies the inclusive upper bound. Resolving
// "end_date: 2026-08-03" to that day's first instant would silently drop the
// last day of every range a caller asks for, which is worse than the parse
// error this replaces.
func TestEndOfDayCoversWholeDay(t *testing.T) {
	t.Run("date only expands to the last second", func(t *testing.T) {
		var got *time.Time
		err := helpers.ParamGroup(map[string]any{"end_date": "2026-08-03"},
			helpers.OptionalTimePointerParam(&got, "end_date", helpers.EndOfDay()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected the pointer to be set")
		}
		want := time.Date(2026, 8, 3, 23, 59, 59, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})

	t.Run("explicit time of day is left alone", func(t *testing.T) {
		var got *time.Time
		err := helpers.ParamGroup(map[string]any{"end_date": "2026-08-03T10:00:00Z"},
			helpers.OptionalTimePointerParam(&got, "end_date", helpers.EndOfDay()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected the pointer to be set")
		}
		want := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("got %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
		}
	})

	t.Run("absent parameter leaves the pointer unset", func(t *testing.T) {
		var got *time.Time
		err := helpers.ParamGroup(map[string]any{"end_date": nil},
			helpers.OptionalTimePointerParam(&got, "end_date", helpers.EndOfDay()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected the pointer to remain nil, got %s", got.Format(time.RFC3339))
		}
	})
}

// TestDateParamAcceptsTimestamp covers the mirror-image slip: a timestamp handed
// to a calendar-date parameter. The time of day is unused, so truncating beats
// rejecting.
func TestDateParamAcceptsTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "date only", input: "2026-08-03"},
		{name: "RFC 3339", input: "2026-08-03T14:30:00Z"},
		{name: "offset omitted", input: "2026-08-03T14:30:00"},
	}

	want := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got twapi.Date
			err := helpers.ParamGroup(map[string]any{"date": tt.input},
				helpers.OptionalDateParam(&got, "date"))
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if !time.Time(got).Equal(want) {
				t.Errorf("got %s, want %s", time.Time(got).Format(time.RFC3339), want.Format(time.RFC3339))
			}
		})
	}

	t.Run("rejects a non-date", func(t *testing.T) {
		var got twapi.Date
		err := helpers.ParamGroup(map[string]any{"date": "03/08/2026"},
			helpers.OptionalDateParam(&got, "date"))
		if err == nil {
			t.Fatal("expected an error for a non-date value")
		}
	})
}

// TestLegacyDateParamAcceptsISO covers the compact YYYYMMDD the v1 endpoints
// want, which is not the shape a model volunteers.
func TestLegacyDateParamAcceptsISO(t *testing.T) {
	want := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	for _, input := range []string{"20260803", "2026-08-03", "2026-08-03T14:30:00Z"} {
		t.Run(input, func(t *testing.T) {
			var got projects.LegacyDate
			err := helpers.ParamGroup(map[string]any{"start_at": input},
				helpers.OptionalLegacyDateParam(&got, "start_at"))
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", input, err)
			}
			if !time.Time(got).Equal(want) {
				t.Errorf("got %s, want %s", time.Time(got).Format(time.RFC3339), want.Format(time.RFC3339))
			}
		})
	}

	t.Run("rejects a non-date", func(t *testing.T) {
		var got projects.LegacyDate
		err := helpers.ParamGroup(map[string]any{"start_at": "03-08-2026"},
			helpers.OptionalLegacyDateParam(&got, "start_at"))
		if err == nil {
			t.Fatal("expected an error for a non-date value")
		}
		if !strings.Contains(err.Error(), "YYYYMMDD") {
			t.Errorf("error should name the accepted layout, got: %v", err)
		}
	})
}

// TestNormalizeDateTime covers the handlers that forward the value to the API as
// a query-string parameter instead of binding it to a time.Time.
func TestNormalizeDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		endOfDay bool
		want     string
	}{
		{name: "empty passes through", input: "", want: ""},
		{name: "date only lower bound", input: "2026-08-03", want: "2026-08-03T00:00:00Z"},
		{name: "date only upper bound", input: "2026-08-03", endOfDay: true, want: "2026-08-03T23:59:59Z"},
		{name: "already RFC 3339", input: "2026-08-03T10:00:00Z", want: "2026-08-03T10:00:00Z"},
		{name: "offset omitted", input: "2026-08-03T10:00:00", want: "2026-08-03T10:00:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := helpers.NormalizeDateTime("created_before", tt.input, tt.endOfDay)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("names the parameter in the error", func(t *testing.T) {
		_, err := helpers.NormalizeDateTime("created_before", "not a date", false)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "created_before") {
			t.Errorf("error should name the parameter, got: %v", err)
		}
	})
}
