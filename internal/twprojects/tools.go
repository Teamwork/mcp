package twprojects

import (
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Projects.
func DefaultToolsetGroup(readOnly, allowDelete bool, engine *twapi.Engine) *toolsets.ToolsetGroup {
	writeTools := []toolsets.ToolWrapper{
		ProjectCreate(engine),
		ProjectUpdate(engine),
		ProjectMemberAdd(engine),
		ProjectCategoryCreate(engine),
		ProjectCategoryUpdate(engine),
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
		NotebookCreate(engine),
		NotebookUpdate(engine),
		SkillCreate(engine),
		SkillUpdate(engine),
		JobRoleCreate(engine),
		JobRoleUpdate(engine),
	}
	if allowDelete {
		writeTools = append(writeTools, []toolsets.ToolWrapper{
			ProjectDelete(engine),
			ProjectCategoryDelete(engine),
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
			NotebookDelete(engine),
			SkillDelete(engine),
			JobRoleDelete(engine),
		}...)
	}

	group := toolsets.NewToolsetGroup(readOnly)
	projectToolset := toolsets.NewToolset("projects", projectDescription).
		AddWriteTools(writeTools...).
		AddReadTools(
			ProjectGet(engine),
			ProjectList(engine),
			ProjectCategoryGet(engine),
			ProjectCategoryList(engine),
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
			UsersWorkload(engine),
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
			NotebookGet(engine),
			NotebookList(engine),
			IndustryList(engine),
			SkillGet(engine),
			SkillList(engine),
			JobRoleGet(engine),
			JobRoleList(engine),
		)
	if !readOnly {
		projectToolset.AddResourceTemplates(
			TimelogCreateAppResourceTemplate(),
		)
	}
	projectToolset.AddPrompts(
		TaskSkillsAndRolesPrompt(engine),
	)

	group.AddToolset(projectToolset)
	return group
}
