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

	// "server/discover" (SEP-2575) is the stateless replacement for the
	// "initialize" handshake: clients probe capabilities and supported protocol
	// versions with it before they hold a token. It has to bypass authentication
	// for the same reason "initialize" does, otherwise the pre-auth connector
	// setup flow answers 401. The legacy entries above are kept so clients on
	// older spec revisions keep working unchanged.
	//
	// https://modelcontextprotocol.io/seps/2575-stateless-mcp
	"server/discover",
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
