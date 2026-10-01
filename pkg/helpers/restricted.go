package helpers

import (
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RestrictedDataKeys are the attribute names, compared case-insensitively,
// that a tool result for a restricted installation never carries, at any
// depth: contact details, profile pictures, timezones and activity times,
// personal cost rates, message and event recipients, free-text notes about a
// person, and internal identifiers. Names, titles, roles and record IDs stay,
// since no request about people can be answered without them.
var RestrictedDataKeys = NewKeySet(
	// contact details
	"email", "emails", "emailaddress", "emailone", "emailtwo", "emailthree", "verifiedemail",
	"phone", "phonenumber", "mobile", "fax",
	"address", "addressone", "addresstwo", "city", "zip", "postcode",
	"linkedinurl", "facebookurl", "twitterhandle", "contacts",
	// appearance, location in time and activity
	"avatar", "avatarurl", "timezone", "timezoneid", "timezonereferencecode",
	"lastlogin", "lastactive", "lastactivityat", "ipaddress",
	// compensation
	"usercost", "userrate",
	// recipients and participants
	"notifieduserids", "cc", "bcc", "originalrecipient",
	"attendees", "organizer", "eventcreator", "location", "videocalllink",
	// notes and identifiers kept about a person
	"notes", "extradata", "externalid", "ldkey", "apikey", "authkey",
)

// RedactKeys deletes, at any depth of a decoded JSON value, every attribute
// whose lower-cased name is in keys.
func RedactKeys(value any, keys KeySet) {
	switch v := value.(type) {
	case map[string]any:
		for name, child := range v {
			if keys.Has(strings.ToLower(name)) {
				delete(v, name)
				continue
			}
			RedactKeys(child, keys)
		}
	case []any:
		for _, item := range v {
			RedactKeys(item, keys)
		}
	}
}

// RestrictToolResult removes RestrictedDataKeys from a tool result in place:
// from every text block that holds a JSON document, and from the structured
// content. A text block that is not JSON (prose, an error message, a file's
// own content) is left as it is.
func RestrictToolResult(result *mcp.CallToolResult) error {
	if result == nil {
		return nil
	}
	for _, content := range result.Content {
		text, ok := content.(*mcp.TextContent)
		if !ok {
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
			continue
		}
		switch decoded.(type) {
		case map[string]any, []any:
		default:
			continue
		}
		RedactKeys(decoded, RestrictedDataKeys)
		redacted, err := json.Marshal(decoded)
		if err != nil {
			return err
		}
		text.Text = string(redacted)
	}
	if result.StructuredContent != nil {
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return err
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return err
		}
		RedactKeys(decoded, RestrictedDataKeys)
		result.StructuredContent = decoded
	}
	return nil
}
