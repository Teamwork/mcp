package request

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// flushRecorder is an http.ResponseWriter that records how often it was
// flushed, standing in for the real connection.
type flushRecorder struct {
	http.ResponseWriter
	flushes int
}

func (f *flushRecorder) Flush() { f.flushes++ }

// TestResponseWriterFlushPropagates guards the reason Flush and Unwrap exist:
// the MCP SDK writes Server-Sent Events through http.ResponseController, which
// resolves the flusher by unwrapping. Without it, flushing silently degrades to
// a no-op and streaming clients hang until the handler returns.
func TestResponseWriterFlushPropagates(t *testing.T) {
	inner := &flushRecorder{ResponseWriter: httptest.NewRecorder()}
	rw := NewResponseWriter(inner)

	// Direct http.Flusher assertion, as used by older middleware.
	flusher, ok := any(rw).(http.Flusher)
	if !ok {
		t.Fatal("ResponseWriter does not implement http.Flusher")
	}
	flusher.Flush()
	if inner.flushes != 1 {
		t.Errorf("after Flush(): inner flushes = %d, want 1", inner.flushes)
	}

	// http.ResponseController resolution, as used by the MCP SDK.
	if err := http.NewResponseController(rw).Flush(); err != nil {
		t.Errorf("http.NewResponseController(rw).Flush() = %v, want nil", err)
	}
	if inner.flushes != 2 {
		t.Errorf("after ResponseController flush: inner flushes = %d, want 2", inner.flushes)
	}
}

func TestResponseWriterUnwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := NewResponseWriter(inner)

	if got := rw.Unwrap(); got != http.ResponseWriter(inner) {
		t.Errorf("Unwrap() = %v, want the wrapped writer", got)
	}
}

func TestResponseWriterCapturesStatusAndBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := NewResponseWriter(recorder)

	rw.WriteHeader(http.StatusAccepted)
	if _, err := rw.Write([]byte("hello ")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := rw.Write([]byte("world")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if got := rw.StatusCode(); got != http.StatusAccepted {
		t.Errorf("StatusCode() = %d, want %d", got, http.StatusAccepted)
	}
	if got := string(rw.Body()); got != "hello world" {
		t.Errorf("Body() = %q, want %q", got, "hello world")
	}
	// The body must still reach the client untouched.
	if got := recorder.Body.String(); got != "hello world" {
		t.Errorf("forwarded body = %q, want %q", got, "hello world")
	}
}

// TestResponseWriterTruncatesLargeBody guards against unbounded memory growth
// on long-lived Server-Sent Events streams, which stay open for the lifetime of
// the request.
func TestResponseWriterTruncatesLargeBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := NewResponseWriter(recorder)

	// Write in chunks, crossing the cap mid-chunk and then well past it.
	chunk := bytes.Repeat([]byte("a"), maxCapturedBodySize/2)
	for range 5 {
		if _, err := rw.Write(chunk); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	body := rw.Body()
	if !strings.HasSuffix(string(body), truncatedBodySuffix) {
		t.Errorf("Body() does not end with %q", truncatedBodySuffix)
	}
	if want := maxCapturedBodySize + len(truncatedBodySuffix); len(body) != want {
		t.Errorf("len(Body()) = %d, want %d", len(body), want)
	}

	// Truncation must only affect what is logged, never what is sent.
	if got, want := recorder.Body.Len(), len(chunk)*5; got != want {
		t.Errorf("forwarded body length = %d, want %d", got, want)
	}
}
