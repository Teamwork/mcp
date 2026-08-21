package helpers

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// dateOnlyLayout is the ISO 8601 calendar-date layout. A parameter value in this
// shape names a whole day rather than an instant, so it is widened to cover the
// day (see parseDateTime).
const dateOnlyLayout = "2006-01-02"

// legacyDateLayout is the compact date layout the legacy v1 endpoints expect.
const legacyDateLayout = "20060102"

// dateTimeLayouts are the layouts accepted for a date-time parameter, tried in
// order. RFC 3339 is what the schema advertises and what the API is given, but a
// model asked for "timelogs from June 24 to August 3" reliably emits a bare
// calendar date, or a local date-time with no offset — rejecting those turns the
// first tool call of a reporting conversation into a wasted round trip. A value
// with no offset is read as UTC.
var dateTimeLayouts = []string{
	time.RFC3339,             // 2026-08-03T14:30:00Z, 2026-08-03T14:30:00+01:00
	"2006-01-02T15:04Z07:00", // seconds omitted
	"2006-01-02T15:04:05",    // offset omitted
	"2006-01-02T15:04",       // seconds and offset omitted
	"2006-01-02 15:04:05",    // space instead of the T separator
	"2006-01-02 15:04",
	dateOnlyLayout, // 2026-08-03
}

// parseDateTime parses a date-time parameter value, accepting every layout in
// dateTimeLayouts.
//
// A date-only value denotes the whole day, so it resolves to the day's first
// instant in UTC, or to its last second when endOfDay is true. Upper-bound
// filters pass endOfDay so that "end_date: 2026-08-03" includes the 3rd instead
// of silently truncating the range at its first instant — a wrong answer is
// worse here than the error this replaces.
func parseDateTime(value string, endOfDay bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range dateTimeLayouts {
		t, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		if layout == dateOnlyLayout && endOfDay {
			t = t.Add(24*time.Hour - time.Second)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%q is not a valid date or date-time, expected YYYY-MM-DD or RFC 3339 "+
		"(e.g. 2026-08-03 or 2026-08-03T14:30:00Z)", value)
}

// parseDate parses a calendar-date parameter value. It prefers the date-only
// layout and falls back to the date-time layouts, truncating to the date, so a
// model that answers a date question with a timestamp is not rejected over the
// unused time of day.
func parseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(dateOnlyLayout, value); err == nil {
		return t, nil
	}
	t, err := parseDateTime(value, false)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a valid date, expected YYYY-MM-DD (e.g. 2026-08-03)", value)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
}

// parseLegacyDate parses a compact YYYYMMDD parameter value, falling back to the
// ISO layouts. The compact form is what the legacy endpoints want but not what a
// model volunteers, so a hyphenated date is accepted and re-encoded rather than
// bounced back to the caller.
func parseLegacyDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(legacyDateLayout, value); err == nil {
		return t, nil
	}
	t, err := parseDate(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a valid date, expected YYYYMMDD (e.g. 20260803)", value)
	}
	return t, nil
}

// EndOfDay marks a date-time parameter as an upper bound, so a date-only value
// resolves to the last second of the day the caller named rather than its first
// instant. Pass it to the time parameter binders for any filter that means "up
// to and including" — end_date, and the *_before window ends a model fills from
// a date range:
//
//	helpers.OptionalTimePointerParam(&req.Filters.EndDate, "end_date", helpers.EndOfDay())
//
// Values that already carry a time of day are left untouched.
func EndOfDay() ParamMiddleware[string] {
	return func(value *string) (bool, error) {
		if value == nil {
			return true, nil
		}
		if _, err := time.Parse(dateOnlyLayout, strings.TrimSpace(*value)); err != nil {
			return true, nil
		}
		*value = strings.TrimSpace(*value) + "T23:59:59Z"
		return true, nil
	}
}

// NormalizeDateTime parses value with the same tolerance as the date-time
// parameter binders and re-renders it as RFC 3339, for handlers that forward the
// value to the API as a string instead of binding it to a time.Time. An empty
// value is returned unchanged. Set endOfDay for upper-bound filters, as
// described on EndOfDay.
func NormalizeDateTime(key, value string, endOfDay bool) (string, error) {
	if strings.TrimSpace(value) == "" {
		return value, nil
	}
	t, err := parseDateTime(value, endOfDay)
	if err != nil {
		return "", fmt.Errorf("invalid time format for %s: %w", key, err)
	}
	return t.Format(time.RFC3339), nil
}

// NullifyEmptyDates rewrites an empty-string value held at any of the named JSON
// keys to null, wherever it appears in a raw API body.
//
// The v1 routes spell "unset" as an empty string on a date field, and the list_*
// contract streams those bodies straight to the caller, so the SDK's
// MarshalJSON — which encodes an unset OptionalDateTime as null — never runs.
// WithDateTypeSchema declares such a field with a "date" or "date-time" format,
// and an empty string satisfies neither, so a client that asserts formats
// discards the whole response. Call this on any raw body carrying a field the
// published schema gives a date format.
//
// Anything unexpected leaves the payload untouched: returning the response whole
// beats failing a read the API already answered.
func NullifyEmptyDates(body []byte, fields ...string) []byte {
	if len(fields) == 0 {
		return body
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return body
	}
	if !nullifyEmptyDates(decoded, fields) {
		return body
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return body
	}
	return encoded
}

// nullifyEmptyDates walks value, replacing an empty string held at one of the
// named keys with nil. It reports whether it changed anything, so the caller can
// return the original bytes untouched when there was nothing to do.
func nullifyEmptyDates(value any, fields []string) bool {
	var changed bool
	switch typed := value.(type) {
	case map[string]any:
		for key, held := range typed {
			if held == "" && slices.Contains(fields, key) {
				typed[key] = nil
				changed = true
				continue
			}
			if nullifyEmptyDates(held, fields) {
				changed = true
			}
		}
	case []any:
		for _, held := range typed {
			if nullifyEmptyDates(held, fields) {
				changed = true
			}
		}
	}
	return changed
}
