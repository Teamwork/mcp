// Package logsafe scrubs request payloads before they reach a log sink or a
// trace tag.
//
// Two things have to happen and the order matters. File content is replaced
// first, so an attachment does not leave part of a customer's document in the
// log; whatever survives is then capped, so an oversized payload of any shape
// cannot fill it. Truncation on its own is not enough, because base64 usually
// leads the arguments object: the retained head would be the readable start of
// the file, and the useful part of the record, the tool name and the other
// parameters, would be what got cut.
package logsafe

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// MaxLoggedBytes caps a scrubbed payload. It matches the cap the response
// writer already applies, so inbound and outbound bodies are treated alike.
const MaxLoggedBytes = 64 << 10

const truncatedSuffix = "...[truncated]"

// contentKeys are the JSON keys whose value is uploaded file content. They are
// upload-specific: "content" is deliberately excluded, since page and comment
// bodies use it for text worth keeping in the log.
var contentKeys = [][]byte{[]byte(`"data"`), []byte(`"fileData"`), []byte(`"file_data"`)}

// contentValue matches the JSON string value under one of contentKeys. The
// value class is "anything but a quote", so it catches a file of any size,
// including a small one, and is not broken by JSON escaping such as "\/". The
// key anchor means an unrelated field whose value happens to be one of these
// words is untouched. There is no nested quantifier, so the match stays linear
// over a multi-megabyte body.
var contentValue = regexp.MustCompile(`"(data|fileData|file_data)"\s*:\s*"[^"]*"`)

// Bytes returns b with file content replaced and the result capped.
func Bytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	scrubbed := b
	if containsContentKey(b) {
		scrubbed = contentValue.ReplaceAllFunc(b, func(match []byte) []byte {
			key := match[:bytes.IndexByte(match, ':')]
			return fmt.Appendf(nil, `%s:"<redacted %d bytes>"`, key, len(match))
		})
	}
	if len(scrubbed) > MaxLoggedBytes {
		return append(bytes.Clone(scrubbed[:MaxLoggedBytes]), truncatedSuffix...)
	}
	return scrubbed
}

// containsContentKey reports whether b mentions any upload key, so the regex
// scan and its allocation are skipped for the payloads that carry no file,
// which is almost all of them.
func containsContentKey(b []byte) bool {
	for _, key := range contentKeys {
		if bytes.Contains(b, key) {
			return true
		}
	}
	return false
}

// String is Bytes over a string.
func String(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(Bytes([]byte(s)))
}

// IsTextualContentType reports whether a body of this content type is worth
// capturing in a log. A pending file upload sends raw bytes, and capturing them
// would copy the customer's file into the log and hold a second copy in memory
// for the life of the request.
func IsTextualContentType(contentType string) bool {
	if contentType == "" {
		// No declared type: the bodies the server sends without one are JSON, and
		// anything carrying a file always declares its type.
		return true
	}
	mediaType := contentType
	if index := strings.IndexByte(mediaType, ';'); index >= 0 {
		mediaType = mediaType[:index]
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json",
		"application/x-www-form-urlencoded",
		"application/xml",
		"application/javascript":
		return true
	}
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml")
}

// ElidedBody is the placeholder logged in place of a body that is not worth
// capturing.
func ElidedBody(length int64, contentType string) string {
	if length < 0 {
		return fmt.Sprintf("<body of %s elided>", contentType)
	}
	return fmt.Sprintf("<%d bytes of %s elided>", length, contentType)
}
