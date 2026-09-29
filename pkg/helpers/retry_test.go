package helpers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/teamwork/mcp/pkg/helpers"
	twapi "github.com/teamwork/twapi-go-sdk"
)

var fastRetry = helpers.RetryPolicy{
	MaxAttempts:   4,
	BaseDelay:     time.Millisecond,
	MaxDelay:      2 * time.Millisecond,
	MaxRetryAfter: time.Second,
}

func statusErr(status int, headers http.Header) error {
	return &twapi.HTTPError{StatusCode: status, Headers: headers, Message: "request failed"}
}

// sequence returns fn that fails with each error in turn, then succeeds.
func sequence(errs ...error) (func() (string, error), *int) {
	calls := new(int)
	return func() (string, error) {
		*calls++
		if *calls <= len(errs) {
			return "", errs[*calls-1]
		}
		return "ok", nil
	}, calls
}

func TestRetryRecoversFromTransientStatus(t *testing.T) {
	for _, status := range []int{
		http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable,
	} {
		fn, calls := sequence(statusErr(status, nil))
		got, err := helpers.Retry(t.Context(), fastRetry, fn)
		if err != nil || got != "ok" {
			t.Errorf("status %d: expected success, got %q, %v", status, got, err)
		}
		if *calls != 2 {
			t.Errorf("status %d: expected 2 attempts, got %d", status, *calls)
		}
	}
}

func TestRetryGivesUpAfterMaxAttempts(t *testing.T) {
	last := statusErr(http.StatusServiceUnavailable, nil)
	fn, calls := sequence(statusErr(500, nil), statusErr(500, nil), statusErr(500, nil), last)
	_, err := helpers.Retry(t.Context(), fastRetry, fn)
	if !errors.Is(err, last) {
		t.Errorf("expected the last error, got %v", err)
	}
	if *calls != fastRetry.MaxAttempts {
		t.Errorf("expected %d attempts, got %d", fastRetry.MaxAttempts, *calls)
	}
}

func TestRetrySkipsPermanentErrors(t *testing.T) {
	for _, err := range []error{
		statusErr(http.StatusBadRequest, nil),
		statusErr(http.StatusNotFound, nil),
		statusErr(http.StatusUnprocessableEntity, nil),
		statusErr(http.StatusBadGateway, nil),
		statusErr(http.StatusGatewayTimeout, nil),
		errors.New("connection reset"),
	} {
		fn, calls := sequence(err)
		if _, got := helpers.Retry(t.Context(), fastRetry, fn); !errors.Is(got, err) {
			t.Errorf("%v: expected it returned as is, got %v", err, got)
		}
		if *calls != 1 {
			t.Errorf("%v: expected 1 attempt, got %d", err, *calls)
		}
	}
}

func TestRetryStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	fn, calls := sequence(statusErr(500, nil), statusErr(500, nil))
	_, err := helpers.Retry(ctx, helpers.TransientRetry, fn)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if _, ok := errors.AsType[*twapi.HTTPError](err); !ok {
		t.Errorf("expected the error of the first attempt, got %v", err)
	}
	if *calls != 1 {
		t.Errorf("expected 1 attempt, got %d", *calls)
	}
}

func TestRetryHonoursRetryAfter(t *testing.T) {
	fn, calls := sequence(statusErr(http.StatusTooManyRequests, http.Header{"Retry-After": {"1"}}))
	start := time.Now()
	if _, err := helpers.Retry(t.Context(), fastRetry, fn); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("expected to wait for Retry-After, waited %s", elapsed)
	}
	if *calls != 2 {
		t.Errorf("expected 2 attempts, got %d", *calls)
	}
}

func TestRetryGivesUpOnLongRetryAfter(t *testing.T) {
	fn, calls := sequence(statusErr(http.StatusServiceUnavailable, http.Header{"Retry-After": {"60"}}))
	if _, err := helpers.Retry(t.Context(), fastRetry, fn); err == nil {
		t.Error("expected the error to be returned")
	}
	if *calls != 1 {
		t.Errorf("expected 1 attempt, got %d", *calls)
	}
}

func TestRetryLargeMaxAttemptsDoesNotOverflow(t *testing.T) {
	policy := helpers.RetryPolicy{MaxAttempts: 80, BaseDelay: time.Nanosecond, MaxDelay: 64 * time.Nanosecond}
	errs := make([]error, 79)
	for i := range errs {
		errs[i] = statusErr(http.StatusInternalServerError, nil)
	}
	fn, calls := sequence(errs...)
	if _, err := helpers.Retry(t.Context(), policy, fn); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if *calls != 80 {
		t.Errorf("expected 80 attempts, got %d", *calls)
	}
}
