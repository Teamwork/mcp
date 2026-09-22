package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// methodsWhitelist are the protocol methods a client may call before it holds a
// token, because they are how it negotiates capabilities in the first place.
//
// The "*/list" methods are deliberately absent. They used to be here so
// registries could index the catalogue anonymously, but a client that completes
// "initialize" and "tools/list" without a token concludes it is connected and
// never sees a 401 — and discovery is 401-gated, so it never authorises and every
// "tools/call" fails. Listing has to answer 401 for the setup flow to start.
//
// https://modelcontextprotocol.io/specification/2026-07-28/basic/lifecycle
// https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization/authorization-server-discovery
var methodsWhitelist = []string{
	"initialize",
	"notifications/initialized",
	"logging/setLevel",

	// "server/discover" (SEP-2575) is the stateless replacement for the
	// "initialize" handshake: clients probe capabilities and supported protocol
	// versions with it before they hold a token. It has to bypass authentication
	// for the same reason "initialize" does, otherwise the pre-auth connector
	// setup flow answers 401.
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
