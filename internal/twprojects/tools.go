package twprojects

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Projects.
func DefaultToolsetGroup(readOnly, allowDelete bool, engine *twapi.Engine) *toolsets.ToolsetGroup {
	writeTools := []server.ServerTool{
		ProjectCreate(engine),
		ProjectUpdate(engine),
		ProjectMemberAdd(engine),
		TasklistCreate(engine),
		TasklistUpdate(engine),
		TaskCreate(engine),
		TaskUpdate(engine),
		UserCreate(engine),
		UserUpdate(engine),
		MilestoneCreate(engine),
		MilestoneUpdate(engine),
		CompanyCreate(engine),
		CompanyUpdate(engine),
		TagCreate(engine),
		TagUpdate(engine),
		TeamCreate(engine),
		TeamUpdate(engine),
		CommentCreate(engine),
		CommentUpdate(engine),
		TimelogCreate(engine),
		TimelogUpdate(engine),
		TimerCreate(engine),
		TimerUpdate(engine),
		TimerPause(engine),
		TimerResume(engine),
		TimerComplete(engine),
		RateInstallationUserUpdate(engine),
		RateInstallationUserBulkUpdate(engine),
		RateProjectUpdate(engine),
		RateProjectAndUsersUpdate(engine),
		RateProjectUserUpdate(engine),
	}
	if allowDelete {
		writeTools = append(writeTools, []server.ServerTool{
			ProjectDelete(engine),
			TasklistDelete(engine),
			TaskDelete(engine),
			UserDelete(engine),
			MilestoneDelete(engine),
			CompanyDelete(engine),
			TagDelete(engine),
			TeamDelete(engine),
			CommentDelete(engine),
			TimelogDelete(engine),
			TimerDelete(engine),
		}...)
	}

	group := toolsets.NewToolsetGroup(readOnly)
	group.AddToolset(toolsets.NewToolset("projects", projectDescription).
		AddWriteTools(writeTools...).
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
			TimerGet(engine),
			TimerList(engine),
			ActivityList(engine),
			ActivityListByProject(engine),
			RateUserGet(engine),
			RateInstallationUserList(engine),
			RateInstallationUserGet(engine),
			RateProjectGet(engine),
			RateProjectUserList(engine),
			RateProjectUserGet(engine),
			RateProjectUserHistoryGet(engine),
		))
	return group
}
