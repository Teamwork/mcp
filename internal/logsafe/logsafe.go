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

// contentValue matches a long JSON string value under one of the keys that
// carry file content.
//
// The alternation is anchored on the key and the value character class is the
// base64 alphabet plus whitespace, so there is no nested quantifier and the
// expression stays linear over a multi-megabyte body. The 256 character floor
// keeps genuinely short values readable, and matching on the key means a search
// term that happens to be the word "data" is left alone.
var contentValue = regexp.MustCompile(`"(data|content|fileData|file_data)"\s*:\s*"[A-Za-z0-9+/=_\-\s]{256,}"`)

// Bytes returns b with file content replaced and the result capped.
func Bytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	scrubbed := contentValue.ReplaceAllFunc(b, func(match []byte) []byte {
		key := match[:bytes.IndexByte(match, ':')]
		return fmt.Appendf(nil, `%s:"<redacted %d bytes>"`, key, len(match))
	})
	if len(scrubbed) > MaxLoggedBytes {
		return append(bytes.Clone(scrubbed[:MaxLoggedBytes]), truncatedSuffix...)
	}
	return scrubbed
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
