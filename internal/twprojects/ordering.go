package twprojects

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/helpers"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// ordering carries the order-by vocabulary of one list endpoint. It is the
// twprojects counterpart of twdesk's setPagination, but it binds typed SDK
// filter fields instead of writing a url.Values: every v3 list request models
// ordering as Filters.OrderBy plus Filters.OrderMode, and the SDK renders both
// itself.
//
// A tool declares its vocabulary once, as an ordering var, and both halves of
// the parameter come off that single slice: orderBySchema publishes it as the
// enum a client reads before calling, and param enforces it in the handler.
// Wiring the enum and the validator separately — as the pre-existing custom
// field, custom item and custom item record tools each do, with a hand-written
// []any{...} literal beside a helpers.RestrictValues list — lets the two drift,
// and a schema listing a value the handler rejects is worse than no enum at
// all: the client believes the value is legal and spends a call finding out it
// is not. The enum itself is not optional. Tool definitions carry no output
// schema and the SDK constants are Go identifiers the model never sees, so the
// enum is the only place these names are published; without it a caller guesses
// and learns the vocabulary one rejection at a time.
type ordering[T ~string] struct {
	// entity is the plural noun used in the parameter descriptions.
	entity string

	// values is the accepted order-by vocabulary, in the order it is published.
	values []T
}

func newOrdering[T ~string](entity string, values ...T) ordering[T] {
	return ordering[T]{entity: entity, values: values}
}

// orderBySchema returns the schema for the tool's order_by parameter.
func (o ordering[T]) orderBySchema() *jsonschema.Schema {
	enum := make([]any, len(o.values))
	for i, value := range o.values {
		enum[i] = string(value)
	}
	return &jsonschema.Schema{
		Description: fmt.Sprintf("The field to sort the %s by. Omit to keep the ordering the API applies by "+
			"default.", o.entity),
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Enum: enum},
			{Type: "null"},
		},
	}
}

// param binds order_by and order_mode into the request's filters.
//
// Neither is defaulted. Every one of these endpoints documents its own default
// ordering — tasks sort by due date, activities by date, and so on — and the
// SDK's querySetString omits an empty value, so an unset filter sends no
// parameter and leaves that default in place. Defaulting here instead, the way
// twdesk's setPagination does for the Desk list endpoints, would silently
// reorder every existing caller's results on upgrade.
func (o ordering[T]) param(orderBy *T, orderMode *twapi.OrderMode) helpers.ParamFunc {
	return func(params map[string]any) error {
		if err := helpers.OptionalParam(orderBy, "order_by", helpers.RestrictValues(o.values...))(params); err != nil {
			return err
		}
		return orderModeParam(orderMode)(params)
	}
}

// orderModeSchema returns the schema for an order_mode parameter.
func orderModeSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "The direction to sort the results in.",
		AnyOf: []*jsonschema.Schema{
			{Type: "string", Enum: []any{
				string(twapi.OrderModeAscending),
				string(twapi.OrderModeDescending),
			}},
			{Type: "null"},
		},
	}
}

// orderModeParam binds order_mode on its own, for the endpoints that accept a
// direction but no order-by field (job roles, skills and custom item fields
// each sort by a fixed column).
func orderModeParam(orderMode *twapi.OrderMode) helpers.ParamFunc {
	return helpers.OptionalParam(orderMode, "order_mode", helpers.RestrictValues(
		twapi.OrderModeAscending,
		twapi.OrderModeDescending,
	))
}

// orderByFieldIDSchema returns the schema for the companion parameter of a
// field-valued order-by, which names the field to sort on and is meaningless
// without it. The task, project and company endpoints take it alongside their
// "customfield" value; custom item records take it alongside "customitemfield".
// An endpoint carrying such a value in its vocabulary but exposing no companion
// parameter can be asked to sort by a custom field without being told which
// one, which the API answers by ignoring the ordering.
func orderByFieldIDSchema(entity, orderByValue string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf("The custom field to sort the %s by. Required when order_by is %q, and ignored "+
			"otherwise.", entity, orderByValue),
		AnyOf: []*jsonschema.Schema{
			{Type: "integer", Minimum: new(1.0)},
			{Type: "null"},
		},
	}
}

// The order-by vocabulary of every v3 list endpoint that has one, taken from
// the SDK constants so a vocabulary the API widens arrives with an SDK bump
// rather than needing to be rediscovered here.
var (
	activityOrdering = newOrdering("activities",
		projects.ActivityOrderByDate,
		projects.ActivityOrderByProjectID,
		projects.ActivityOrderByUserID,
		projects.ActivityOrderByActivityTypes,
		projects.ActivityOrderByID,
	)

	calendarOrdering = newOrdering("calendars",
		projects.CalendarOrderByName,
		projects.CalendarOrderByID,
	)

	calendarEventOrdering = newOrdering("calendar events",
		projects.CalendarEventOrderByStartTime,
		projects.CalendarEventOrderByUpdated,
		projects.CalendarEventOrderByID,
	)

	customFieldOrdering = newOrdering("custom fields",
		projects.CustomFieldOrderByName,
		projects.CustomFieldOrderByProject,
		projects.CustomFieldOrderByDateCreated,
		projects.CustomFieldOrderByDateUpdated,
		projects.CustomFieldOrderByID,
	)

	customItemOrdering = newOrdering("custom items",
		projects.CustomItemOrderByName,
		projects.CustomItemOrderByID,
	)

	customItemRecordOrdering = newOrdering("custom item records",
		projects.CustomItemRecordOrderByDisplayOrder,
		projects.CustomItemRecordOrderByName,
		projects.CustomItemRecordOrderByCustomItemField,
		projects.CustomItemRecordOrderByID,
	)

	commentOrdering = newOrdering("comments",
		projects.CommentOrderByAll,
		projects.CommentOrderByDate,
		projects.CommentOrderByProject,
		projects.CommentOrderByUser,
		projects.CommentOrderByType,
		projects.CommentOrderByID,
	)

	companyOrdering = newOrdering("companies",
		projects.CompanyOrderByName,
		projects.CompanyOrderByAccounts,
		projects.CompanyOrderByClients,
		projects.CompanyOrderByCollaborators,
		projects.CompanyOrderByContacts,
		projects.CompanyOrderByProjects,
		projects.CompanyOrderByTasks,
		projects.CompanyOrderByCountry,
		projects.CompanyOrderByHealth,
		projects.CompanyOrderByWebsite,
		projects.CompanyOrderByEmail,
		projects.CompanyOrderByPhone,
		projects.CompanyOrderByFax,
		projects.CompanyOrderByIndustry,
		projects.CompanyOrderByDateAdded,
		projects.CompanyOrderByOwnerCompany,
		projects.CompanyOrderByOwnerName,
		projects.CompanyOrderByTasksCompletion,
		projects.CompanyOrderByCustomField,
		projects.CompanyOrderByID,
	)

	messageOrdering = newOrdering("messages",
		projects.MessageOrderByCreatedAt,
		projects.MessageOrderByUpdatedAt,
		projects.MessageOrderByCategory,
		projects.MessageOrderByProject,
		projects.MessageOrderByCreatedBy,
		projects.MessageOrderByUnread,
		projects.MessageOrderByID,
	)

	messageReplyOrdering = newOrdering("message replies",
		projects.MessageReplyOrderByCreatedAt,
		projects.MessageReplyOrderByID,
	)

	milestoneOrdering = newOrdering("milestones",
		projects.MilestoneOrderByDate,
		projects.MilestoneOrderByDateOnly,
		projects.MilestoneOrderByName,
		projects.MilestoneOrderByProject,
		projects.MilestoneOrderByUser,
		projects.MilestoneOrderByDateCreated,
		projects.MilestoneOrderByDateUpdated,
		projects.MilestoneOrderByID,
	)

	notebookOrdering = newOrdering("notebooks",
		projects.NotebookOrderByName,
		projects.NotebookOrderByProject,
		projects.NotebookOrderByDateCreated,
		projects.NotebookOrderByDateUpdated,
		projects.NotebookOrderByCategory,
		projects.NotebookOrderByID,
	)

	projectOrdering = newOrdering("projects",
		projects.ProjectOrderByBudgetUsed,
		projects.ProjectOrderByCategoryName,
		projects.ProjectOrderByCompanyName,
		projects.ProjectOrderByCreatorName,
		projects.ProjectOrderByCustomField,
		projects.ProjectOrderByDateCreated,
		projects.ProjectOrderByDueDate,
		projects.ProjectOrderByHealth,
		projects.ProjectOrderByLastActivity,
		projects.ProjectOrderByLastWorkedOn,
		projects.ProjectOrderByMobileSpecial,
		projects.ProjectOrderByName,
		projects.ProjectOrderByNameCaseInsensitive,
		projects.ProjectOrderByOwnerCompany,
		projects.ProjectOrderByOwnerName,
		projects.ProjectOrderByStarred,
		projects.ProjectOrderByStarredCompanyName,
		projects.ProjectOrderByStarredFirst,
		projects.ProjectOrderByStartDate,
		projects.ProjectOrderByStatus,
		projects.ProjectOrderByTaskCompletion,
		projects.ProjectOrderByID,
	)

	tagOrdering = newOrdering("tags",
		projects.TagOrderByName,
		projects.TagOrderByCount,
		projects.TagOrderByProject,
		projects.TagOrderByColor,
		projects.TagOrderByDateLastUpdated,
		projects.TagOrderByProjectDateLastUsed,
		projects.TagOrderByID,
	)

	taskOrdering = newOrdering("tasks",
		projects.TaskOrderByID,
		projects.TaskOrderByStartDate,
		projects.TaskOrderByCreatedAt,
		projects.TaskOrderByPriority,
		projects.TaskOrderByProject,
		projects.TaskOrderByFlattenedTasklist,
		projects.TaskOrderByCompany,
		projects.TaskOrderByManual,
		projects.TaskOrderByActive,
		projects.TaskOrderByCompletedAt,
		projects.TaskOrderByDueStartDate,
		projects.TaskOrderByAllDates,
		projects.TaskOrderByTasklistName,
		projects.TaskOrderByTasklistDisplayOrder,
		projects.TaskOrderByTasklistID,
		projects.TaskOrderByDueDate,
		projects.TaskOrderByUpdatedAt,
		projects.TaskOrderByTaskName,
		projects.TaskOrderByCreatedBy,
		projects.TaskOrderByCompletedBy,
		projects.TaskOrderByAssignedTo,
		projects.TaskOrderByTaskStatus,
		projects.TaskOrderByTaskDueDate,
		projects.TaskOrderByCustomField,
		projects.TaskOrderByEstimatedTime,
		projects.TaskOrderByBoardColumn,
		projects.TaskOrderByTaskGroupID,
		projects.TaskOrderByTaskGroupName,
		projects.TaskOrderByTaskGroup,
		projects.TaskOrderByDisplayOrder,
		projects.TaskOrderByProjectManual,
		projects.TaskOrderByStageDisplayOrder,
		projects.TaskOrderByStage,
		projects.TaskOrderByParentTask,
	)

	tasklistOrdering = newOrdering("task lists",
		projects.TasklistOrderByDisplayOrder,
		projects.TasklistOrderByName,
		projects.TasklistOrderByStatus,
		projects.TasklistOrderByCreatedAt,
		projects.TasklistOrderByUpdatedAt,
		projects.TasklistOrderByProject,
		projects.TasklistOrderByID,
	)

	tasklistBudgetOrdering = newOrdering("task list budgets",
		projects.TasklistBudgetOrderByDateCreated,
		projects.TasklistBudgetOrderByDisplayOrder,
		projects.TasklistBudgetOrderByID,
	)

	teamOrdering = newOrdering("teams",
		projects.TeamOrderByName,
		projects.TeamOrderByPicker,
		projects.TeamOrderByDateAdded,
	)

	timelogOrdering = newOrdering("timelogs",
		projects.TimelogOrderByCompany,
		projects.TimelogOrderByDate,
		projects.TimelogOrderByDateUpdated,
		projects.TimelogOrderByProject,
		projects.TimelogOrderByTask,
		projects.TimelogOrderByTasklist,
		projects.TimelogOrderByUser,
		projects.TimelogOrderByDescription,
		projects.TimelogOrderByBilled,
		projects.TimelogOrderByBillable,
		projects.TimelogOrderByTimeSpent,
		projects.TimelogOrderByID,
	)

	timeReportOrdering = newOrdering("summary rows",
		projects.TimeReportOrderByName,
		projects.TimeReportOrderByLoggedTime,
		projects.TimeReportOrderByBillableTime,
		projects.TimeReportOrderByNonBillableTime,
		projects.TimeReportOrderByBilledTime,
		projects.TimeReportOrderByBudget,
	)

	userOrdering = newOrdering("users",
		projects.UserOrderByName,
		projects.UserOrderByNameCaseInsensitive,
		projects.UserOrderByCompany,
		projects.UserOrderByID,
	)

	workflowStageOrdering = newOrdering("workflow stages",
		projects.WorkflowStageOrderByID,
		projects.WorkflowStageOrderByName,
		projects.WorkflowStageOrderByDisplayOrder,
	)
)
