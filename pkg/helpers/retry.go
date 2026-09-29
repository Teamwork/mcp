package helpers

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	twapi "github.com/teamwork/twapi-go-sdk"
)

// RetryPolicy bounds how Retry repeats a call that failed with a transient
// status.
type RetryPolicy struct {
	// MaxAttempts counts the first call, so 4 means up to 3 retries.
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	// MaxRetryAfter is the longest Retry-After honoured; a longer one ends the retries.
	MaxRetryAfter time.Duration
}

// TransientRetry is the policy for writes that are safe to repeat.
var TransientRetry = RetryPolicy{
	MaxAttempts:   4,
	BaseDelay:     200 * time.Millisecond,
	MaxDelay:      2 * time.Second,
	MaxRetryAfter: 5 * time.Second,
}

// Retry repeats fn on 429, 500 or 503; only use it for calls that are safe to repeat.
func Retry[T any](ctx context.Context, policy RetryPolicy, fn func() (T, error)) (T, error) {
	var result T
	var err error
	for attempt := range max(policy.MaxAttempts, 1) {
		if attempt > 0 {
			wait, ok := retryWait(policy, attempt, err)
			if !ok {
				return result, err
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return result, errors.Join(err, ctx.Err())
			case <-timer.C:
			}
		}
		if result, err = fn(); err == nil || !isTransient(err) {
			return result, err
		}
	}
	return result, err
}

func isTransient(err error) bool {
	httpErr, ok := errors.AsType[*twapi.HTTPError](err)
	if !ok {
		return false
	}
	switch httpErr.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable:
		return true
	}
	return false
}

// retryWait is exponential backoff with jitter over its upper half, raised to
// the server's Retry-After when it sends one.
func retryWait(policy RetryPolicy, attempt int, err error) (time.Duration, bool) {
	backoff := max(policy.BaseDelay, 0)
	for i := 1; i < attempt && backoff <= policy.MaxDelay/2; i++ {
		backoff *= 2
	}
	backoff = min(backoff, policy.MaxDelay)
	wait := backoff/2 + rand.N(backoff/2+1)

	if httpErr, ok := errors.AsType[*twapi.HTTPError](err); ok {
		if after, ok := parseRetryAfter(httpErr.Headers.Get("Retry-After")); ok {
			if after > policy.MaxRetryAfter {
				return 0, false
			}
			wait = max(wait, after)
		}
	}
	return wait, true
}

func parseRetryAfter(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(time.Until(at), 0), true
	}
	return 0, false
}
