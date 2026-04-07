package twprojects

import (
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
	)
	if err != nil {
		return nil, helpers.NewToolResultTextError("invalid %s: %s", label, err)
	}
	return &userGroups, nil
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
	)
	if err != nil {
		return nil, helpers.NewToolResultTextError("invalid %s: %s", label, err)
	}
	return &userGroups, nil
}
