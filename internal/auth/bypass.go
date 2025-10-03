package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

var methodsWhitelist = []string{
	// allow some protocol methods to bypass authentication
	//
	// https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle
	// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#listing-tools
	// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#listing-resources
	// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#resource-templates
	// https://modelcontextprotocol.io/specification/2025-06-18/server/prompts#listing-prompts
	"initialize",
	"notifications/initialized",
	"logging/setLevel",
	"tools/list",
	"resources/list",
	"resources/templates/list",
	"prompts/list",
}

// Bypass checks if the protocol method can bypass authentication.
func Bypass(data []byte) (bool, error) {
	var baseMessage struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(data, &baseMessage); err != nil {
		return false, fmt.Errorf("parse error: %w", err)
	}
	if !BypassMethod(baseMessage.Method) {
		return false, errors.New("not authenticated")
	}
	return true, nil
}

// BypassMethod checks if the protocol method can bypass authentication.
func BypassMethod(method string) bool {
	return slices.Contains(methodsWhitelist, method)
}
