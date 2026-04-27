package twprojects

import (
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

const (
	peopleDescription  = "Users, companies, teams, skills, job roles, and workload management in Teamwork.com."
	timeDescription    = "Time tracking via timelogs, timers, and budget reporting in Teamwork.com."
	contentDescription = "Comments, notebooks, milestones, tags, and activity feeds in Teamwork.com."
)

// Sub-toolset keys for twprojects. These are the valid values for the
// -toolsets flag when selecting Teamwork Projects functionality.
const (
	// ToolsetProjects covers project, category, template, and member management.
	ToolsetProjects toolsets.Method = "twprojects-projects"
	// ToolsetTasks covers task and tasklist management.
	ToolsetTasks toolsets.Method = "twprojects-tasks"
	// ToolsetPeople covers users, companies, teams, skills, job roles, and workload.
	ToolsetPeople toolsets.Method = "twprojects-people"
	// ToolsetTime covers timelogs and timers.
	ToolsetTime toolsets.Method = "twprojects-time"
	// ToolsetContent covers comments, notebooks, milestones, tags, activities, and budgets.
	ToolsetContent toolsets.Method = "twprojects-content"
)

func init() {
	toolsets.RegisterMethod(ToolsetProjects)
	toolsets.RegisterMethod(ToolsetTasks)
	toolsets.RegisterMethod(ToolsetPeople)
	toolsets.RegisterMethod(ToolsetTime)
	toolsets.RegisterMethod(ToolsetContent)
}

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Projects.
func DefaultToolsetGroup(readOnly, allowDelete bool, engine *twapi.Engine) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly)

	// --- projects sub-toolset ---
	projectsWriteTools := []toolsets.ToolWrapper{
		ProjectCategoryCreate(engine),
		ProjectCategoryUpdate(engine),
		ProjectClone(engine),
		ProjectCreate(engine),
		ProjectMemberAdd(engine),
		ProjectTemplateCreate(engine),
		ProjectUpdate(engine),
	}
	if allowDelete {
		projectsWriteTools = append(projectsWriteTools,
			ProjectCategoryDelete(engine),
			ProjectDelete(engine),
		)
	}
	projectsToolset := toolsets.NewToolset(ToolsetProjects, projectDescription).
		AddWriteTools(projectsWriteTools...).
		AddReadTools(
			ProjectCategoryGet(engine),
			ProjectCategoryList(engine),
			ProjectGet(engine),
			ProjectList(engine),
			ProjectTemplateList(engine),
		)
	group.AddToolset(projectsToolset)

	// --- tasks sub-toolset ---
	tasksWriteTools := []toolsets.ToolWrapper{
		TaskComplete(engine),
		TaskCreate(engine),
		TasklistCreate(engine),
		TasklistUpdate(engine),
		TaskUpdate(engine),
		WorkflowCreate(engine),
		WorkflowUpdate(engine),
		WorkflowProjectLink(engine),
		WorkflowStageCreate(engine),
		WorkflowStageUpdate(engine),
		WorkflowStageTaskMove(engine),
	}
	if allowDelete {
		tasksWriteTools = append(tasksWriteTools,
			TaskDelete(engine),
			TasklistDelete(engine),
			WorkflowDelete(engine),
			WorkflowStageDelete(engine),
		)
	}
	tasksToolset := toolsets.NewToolset(ToolsetTasks, taskDescription).
		AddWriteTools(tasksWriteTools...).
		AddReadTools(
			TaskGet(engine),
			TaskList(engine),
			TaskListByProject(engine),
			TaskListByTasklist(engine),
			TasklistGet(engine),
			TasklistList(engine),
			TasklistListByProject(engine),
			WorkflowGet(engine),
			WorkflowList(engine),
			WorkflowStageGet(engine),
			WorkflowStageList(engine),
		)
	tasksToolset.AddPrompts(TaskSkillsAndRolesPrompt(engine))
	group.AddToolset(tasksToolset)

	// --- people sub-toolset ---
	peopleWriteTools := []toolsets.ToolWrapper{
		CompanyCreate(engine),
		CompanyUpdate(engine),
		JobRoleCreate(engine),
		JobRoleUpdate(engine),
		SkillCreate(engine),
		SkillUpdate(engine),
		TeamCreate(engine),
		TeamUpdate(engine),
		UserCreate(engine),
		UserUpdate(engine),
	}
	if allowDelete {
		peopleWriteTools = append(peopleWriteTools,
			CompanyDelete(engine),
			JobRoleDelete(engine),
			SkillDelete(engine),
			TeamDelete(engine),
			UserDelete(engine),
		)
	}
	peopleToolset := toolsets.NewToolset(ToolsetPeople, peopleDescription).
		AddWriteTools(peopleWriteTools...).
		AddReadTools(
			CompanyGet(engine),
			CompanyList(engine),
			IndustryList(engine),
			JobRoleGet(engine),
			JobRoleList(engine),
			SkillGet(engine),
			SkillList(engine),
			TeamGet(engine),
			TeamList(engine),
			TeamListByCompany(engine),
			TeamListByProject(engine),
			UserGet(engine),
			UserGetMe(engine),
			UserList(engine),
			UserListByProject(engine),
			UsersWorkload(engine),
		)
	group.AddToolset(peopleToolset)

	// --- time sub-toolset ---
	timeWriteTools := []toolsets.ToolWrapper{
		TimelogCreate(engine),
		TimelogUpdate(engine),
		TimerComplete(engine),
		TimerCreate(engine),
		TimerPause(engine),
		TimerResume(engine),
		TimerUpdate(engine),
	}
	if allowDelete {
		timeWriteTools = append(timeWriteTools,
			TimelogDelete(engine),
			TimerDelete(engine),
		)
	}
	timeToolset := toolsets.NewToolset(ToolsetTime, timeDescription).
		AddWriteTools(timeWriteTools...).
		AddReadTools(
			ProjectBudgetList(engine),
			TasklistBudgetList(engine),
			TimelogGet(engine),
			TimelogList(engine),
			TimelogListByProject(engine),
			TimelogListByTask(engine),
			TimerGet(engine),
			TimerList(engine),
		)
	if !readOnly {
		timeToolset.AddResources(TimelogCreateAppResource())
	}
	group.AddToolset(timeToolset)

	// --- content sub-toolset ---
	contentWriteTools := []toolsets.ToolWrapper{
		CommentCreate(engine),
		CommentUpdate(engine),
		NotebookCreate(engine),
		NotebookUpdate(engine),
		MilestoneCreate(engine),
		MilestoneUpdate(engine),
		TagCreate(engine),
		TagUpdate(engine),
		MessageCreate(engine),
		MessageUpdate(engine),
		MessageReplyCreate(engine),
		MessageReplyUpdate(engine),
		LinkCreate(engine),
		LinkUpdate(engine),
	}
	if allowDelete {
		contentWriteTools = append(contentWriteTools,
			CommentDelete(engine),
			MilestoneDelete(engine),
			NotebookDelete(engine),
			TagDelete(engine),
			MessageDelete(engine),
			MessageReplyDelete(engine),
			LinkDelete(engine),
		)
	}
	contentToolset := toolsets.NewToolset(ToolsetContent, contentDescription).
		AddWriteTools(contentWriteTools...).
		AddReadTools(
			ActivityList(engine),
			ActivityListByProject(engine),
			CommentGet(engine),
			CommentList(engine),
			CommentListByFileVersion(engine),
			CommentListByMilestone(engine),
			CommentListByNotebook(engine),
			CommentListByTask(engine),
			CommentListByLink(engine),
			MilestoneGet(engine),
			MilestoneList(engine),
			MilestoneListByProject(engine),
			NotebookGet(engine),
			NotebookList(engine),
			TagGet(engine),
			TagList(engine),
			MessageGet(engine),
			MessageList(engine),
			MessageReplyGet(engine),
			MessageReplyList(engine),
			LinkGet(engine),
			LinkList(engine),
		)
	group.AddToolset(contentToolset)

	return group
}
