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
			TasklistCreate(engine),
			TasklistUpdate(engine),
			TasklistDelete(engine),
			TaskCreate(engine),
			TaskUpdate(engine),
			TaskDelete(engine),
			UserCreate(engine),
			UserUpdate(engine),
			UserDelete(engine),
		).
		AddReadTools(
			ProjectGet(engine),
			ProjectList(engine),
			TasklistGet(engine),
			TasklistList(engine),
			TasklistListByProject(engine),
			TaskGet(engine),
			TaskList(engine),
			TaskListByTasklist(engine),
			TaskListByProject(engine),
			UserGet(engine),
			UserList(engine),
			UserListByProject(engine),
		))
	return group
}
