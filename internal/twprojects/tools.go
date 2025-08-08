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
			ProjectMemberAdd(engine),
			TasklistCreate(engine),
			TasklistUpdate(engine),
			TasklistDelete(engine),
			TaskCreate(engine),
			TaskUpdate(engine),
			TaskDelete(engine),
			UserCreate(engine),
			UserUpdate(engine),
			UserDelete(engine),
			MilestoneCreate(engine),
			MilestoneUpdate(engine),
			MilestoneDelete(engine),
			CompanyCreate(engine),
			CompanyUpdate(engine),
			CompanyDelete(engine),
			TagCreate(engine),
			TagUpdate(engine),
			TagDelete(engine),
			TeamCreate(engine),
			TeamUpdate(engine),
			TeamDelete(engine),
			CommentCreate(engine),
			CommentUpdate(engine),
			CommentDelete(engine),
			TimelogCreate(engine),
			TimelogUpdate(engine),
			TimelogDelete(engine),
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
			UserGetMe(engine),
			UserList(engine),
			UserListByProject(engine),
			MilestoneGet(engine),
			MilestoneList(engine),
			MilestoneListByProject(engine),
			CompanyGet(engine),
			CompanyList(engine),
			TagGet(engine),
			TagList(engine),
			TeamGet(engine),
			TeamList(engine),
			TeamListByCompany(engine),
			TeamListByProject(engine),
			CommentGet(engine),
			CommentList(engine),
			CommentListByFileVersion(engine),
			CommentListByMilestone(engine),
			CommentListByNotebook(engine),
			CommentListByTask(engine),
			TimelogGet(engine),
			TimelogList(engine),
			TimelogListByProject(engine),
			TimelogListByTask(engine),
		))
	return group
}
