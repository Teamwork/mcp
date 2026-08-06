package twprojects

import (
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/twapi-go-sdk/projects"
)

func parseUserGroups(
	arguments map[string]any,
	id, label string,
) (*projects.UserGroups, *mcp.CallToolResult) {
	content, ok := arguments[id]
	if !ok {
		return nil, nil
	}
	contentMap, ok := content.(map[string]any)
	if !ok {
		return nil, helpers.NewToolResultTextError("invalid %s: expected an object, got %T", label, content)
	}
	if contentMap == nil {
		return nil, nil
	}

	var userGroups projects.UserGroups
	err := helpers.ParamGroup(contentMap,
		helpers.OptionalNumericListParam(&userGroups.UserIDs, "user_ids"),
		helpers.OptionalNumericListParam(&userGroups.CompanyIDs, "company_ids"),
		helpers.OptionalNumericListParam(&userGroups.TeamIDs, "team_ids"),
		helpers.OptionalNumericListParam(&userGroups.JobRoleIDs, "job_role_ids"),
	)
	if err != nil {
		return nil, helpers.NewToolResultTextError("invalid %s: %s", label, err)
	}
	return &userGroups, nil
}

// notifyChoice is the parsed form of a "notify" argument.
type notifyChoice int

const (
	// notifyChoiceDefault: notify was absent or null; caller applies its default.
	notifyChoiceDefault notifyChoice = iota
	notifyChoiceAll
	notifyChoiceFollowers
	notifyChoiceGroup
	// notifyChoiceNone: caller leaves Notify unset; the API then notifies nobody.
	notifyChoiceNone
)

// parseNotify binds the "notify" argument shared by the message, message
// reply, link and comment tools, coercing near-misses: an array of IDs
// becomes {"user_ids": [...]}, true means followers (withFollowers) or
// "all", false means nobody. The error lists every accepted shape.
func parseNotify(
	arguments map[string]any,
	withFollowers bool,
) (notifyChoice, *projects.LegacyUserGroups, *mcp.CallToolResult) {
	value, ok := arguments["notify"]
	if !ok || value == nil {
		return notifyChoiceDefault, nil, nil
	}
	switch typed := value.(type) {
	case string:
		if strings.EqualFold(typed, "all") {
			return notifyChoiceAll, nil, nil
		}
	case bool:
		if !typed {
			return notifyChoiceNone, nil, nil
		}
		if withFollowers {
			return notifyChoiceFollowers, nil, nil
		}
		return notifyChoiceAll, nil, nil
	case []any:
		groups, toolResult := parseLegacyUserGroups(
			map[string]any{"notify": map[string]any{"user_ids": typed}},
			"notify",
			"notifiers",
		)
		if toolResult == nil && groups != nil && len(groups.UserIDs) > 0 {
			return notifyChoiceGroup, groups, nil
		}
	case map[string]any:
		groups, toolResult := parseLegacyUserGroups(arguments, "notify", "notifiers")
		if toolResult != nil {
			return notifyChoiceDefault, nil, toolResult
		}
		if groups == nil {
			return notifyChoiceDefault, nil, nil
		}
		return notifyChoiceGroup, groups, nil
	}
	return notifyChoiceDefault, nil, helpers.NewToolResultTextError(
		`invalid parameters: notify must be the string "all", a boolean (false notifies nobody), ` +
			"an array of user IDs (e.g. [123, 456]), or an object with user_ids, company_ids, " +
			`team_ids and/or job_role_ids (e.g. {"user_ids": [123, 456]})`)
}

func parseLegacyUserGroups(
	arguments map[string]any,
	id, label string,
) (*projects.LegacyUserGroups, *mcp.CallToolResult) {
	content, ok := arguments[id]
	if !ok {
		return nil, nil
	}
	contentMap, ok := content.(map[string]any)
	if !ok {
		return nil, helpers.NewToolResultTextError("invalid %s: expected an object, got %T", label, content)
	}
	if contentMap == nil {
		return nil, nil
	}

	var userGroups projects.LegacyUserGroups
	err := helpers.ParamGroup(contentMap,
		helpers.OptionalNumericListParam(&userGroups.UserIDs, "user_ids"),
		helpers.OptionalNumericListParam(&userGroups.CompanyIDs, "company_ids"),
		helpers.OptionalNumericListParam(&userGroups.TeamIDs, "team_ids"),
		helpers.OptionalNumericListParam(&userGroups.JobRoleIDs, "job_role_ids"),
	)
	if err != nil {
		return nil, helpers.NewToolResultTextError("invalid %s: %s", label, err)
	}
	return &userGroups, nil
}
