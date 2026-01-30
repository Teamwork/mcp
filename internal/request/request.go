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
	RemoteIP      string // X-Forwarded-For
	RemoteHost    string // X-Forwarded-Host
	RemoteProto   string // X-Forwarded-Proto
	RemotePort    int64  // X-Forwarded-Port
	RemoteHeaders http.Header
	TraceID       string
}

// NewInfo creates a new Info instance with the provided values.
func NewInfo(r *http.Request) Info {
	var info Info
	if remoteAddr, remotePort, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		info.RemoteIP = remoteAddr
		if port, err := net.LookupPort("tcp", remotePort); err == nil {
			info.RemotePort = int64(port)
		}
	}
	info.RemoteHost = r.Host
	info.RemoteProto = r.Proto
	info.RemoteHeaders = r.Header.Clone()

	traceHTTPHeaders := []string{
		"X-Amzn-Trace-Id",
		"X-Request-Id",
		"X-Correlation-Id",
		"Correlation-Id",
	}
	for _, header := range traceHTTPHeaders {
		if val := r.Header.Get(header); val != "" {
			info.TraceID = val
			break
		}
	}
	if info.TraceID == "" {
		info.TraceID = uuid.NewString()
	}

	return info
}

// WithInfo adds the Info to the context.
func WithInfo(ctx context.Context, info Info) context.Context {
	return context.WithValue(ctx, key{}, info)
}

// InfoFromContext retrieves the Info from the context.
func InfoFromContext(ctx context.Context) (Info, bool) {
	info, ok := ctx.Value(key{}).(Info)
	return info, ok
}

// SetProxyHeaders sets the proxy headers in the request.
func SetProxyHeaders(r *http.Request) {
	info, ok := r.Context().Value(key{}).(Info)
	if !ok {
		return
	}

	// We cannot set "X-Forwarded-Host" because it may replace the Host header
	// when hitting the backend API. For consistency, we will also skip
	// "X-Forwarded-Proto" and "X-Forwarded-Port".

	r.Header.Set("Sent-By", "tw-mcp-server")
	r.Header.Set("X-Forwarded-For", info.RemoteIP)

	if r.Header != nil {
		if headerValue := r.Header.Get("X-Forwarded-For"); headerValue != "" {
			xForwardedForParts := strings.Split(headerValue, ",")
			if xForwardedForParts[len(xForwardedForParts)-1] != info.RemoteIP {
				xForwardedForParts = append(xForwardedForParts, info.RemoteIP)
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
