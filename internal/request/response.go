package request

import (
	"bytes"
	"net/http"
)

// maxCapturedBodySize caps how much of the response body is retained for
// logging. Streamable HTTP responses are Server-Sent Events streams that can
// stay open for the whole lifetime of a request, so capturing them in full
// would grow without bound.
const maxCapturedBodySize = 64 << 10 // 64 KiB

// truncatedBodySuffix is appended to the captured body when it was truncated.
const truncatedBodySuffix = "...[truncated]"

// ResponseWriter is a custom http.ResponseWriter that captures the status code
// and response body.
//
// It deliberately implements http.Flusher and Unwrap so that
// http.ResponseController can reach the flusher of the wrapped writer. The MCP
// SDK writes Server-Sent Events through http.ResponseController: without these
// methods flushing silently becomes a no-op and clients waiting on the stream
// receive nothing until the handler returns.
type ResponseWriter struct {
	http.ResponseWriter

	statusCode int
	body       bytes.Buffer
	truncated  bool
}

// Ensure the wrapper keeps satisfying the interfaces the MCP SDK relies on.
var (
	_ http.ResponseWriter = (*ResponseWriter)(nil)
	_ http.Flusher        = (*ResponseWriter)(nil)
)

// NewResponseWriter creates a new ResponseWriter.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status code
	}
}

// WriteHeader captures the status code.
func (w *ResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures the response body, up to maxCapturedBodySize, and forwards it
// to the wrapped writer.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if remaining := maxCapturedBodySize - w.body.Len(); remaining > 0 {
		if len(b) <= remaining {
			_, _ = w.body.Write(b) // if error occurs, we ignore it
		} else {
			_, _ = w.body.Write(b[:remaining]) // if error occurs, we ignore it
			w.truncated = true
		}
	} else if len(b) > 0 {
		w.truncated = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush forwards the flush to the wrapped writer, when it supports it. This
// keeps Server-Sent Events streams responsive while still being logged.
func (w *ResponseWriter) Flush() {
	// http.NewResponseController resolves the flusher through Unwrap, so it
	// handles arbitrarily deep wrapper chains.
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap returns the wrapped http.ResponseWriter, allowing
// http.ResponseController to reach optional interfaces (flushing, hijacking,
// deadlines) implemented further down the chain.
func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// StatusCode returns the captured status code.
func (w *ResponseWriter) StatusCode() int {
	return w.statusCode
}

// Body returns the captured response body. Bodies larger than
// maxCapturedBodySize are truncated and marked as such.
func (w *ResponseWriter) Body() []byte {
	if !w.truncated {
		return w.body.Bytes()
	}
	return append(bytes.Clone(w.body.Bytes()), truncatedBodySuffix...)
}
