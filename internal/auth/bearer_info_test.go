package auth

import (
	"errors"
	"net/http"
	"testing"
)

func TestClassifyAuthStatus(t *testing.T) {
	const url = "https://www.teamwork.com/launchpad/v1/userinfo.json"

	tests := []struct {
		name       string
		statusCode int
		want       error
	}{
		{name: "ok validates the token", statusCode: http.StatusOK, want: nil},

		// Teamwork API looked at the token and refused it. Only these may lead
		// to a re-authorisation challenge.
		{name: "unauthorized is a rejection", statusCode: http.StatusUnauthorized, want: ErrBearerInfoUnauthorized},
		{name: "forbidden is a rejection", statusCode: http.StatusForbidden, want: ErrBearerInfoUnauthorized},

		// Validation never happened. These used to be reported as rejections,
		// which made the client discard a valid token.
		{name: "internal server error is inconclusive", statusCode: http.StatusInternalServerError,
			want: ErrBearerInfoUnavailable},
		{name: "bad gateway is inconclusive", statusCode: http.StatusBadGateway,
			want: ErrBearerInfoUnavailable},
		{name: "service unavailable is inconclusive", statusCode: http.StatusServiceUnavailable,
			want: ErrBearerInfoUnavailable},
		{name: "gateway timeout is inconclusive", statusCode: http.StatusGatewayTimeout,
			want: ErrBearerInfoUnavailable},
		{name: "too many requests is inconclusive", statusCode: http.StatusTooManyRequests,
			want: ErrBearerInfoUnavailable},
		{name: "bad request is inconclusive", statusCode: http.StatusBadRequest,
			want: ErrBearerInfoUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyAuthStatus(tt.statusCode, url)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("classifyAuthStatus(%d) = %v, want nil", tt.statusCode, err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("classifyAuthStatus(%d) = %v, want %v", tt.statusCode, err, tt.want)
			}
		})
	}
}

// TestClassifyAuthStatusNeverConfusesUnavailableWithRejection guards the
// invariant the middleware depends on: an inconclusive result must never be
// mistaken for a rejection, because only a rejection is allowed to tell the
// OAuth client to throw its token away.
func TestClassifyAuthStatusNeverConfusesUnavailableWithRejection(t *testing.T) {
	for statusCode := 100; statusCode < 600; statusCode++ {
		err := classifyAuthStatus(statusCode, "https://example.com")
		if err == nil {
			continue
		}
		rejected := errors.Is(err, ErrBearerInfoUnauthorized)
		if rejected != (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden) {
			t.Errorf("status %d: rejection = %v, want %v", statusCode, rejected,
				statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden)
		}
		if rejected && errors.Is(err, ErrBearerInfoUnavailable) {
			t.Errorf("status %d: classified as both a rejection and inconclusive", statusCode)
		}
	}
}
