package twprojects

import (
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Projects.
func DefaultToolsetGroup(readOnly bool, engine *twapi.Engine) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly)
	group.AddToolset(toolsets.NewToolset("projects", projectDescription).
		AddWriteTools(
			ProjectCreate(engine),
			ProjectUpdate(engine),
			ProjectDelete(engine),
		).
		AddReadTools(
			ProjectGet(engine),
			ProjectList(engine),
		))
	return group
}
