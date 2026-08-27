package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodAllocationCreate     toolsets.Method = "twprojects-create_allocation"
	MethodAllocationUpdate     toolsets.Method = "twprojects-update_allocation"
	MethodAllocationDelete     toolsets.Method = "twprojects-delete_allocation"
	MethodAllocationRestore    toolsets.Method = "twprojects-restore_allocation"
	MethodAllocationTaskLink   toolsets.Method = "twprojects-link_task_to_allocation"
	MethodAllocationTaskUnlink toolsets.Method = "twprojects-unlink_task_from_allocation"
	MethodAllocationGet        toolsets.Method = "twprojects-get_allocation"
	MethodAllocationList       toolsets.Method = "twprojects-list_allocations"
)

// allocationWebPath is the in-app route of an allocation's quick view.
const allocationWebPath = "/app/allocations"

var (
	allocationGetOutputSchema  *jsonschema.Schema
	allocationListOutputSchema *jsonschema.Schema
)

// allocationOrdering is the order-by vocabulary of the allocations list
// endpoint. The values are lowercase and unseparated, which is what the
// endpoint accepts.
var allocationOrdering = newOrdering("allocations",
	projects.AllocationOrderByStartDate,
	projects.AllocationOrderByEndDate,
	projects.AllocationOrderByProject,
	projects.AllocationOrderByAssignedUser,
	projects.AllocationOrderByID,
)

// allocationProjectStatuses is the project-status filter vocabulary.
var allocationProjectStatuses = []projects.AllocationProjectStatus{
	projects.AllocationProjectStatusActive,
	projects.AllocationProjectStatusCurrent,
	projects.AllocationProjectStatusLate,
	projects.AllocationProjectStatusUpcoming,
	projects.AllocationProjectStatusCompleted,
	projects.AllocationProjectStatusDeleted,
}

func init() {
	var err error

	// generate the output schemas only once. WithDateTypeSchema is mandatory
	// here: the allocation model carries twapi.Date, which jsonschema.For would
	// otherwise describe by time.Time's unexported fields — an opaque object,
	// while MarshalJSON writes a string. Every row would then fail output-schema
	// validation at any validating client.
	allocationGetOutputSchema, err = jsonschema.For[projects.AllocationGetResponse](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for AllocationGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(allocationGetOutputSchema)
	allocationListOutputSchema, err = jsonschema.For[projects.AllocationListResponse](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for AllocationListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(allocationListOutputSchema)
}

// allocationFinancialDetailsSchema returns the schema for the opt-in that adds
// forecasted revenue and cost to the response.
func allocationFinancialDetailsSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Include forecasted revenue and cost for the allocated time. Requires the scheduler rates " +
			"entitlement and financial permission on the project; without either, the request still succeeds but the " +
			"figures are withheld. Read canViewFinancialDetails on each row to tell a withheld figure from a zero one.",
		AnyOf: []*jsonschema.Schema{
			{Type: "boolean"},
			{Type: "null"},
		},
	}
}

// allocationSideloads returns the sideloads a verbose read requests.
// financialDetails is deliberately absent: it is gated, and it is money data
// the caller has to ask for.
func allocationSideloads(financialDetails bool) []projects.AllocationSideload {
	sideloads := []projects.AllocationSideload{
		projects.AllocationSideloadProjects,
		projects.AllocationSideloadAssignee,
	}
	if financialDetails {
		sideloads = append(sideloads, projects.AllocationSideloadFinancialDetails)
	}
	return sideloads
}

// allocationUpsertParams binds the writable attributes shared by create and
// update. Required-ness differs between the two, so the two required-on-create
// fields are bound by the callers instead.
func allocationUpsertParams(upsert *projects.AllocationUpsert) []helpers.ParamFunc {
	return []helpers.ParamFunc{
		helpers.OptionalNumericPointerParam(&upsert.ProjectID, "project_id"),
		helpers.OptionalNumericPointerParam(&upsert.AssignedUserID, "assigned_user_id"),
		helpers.OptionalPointerParam(&upsert.Title, "title"),
		helpers.OptionalPointerParam(&upsert.Description, "description"),
		helpers.OptionalDatePointerParam(&upsert.StartDate, "start_date"),
		helpers.OptionalDatePointerParam(&upsert.EndDate, "end_date"),
		helpers.OptionalNumericPointerParam(&upsert.SecondsPerDay, "seconds_per_day"),
		helpers.OptionalPointerParam(&upsert.Color, "color"),
		helpers.OptionalPointerParam(&upsert.IsBillable, "is_billable"),
		helpers.OptionalPointerParam(&upsert.IgnoreCollisions, "ignore_collisions"),
		helpers.OptionalPointerParam(&upsert.InformOfOverAllocation, "inform_of_over_allocation"),
		helpers.OptionalNumericListParam(&upsert.LinkedTaskIDs, "linked_task_ids"),
	}
}

// allocationUpsertProperties returns the input-schema properties shared by
// create and update. required marks the create-only requirements, which the
// update tool publishes as nullable instead.
func allocationUpsertProperties(required bool) map[string]*jsonschema.Schema {
	nullable := func(schema *jsonschema.Schema) *jsonschema.Schema {
		if required {
			return schema
		}
		relaxed := *schema
		relaxed.Type = ""
		relaxed.Enum = nil
		relaxed.Minimum = nil
		relaxed.MaxLength = nil
		relaxed.AnyOf = []*jsonschema.Schema{
			{Type: schema.Type, Enum: schema.Enum, Minimum: schema.Minimum, MaxLength: schema.MaxLength},
			{Type: "null"},
		}
		return &relaxed
	}

	return map[string]*jsonschema.Schema{
		"project_id": nullable(&jsonschema.Schema{
			Type:        "integer",
			Minimum:     new(1.0),
			Description: "The ID of the project to commit the time to.",
		}),
		"assigned_user_id": nullable(&jsonschema.Schema{
			Type:    "integer",
			Minimum: new(1.0),
			Description: "The ID of the user whose time is committed. Accepts a real person or a placeholder user " +
				"— a stand-in used to plan work before the person who will do it is known. Nothing in the response " +
				"distinguishes the two, so confirm which one an ID refers to before reporting who is booked.",
		}),
		"title": nullable(&jsonschema.Schema{
			Type:        "string",
			MaxLength:   new(100),
			Description: "The name of the allocation, at most 100 characters.",
		}),
		"start_date": nullable(&jsonschema.Schema{
			Type:        "string",
			Description: "The first day of the allocation (format: YYYY-MM-DD).",
		}),
		"end_date": nullable(&jsonschema.Schema{
			Type: "string",
			Description: "The last day of the allocation (format: YYYY-MM-DD). Must not precede start_date. " +
				"Extending it ADDS committed time rather than spreading the existing total: the per-day rate is " +
				"what is held constant, so the total is the working days in the range times that rate.",
		}),
		"seconds_per_day": nullable(&jsonschema.Schema{
			Type:    "integer",
			Minimum: new(60.0),
			Description: "The time committed on each working day of the range, in SECONDS — 4 hours a day is " +
				"14400. Must be between 60 (one minute) and 86400 (24 hours). Seconds rather than hours because " +
				"the hours form is a float and rounds. This rate is what is held constant: the total is the " +
				"working days in the range times this, so widening the range adds committed time.",
		}),
		"color": nullable(&jsonschema.Schema{
			Type:        "string",
			Description: "The allocation's colour as six hexadecimal digits, with or without a leading '#'.",
		}),
		"description": {
			Description: "An optional description of the allocation, at most 255 characters.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string", MaxLength: new(255)},
				{Type: "null"},
			},
		},
		"is_billable": {
			Description: "Whether the allocated time can be charged to a client.",
			AnyOf: []*jsonschema.Schema{
				{Type: "boolean"},
				{Type: "null"},
			},
		},
		"inform_of_over_allocation": {
			Description: "Accept a change that puts the user over their capacity and report it, rather than " +
				"rejecting it. Defaults to true, and the result says so when it happens. Turning it off means an " +
				"over-allocating change is refused outright.",
			AnyOf: []*jsonschema.Schema{
				{Type: "boolean"},
				{Type: "null"},
			},
		},
		"ignore_collisions": {
			Description: "Skip the capacity check altogether. Prefer inform_of_over_allocation: this one also " +
				"lets the change through, but suppresses the over-allocation report with it, so nobody is told " +
				"the person is over-booked. It takes precedence when both are set.",
			AnyOf: []*jsonschema.Schema{
				{Type: "boolean"},
				{Type: "null"},
			},
		},
		"linked_task_ids": {
			Description: "The tasks to associate with the allocation. This REPLACES the whole set of linked tasks, " +
				"so send every task that should stay linked. To add or remove one task without touching the rest, " +
				"use twprojects-link_task_to_allocation or twprojects-unlink_task_from_allocation instead.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
	}
}

// AllocationCreate creates an allocation in Teamwork.com.
func AllocationCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	properties := allocationUpsertProperties(true)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationCreate),
			Description: "Commit a user's time to a project over a date range (a scheduler allocation). This is " +
				"planned time, a separate plane from task estimates and logged time.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Allocation",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required: []string{
					"project_id", "assigned_user_id", "title", "start_date", "end_date", "seconds_per_day", "color",
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationCreateRequest projects.AllocationCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				allocationUpsertParams(&allocationCreateRequest.Allocation)...,
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Default the report on. Left off, a change that overruns the user's
			// capacity is refused, and the caller's only other lever is
			// ignore_collisions, which lets it through while hiding that it
			// happened — the worst of the three outcomes for a caller acting on
			// someone else's schedule.
			if allocationCreateRequest.Allocation.InformOfOverAllocation == nil {
				allocationCreateRequest.Allocation.InformOfOverAllocation = new(true)
			}

			allocation, err := projects.AllocationCreate(ctx, engine, allocationCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create allocation")
			}
			if allocation.Allocation.OverAllocated {
				return helpers.NewToolResultText("Allocation created successfully with ID %d. Note that it puts "+
					"the assigned user over their capacity for this period.", allocation.Allocation.ID), nil
			}
			return helpers.NewToolResultText("Allocation created successfully with ID %d",
				allocation.Allocation.ID), nil
		},
	}
}

// AllocationUpdate updates an allocation in Teamwork.com.
func AllocationUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	properties := allocationUpsertProperties(false)
	properties["id"] = &jsonschema.Schema{
		Type:        "integer",
		Minimum:     new(1.0),
		Description: "The ID of the allocation to update.",
	}

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationUpdate),
			Description: "Update an allocation. Changing end_date ADDS or REMOVES committed time rather than " +
				"redistributing it, because the per-day rate is what is held constant.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Allocation",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required:   []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationUpdateRequest projects.AllocationUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			params := append(
				[]helpers.ParamFunc{helpers.RequiredNumericParam(&allocationUpdateRequest.Path.ID, "id")},
				allocationUpsertParams(&allocationUpdateRequest.Allocation)...,
			)
			if err := helpers.ParamGroup(arguments, params...); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if allocationUpdateRequest.Allocation.InformOfOverAllocation == nil {
				allocationUpdateRequest.Allocation.InformOfOverAllocation = new(true)
			}

			allocation, err := projects.AllocationUpdate(ctx, engine, allocationUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update allocation")
			}
			if allocation.Allocation.OverAllocated {
				return helpers.NewToolResultText("Allocation updated successfully. Note that it puts the assigned " +
					"user over their capacity for this period."), nil
			}
			return helpers.NewToolResultText("Allocation updated successfully"), nil
		},
	}
}

// AllocationDelete deletes an allocation in Teamwork.com.
func AllocationDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationDelete),
			Description: "Delete an allocation. This is a soft delete: the allocation can be found again with " +
				"show_deleted on twprojects-list_allocations and brought back with twprojects-restore_allocation.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Allocation",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the allocation to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationDeleteRequest projects.AllocationDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&allocationDeleteRequest.Path.ID, "id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// HardDelete is deliberately not exposed: a recoverable delete is what
			// pairs with twprojects-restore_allocation.
			if _, err := projects.AllocationDelete(ctx, engine, allocationDeleteRequest); err != nil {
				return helpers.HandleAPIError(err, "failed to delete allocation")
			}
			return helpers.NewToolResultText("Allocation deleted successfully"), nil
		},
	}
}

// AllocationRestore restores a soft-deleted allocation in Teamwork.com.
func AllocationRestore(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationRestore),
			Description: "Restore a deleted allocation. Find the ID first with twprojects-list_allocations and " +
				"show_deleted set, since a deleted allocation is otherwise not returned.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Restore Allocation",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the allocation to restore.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationRestoreRequest projects.AllocationRestoreRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&allocationRestoreRequest.Path.ID, "id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if _, err := projects.AllocationRestore(ctx, engine, allocationRestoreRequest); err != nil {
				return helpers.HandleAPIError(err, "failed to restore allocation")
			}
			return helpers.NewToolResultText("Allocation restored successfully"), nil
		},
	}
}

// AllocationTaskLink links a task to an allocation in Teamwork.com.
func AllocationTaskLink(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationTaskLink),
			Description: "Link one task to an allocation, showing what task work sits behind the committed time. " +
				"The task and the allocation must be in the same project. This adds a single link and leaves the " +
				"allocation's other links alone, unlike linked_task_ids on twprojects-update_allocation, which " +
				"replaces the whole set.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Link Task To Allocation",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"allocation_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the allocation.",
					},
					"task_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the task to link. Must belong to the allocation's project.",
					},
				},
				Required: []string{"allocation_id", "task_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationTaskLinkRequest projects.AllocationTaskLinkRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&allocationTaskLinkRequest.Path.AllocationID, "allocation_id"),
				helpers.RequiredNumericParam(&allocationTaskLinkRequest.Path.TaskID, "task_id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if _, err := projects.AllocationTaskLink(ctx, engine, allocationTaskLinkRequest); err != nil {
				return helpers.HandleAPIError(err, "failed to link task to allocation")
			}
			return helpers.NewToolResultText("Task linked to allocation successfully"), nil
		},
	}
}

// AllocationTaskUnlink unlinks a task from an allocation in Teamwork.com.
func AllocationTaskUnlink(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationTaskUnlink),
			Description: "Remove the link between one task and an allocation. Only the association is removed: " +
				"both the task and the allocation are left in place. This removes a single link and leaves the " +
				"allocation's other links alone.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Unlink Task From Allocation",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"allocation_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the allocation.",
					},
					"task_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the task to unlink.",
					},
				},
				Required: []string{"allocation_id", "task_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationTaskUnlinkRequest projects.AllocationTaskUnlinkRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&allocationTaskUnlinkRequest.Path.AllocationID, "allocation_id"),
				helpers.RequiredNumericParam(&allocationTaskUnlinkRequest.Path.TaskID, "task_id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if _, err := projects.AllocationTaskUnlink(ctx, engine, allocationTaskUnlinkRequest); err != nil {
				return helpers.HandleAPIError(err, "failed to unlink task from allocation")
			}
			return helpers.NewToolResultText("Task unlinked from allocation successfully"), nil
		},
	}
}

// AllocationGet retrieves an allocation in Teamwork.com.
func AllocationGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationGet),
			Description: "Get an allocation. linkedTaskEstimatedTime counts each linked task whole, and a task " +
				"can sit behind more than one allocation, so it must not be summed across allocations.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Allocation",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the allocation to get.",
					},
					"include_financial_details": allocationFinancialDetailsSchema(),
					"fields":                    helpers.FieldsSchema[projects.Allocation]("allocation"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(allocationGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var allocationGetRequest projects.AllocationGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			var financialDetails bool
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&allocationGetRequest.Path.ID, "id"),
				helpers.OptionalParam(&financialDetails, "include_financial_details"),
				helpers.OptionalFieldsParam[projects.Allocation](&allocationGetRequest.Fields.Allocation, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(allocationGetRequest.Fields.Allocation) > 0 {
				// A selection wins over the sideloads: sideloading would hand back the
				// bulk the selection exists to avoid. financialDetails is kept, since
				// it is an attribute of the allocation the caller asked for by name.
				allocationGetRequest.Include = nil
				if financialDetails {
					allocationGetRequest.Include = []projects.AllocationSideload{
						projects.AllocationSideloadFinancialDetails,
					}
				}
				return helpers.NewRawToolResult(ctx, engine, allocationGetRequest, "failed to get allocation",
					helpers.WebLinkerWithIDPathBuilder(allocationWebPath),
				)
			}
			allocationGetRequest.Include = allocationSideloads(financialDetails)

			allocation, err := projects.AllocationGet(ctx, engine, allocationGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get allocation")
			}

			encoded, err := json.Marshal(allocation)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder(allocationWebPath),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, allocation,
					helpers.WebLinkerWithIDPathBuilder(allocationWebPath),
				),
			}, nil
		},
	}
}

// AllocationList lists allocations in Teamwork.com.
func AllocationList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodAllocationList),
			Description: "List scheduler allocations — who is committed to which project, and when. ALWAYS pass " +
				"start_date and end_date: with neither set the endpoint returns only today through 30 days from " +
				"today, and says nothing about having narrowed the range, so an unbounded call silently answers a " +
				"question about a wider period with one month of data. Allocations are planned time and a separate " +
				"plane from task estimates and logged time; the two are not summed. linkedTaskEstimatedTime " +
				"counts each linked task whole and must not be summed across allocations.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Allocations",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"start_date": helpers.DateFilterSchema(
						"Return allocations overlapping this day onwards (format: YYYY-MM-DD). Defaults to today " +
							"when omitted."),
					"end_date": helpers.DateFilterSchema(
						"Return allocations overlapping up to and including this day (format: YYYY-MM-DD). " +
							"Defaults to 30 days from today when omitted."),
					"search_term": {
						Description: "A search term to filter allocations by title.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"assigned_user_ids": {
						Description: "Only return allocations assigned to these users. Accepts real people and " +
							"placeholder users alike.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"assigned_user_team_ids": {
						Description: "Only return allocations whose assigned user belongs to one of these teams.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "Only return allocations on these projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_owner_ids": {
						Description: "Only return allocations on projects owned by these users.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_category_ids": {
						Description: "Only return allocations on projects in these categories.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_company_ids": {
						Description: "Only return allocations on projects belonging to these companies.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_tag_ids": {
						Description: "Only return allocations on projects carrying these tags.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"match_all_project_tags": {
						Description: "Require a project to carry every tag in project_tag_ids rather than any of them.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"project_status": {
						Description: "Only return allocations on projects with this status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{
								string(projects.AllocationProjectStatusActive),
								string(projects.AllocationProjectStatusCurrent),
								string(projects.AllocationProjectStatusLate),
								string(projects.AllocationProjectStatusUpcoming),
								string(projects.AllocationProjectStatusCompleted),
								string(projects.AllocationProjectStatusDeleted),
							}},
							{Type: "null"},
						},
					},
					"updated_after": helpers.DateTimeFilterSchema(
						"Only return allocations updated strictly after this moment; the boundary itself " +
							"does not match."),
					"deleted_after": helpers.DateTimeFilterSchema(
						"Only return allocations deleted at or after this moment; the boundary itself " +
							"matches, unlike updated_after. Pair it with show_deleted, which is " +
							"what switches the results to deleted allocations."),
					"show_deleted": {
						Description: "Return ONLY deleted allocations instead of the active ones — this replaces " +
							"the result set rather than adding to it, so a call with this set says nothing about " +
							"what is currently scheduled. Deleting an allocation is a soft delete, and this is how " +
							"a deleted one is found again.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_financial_details": allocationFinancialDetailsSchema(),
					"order_by":                  allocationOrdering.orderBySchema(),
					"order_mode":                orderModeSchema(),
					"page":                      helpers.PageSchema(),
					"page_size":                 helpers.PageSizeSchema(),
					"verbose":                   helpers.VerboseSchema(),
					"count_only":                helpers.CountOnlySchema("allocations"),
					"fields":                    helpers.FieldsSchema[projects.Allocation]("allocation"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(allocationListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			allocationListRequest := projects.NewAllocationListRequest()

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			var financialDetails bool
			filters := &allocationListRequest.Filters
			err := helpers.ParamGroup(arguments,
				helpers.OptionalDatePointerParam(&filters.StartDate, "start_date"),
				helpers.OptionalDatePointerParam(&filters.EndDate, "end_date"),
				helpers.OptionalParam(&filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&filters.AssignedUserIDs, "assigned_user_ids"),
				helpers.OptionalNumericListParam(&filters.AssignedUserTeamIDs, "assigned_user_team_ids"),
				helpers.OptionalNumericListParam(&filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericListParam(&filters.ProjectOwnerIDs, "project_owner_ids"),
				helpers.OptionalNumericListParam(&filters.ProjectCategoryIDs, "project_category_ids"),
				helpers.OptionalNumericListParam(&filters.ProjectCompanyIDs, "project_company_ids"),
				helpers.OptionalNumericListParam(&filters.ProjectTagIDs, "project_tag_ids"),
				helpers.OptionalPointerParam(&filters.MatchAllProjectTags, "match_all_project_tags"),
				helpers.OptionalParam(&filters.ProjectStatus, "project_status",
					helpers.RestrictValues(allocationProjectStatuses...)),
				helpers.OptionalTimePointerParam(&filters.UpdatedAfter, "updated_after"),
				helpers.OptionalTimePointerParam(&filters.DeletedAfter, "deleted_after"),
				helpers.OptionalPointerParam(&filters.ShowDeleted, "show_deleted"),
				allocationOrdering.param(&filters.OrderBy, &filters.OrderMode),
				helpers.OptionalNumericParam(&filters.Page, "page"),
				helpers.OptionalNumericParam(&filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalParam(&financialDetails, "include_financial_details"),
				helpers.OptionalFieldsParam[projects.Allocation](&filters.Fields.Allocations, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Every filter is bound by this point, and no row shaping has been
			// applied yet, so a count query carries the filters and nothing else.
			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, allocationListRequest, "failed to count allocations")
			}

			switch {
			case len(filters.Fields.Allocations) > 0:
				// A selection wins over verbose, and asks for no sideloads beyond the
				// financial details the caller opted into explicitly.
				if financialDetails {
					filters.Include = []projects.AllocationSideload{projects.AllocationSideloadFinancialDetails}
				}
			case verbose:
				filters.Include = allocationSideloads(financialDetails)
			default:
				filters.Fields.Allocations = []projects.AllocationField{
					projects.AllocationFieldID,
					projects.AllocationFieldTitle,
				}
				if financialDetails {
					filters.Include = []projects.AllocationSideload{projects.AllocationSideloadFinancialDetails}
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, allocationListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list allocations")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list allocations"),
					"failed to list allocations",
				)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder(allocationWebPath))
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
