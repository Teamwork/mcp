package request

import (
	"bytes"
	"net/http"
)

// ResponseWriter is a custom http.ResponseWriter that captures the status code
// and response body.
type ResponseWriter struct {
	http.ResponseWriter

	statusCode int
	body       bytes.Buffer
}

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

// Write captures the response body.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	_, _ = w.body.Write(b) // if error occurs, we ignore it
	return w.ResponseWriter.Write(b)
}

// StatusCode returns the captured status code.
func (w *ResponseWriter) StatusCode() int {
	return w.statusCode
}

// Body returns the captured response body.
func (w *ResponseWriter) Body() []byte {
	return w.body.Bytes()
}
