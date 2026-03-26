package request

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type key struct{}

// Info stores request information in the context.
type Info struct {
	remoteIP        string // X-Forwarded-For
	remoteHost      string // X-Forwarded-Host
	remoteProto     string // X-Forwarded-Proto
	remotePort      int64  // X-Forwarded-Port
	remoteHeaders   http.Header
	traceID         string
	installationID  int64
	installationURL string
	userID          int64
}

// TraceID returns the request trace ID.
func (i *Info) TraceID() string {
	if i == nil {
		return ""
	}
	return i.traceID
}

// SetAuth sets the authenticated information.
func (i *Info) SetAuth(installationID int64, installationURL string, userID int64) {
	if i == nil {
		return
	}
	i.installationID = installationID
	i.installationURL = installationURL
	i.userID = userID
}

// InstallationID returns the authenticated installation ID.
func (i *Info) InstallationID() int64 {
	if i == nil {
		return 0
	}
	return i.installationID
}

// InstallationURL returns the authenticated installation URL.
func (i *Info) InstallationURL() string {
	if i == nil {
		return ""
	}
	return i.installationURL
}

// UserID returns the authenticated user ID.
func (i *Info) UserID() int64 {
	if i == nil {
		return 0
	}
	return i.userID
}

// NewInfo creates a new Info instance with the provided values.
func NewInfo(r *http.Request) *Info {
	var info Info
	if remoteAddr, remotePort, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		info.remoteIP = remoteAddr
		if port, err := net.LookupPort("tcp", remotePort); err == nil {
			info.remotePort = int64(port)
		}
	}
	info.remoteHost = r.Host
	info.remoteProto = r.Proto
	info.remoteHeaders = r.Header.Clone()

	traceHTTPHeaders := []string{
		"X-Amzn-Trace-Id",
		"X-Request-Id",
		"X-Correlation-Id",
		"Correlation-Id",
	}
	for _, header := range traceHTTPHeaders {
		if val := r.Header.Get(header); val != "" {
			info.traceID = val
			break
		}
	}
	if info.traceID == "" {
		info.traceID = uuid.NewString()
	}

	return &info
}

// WithInfo adds the Info to the context.
func WithInfo(ctx context.Context, info *Info) context.Context {
	if info == nil {
		return ctx
	}
	return context.WithValue(ctx, key{}, info)
}

// InfoFromContext retrieves the Info from the context.
func InfoFromContext(ctx context.Context) (*Info, bool) {
	info, ok := ctx.Value(key{}).(*Info)
	return info, ok
}

// SetProxyHeaders sets the proxy headers in the request.
func SetProxyHeaders(r *http.Request) {
	info, ok := r.Context().Value(key{}).(*Info)
	if !ok {
		return
	}

	// We cannot set "X-Forwarded-Host" because it may replace the Host header
	// when hitting the backend API. For consistency, we will also skip
	// "X-Forwarded-Proto" and "X-Forwarded-Port".

	r.Header.Set("Sent-By", "tw-mcp-server")
	r.Header.Set("X-Forwarded-For", info.remoteIP)

	if r.Header != nil {
		if headerValue := r.Header.Get("X-Forwarded-For"); headerValue != "" {
			xForwardedForParts := strings.Split(headerValue, ",")
			if xForwardedForParts[len(xForwardedForParts)-1] != info.remoteIP {
				xForwardedForParts = append(xForwardedForParts, info.remoteIP)
			}
			r.Header.Set("X-Forwarded-For", strings.Join(xForwardedForParts, ","))
		}
		if headerValue := r.Header.Get("X-Real-IP"); headerValue != "" {
			r.Header.Set("X-Real-IP", headerValue)
		}
		if headerValue := r.Header.Get("X-Request-ID"); headerValue != "" {
			r.Header.Set("X-Request-ID", headerValue)
		}
		if headerValue := r.Header.Get("X-Amzn-Trace-ID"); headerValue != "" {
			r.Header.Set("X-Amzn-Trace-ID", headerValue)
		}

		// https://www.w3.org/TR/trace-context/
		if headerValue := r.Header.Get("Traceparent"); headerValue != "" {
			r.Header.Set("Traceparent", headerValue)
		}
		if headerValue := r.Header.Get("Tracestate"); headerValue != "" {
			r.Header.Set("Tracestate", headerValue)
		}
	}

	// RFC 7239
	//
	// Similar from the "X-Forwarded-For" header, we will not set the "host" and
	// "proto" parameters.
	r.Header.Set("Forwarded", fmt.Sprintf("for=%s", r.Header.Get("X-Forwarded-For")))
}
