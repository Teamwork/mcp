package twprojects

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/teamwork/mcp/internal/toolsets"
)

// contentTruncationLimit caps content-bearing fields in list-shaped responses.
// Most records fit under it while the long tail (bot reports, agent summaries)
// no longer dominates the payload; full text stays one get call away.
const contentTruncationLimit = 500

// truncateContent shortens a content field, reporting whether it had to. The
// marker is inline so the cut cannot be missed, and names the total size and
// the call that returns the full text.
func truncateContent(content string, method toolsets.Method, id any) (string, bool) {
	// Byte length is never below rune count, so this rejects the common case
	// without allocating.
	if len(content) <= contentTruncationLimit {
		return content, false
	}
	runes := []rune(content)
	if len(runes) <= contentTruncationLimit {
		return content, false
	}

	var marker strings.Builder
	fmt.Fprintf(&marker, "...[truncated — %s chars total", formatThousands(len(runes)))
	if entityID := formatEntityID(id); entityID != "" {
		fmt.Fprintf(&marker, ", %s(id=%s) for full text", method, entityID)
	}
	marker.WriteString("]")

	return string(runes[:contentTruncationLimit]) + marker.String(), true
}

// formatEntityID renders a decoded record's id for the truncation marker, or
// an empty string when there is nothing addressable to point at.
func formatEntityID(id any) string {
	switch value := id.(type) {
	case float64:
		if math.Trunc(value) == value {
			return strconv.FormatInt(int64(value), 10)
		}
	case string:
		return value
	}
	return ""
}

// formatThousands renders a non-negative count with thousands separators.
func formatThousands(n int) string {
	digits := strconv.Itoa(n)
	if len(digits) <= 3 {
		return digits
	}
	var formatted strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			formatted.WriteRune(',')
		}
		formatted.WriteRune(digit)
	}
	return formatted.String()
}
