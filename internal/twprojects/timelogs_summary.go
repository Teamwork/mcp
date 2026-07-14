package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodSummarizeTimelogs toolsets.Method = "twprojects-summarize_timelogs"
)

// timelogSummaryPageSize is the page size used when paginating the underlying
// time report endpoint. It is deliberately large so the vast majority of
// windows resolve in a single request.
const timelogSummaryPageSize = 500

// timelogSummaryMaxPages caps internal pagination. If the report still has more
// pages after this many, the tool fails loudly rather than returning partial
// totals — the caller is told to narrow the window or add filters.
const timelogSummaryMaxPages = 10

// timelogSummaryColumns holds the ten time aggregate columns shared by the
// totals block and every group row. Minutes are exact integers (authoritative,
// always reconcile); hours are minutes ÷ 60 rounded to two decimals, for
// narration and rate math.
type timelogSummaryColumns struct {
	LoggedMinutes           int64   `json:"loggedMinutes"`
	LoggedHours             float64 `json:"loggedHours"`
	BillableMinutes         int64   `json:"billableMinutes"`
	BillableHours           float64 `json:"billableHours"`
	NonBillableMinutes      int64   `json:"nonBillableMinutes"`
	NonBillableHours        float64 `json:"nonBillableHours"`
	BilledMinutes           int64   `json:"billedMinutes"`
	BilledHours             float64 `json:"billedHours"`
	UnbilledBillableMinutes int64   `json:"unbilledBillableMinutes"`
	UnbilledBillableHours   float64 `json:"unbilledBillableHours"`
}

// timelogSummaryScope echoes back the query that produced the report.
type timelogSummaryScope struct {
	GroupBy   string `json:"groupBy"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// timelogSummaryTotals is the roll-up across every group.
type timelogSummaryTotals struct {
	timelogSummaryColumns
	GroupCount int64 `json:"groupCount"`
}

// timelogSummaryGroup is one grouped row (a user or a project).
type timelogSummaryGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	timelogSummaryColumns
}

// timelogSummaryResult is the full tool response.
type timelogSummaryResult struct {
	Scope  timelogSummaryScope   `json:"scope"`
	Totals timelogSummaryTotals  `json:"totals"`
	Groups []timelogSummaryGroup `json:"groups"`
}

// timelogSummaryAccumulator sums the raw minute columns of the underlying time
// report before they are projected into a timelogSummaryColumns.
type timelogSummaryAccumulator struct {
	logged      int64
	billable    int64
	nonBillable int64
	billed      int64
}

func (a *timelogSummaryAccumulator) add(c projects.TimeReportColumns) {
	a.logged += c.LoggedTime
	a.billable += c.BillableTime
	a.nonBillable += c.NonBillableTime
	a.billed += c.BilledTime
}

// columns projects the accumulated minutes into the published column set,
// deriving hours and the unbilled-billable difference. unbilledBillable is
// billable − billed by definition.
func (a timelogSummaryAccumulator) columns() timelogSummaryColumns {
	unbilledBillable := a.billable - a.billed
	return timelogSummaryColumns{
		LoggedMinutes:           a.logged,
		LoggedHours:             minutesToHours(a.logged),
		BillableMinutes:         a.billable,
		BillableHours:           minutesToHours(a.billable),
		NonBillableMinutes:      a.nonBillable,
		NonBillableHours:        minutesToHours(a.nonBillable),
		BilledMinutes:           a.billed,
		BilledHours:             minutesToHours(a.billed),
		UnbilledBillableMinutes: unbilledBillable,
		UnbilledBillableHours:   minutesToHours(unbilledBillable),
	}
}

// minutesToHours converts exact minutes to hours rounded to two decimals.
func minutesToHours(minutes int64) float64 {
	return math.Round(float64(minutes)/60*100) / 100
}

var timelogSummaryOutputSchema *jsonschema.Schema

func init() {
	var err error

	// The output schema is intentionally strict (every field required,
	// additionalProperties false, no WithOptionalFields relaxation) so it stays
	// OpenAI-strict compatible: the response always carries the full column set.
	timelogSummaryOutputSchema, err = jsonschema.For[timelogSummaryResult](&jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for timelogSummaryResult: %v", err))
	}
}

// timelogSummaryIDListSchema returns the schema for an optional integer-ID list
// filter, wrapped in AnyOf with null so callers may omit it.
func timelogSummaryIDListSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: description,
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			{Type: "null"},
		},
	}
}

// SummarizeTimelogs returns deterministic, complete time aggregates grouped by
// user or project for a required date window, paginating the underlying time
// report internally so a single call yields every group.
func SummarizeTimelogs(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSummarizeTimelogs),
			Description: "Deterministic, complete time-tracking totals for a date window, grouped by user or " +
				"project. Returns every group in one call with exact minute sums and 2-decimal hours — no " +
				"pagination for the caller, any model tier. Use this instead of twprojects-list_timelogs whenever " +
				"the question is about totals, sums, or breakdowns (e.g. \"how many hours did X log\", \"time per " +
				"project this month\", billable vs billed vs unbilled); use list_timelogs only when you need the " +
				"individual timelog entries. Minutes are exact and authoritative; hours are minutes ÷ 60 rounded " +
				"to 2 decimals. unbilledBillable = billable − billed. The sum of the group columns equals the " +
				"totals block exactly (reconcile in minutes, not hours).",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Summarize Timelogs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"start_date": {
						Type:        "string",
						Format:      "date",
						Description: "Inclusive start of the report window (YYYY-MM-DD).",
					},
					"end_date": {
						Type:        "string",
						Format:      "date",
						Description: "Inclusive end of the report window (YYYY-MM-DD).",
					},
					"group_by": {
						Type:        "string",
						Enum:        []any{"user", "project"},
						Default:     []byte(`"user"`),
						Description: "Dimension to group totals by. Defaults to user.",
					},
					"project_ids":     timelogSummaryIDListSchema("Filter to timelogs on these projects."),
					"user_ids":        timelogSummaryIDListSchema("Filter to timelogs logged for these users."),
					"task_ids":        timelogSummaryIDListSchema("Filter to timelogs on these tasks."),
					"tasklist_ids":    timelogSummaryIDListSchema("Filter to timelogs on these task lists."),
					"company_ids":     timelogSummaryIDListSchema("Filter to timelogs on projects of these companies/clients."),
					"team_ids":        timelogSummaryIDListSchema("Filter to timelogs logged by members of these teams."),
					"timelog_tag_ids": timelogSummaryIDListSchema("Filter to timelogs carrying these tags."),
					"include_archived_projects": {
						Description: "Include time from archived projects. Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte(`false`),
					},
				},
				Required: []string{"start_date", "end_date"},
			},
			OutputSchema: timelogSummaryOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var (
				startDate       twapi.Date
				endDate         twapi.Date
				groupBy         = "user"
				projectIDs      []int64
				userIDs         []int64
				taskIDs         []int64
				tasklistIDs     []int64
				companyIDs      []int64
				teamIDs         []int64
				timelogTagIDs   []int64
				includeArchived bool
			)

			err := helpers.ParamGroup(arguments,
				helpers.RequiredDateParam(&startDate, "start_date"),
				helpers.RequiredDateParam(&endDate, "end_date"),
				helpers.OptionalParam(&groupBy, "group_by", helpers.RestrictValues("user", "project")),
				helpers.OptionalNumericListParam(&projectIDs, "project_ids"),
				helpers.OptionalNumericListParam(&userIDs, "user_ids"),
				helpers.OptionalNumericListParam(&taskIDs, "task_ids"),
				helpers.OptionalNumericListParam(&tasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&companyIDs, "company_ids"),
				helpers.OptionalNumericListParam(&teamIDs, "team_ids"),
				helpers.OptionalNumericListParam(&timelogTagIDs, "timelog_tag_ids"),
				helpers.OptionalParam(&includeArchived, "include_archived_projects"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Reject a reversed window. An empty window (no logs in range) is a
			// valid query that returns zeros, so only start > end is an error.
			if time.Time(startDate).After(time.Time(endDate)) {
				return helpers.NewToolResultTextError(
					"invalid parameters: start_date (%s) must be on or before end_date (%s)",
					startDate.String(), endDate.String(),
				), nil
			}

			// Map group_by to the report's grouping dimension, the precanned
			// reportType variant, and the sideload used to resolve group names.
			var (
				dimension  projects.TimeReportType
				reportType projects.TimeReportReportType
				sideload   projects.TimeReportSideload
			)
			switch groupBy {
			case "project":
				dimension = projects.TimeReportTypeProject
				reportType = projects.TimeReportReportTypeProjectLoggedTime
				sideload = projects.TimeReportSideloadProjects
			default: // "user"
				dimension = projects.TimeReportTypeUser
				reportType = projects.TimeReportReportTypeUserLoggedTime
				sideload = projects.TimeReportSideloadUsers
			}

			timeReportRequest := projects.NewTimeReportListRequest(dimension, startDate, endDate)
			timeReportRequest.Filters.ReportType = reportType
			timeReportRequest.Filters.ProjectIDs = projectIDs
			timeReportRequest.Filters.UserIDs = userIDs
			timeReportRequest.Filters.TaskIDs = taskIDs
			timeReportRequest.Filters.TasklistIDs = tasklistIDs
			timeReportRequest.Filters.CompanyIDs = companyIDs
			timeReportRequest.Filters.TeamIDs = teamIDs
			timeReportRequest.Filters.TimelogTagIDs = timelogTagIDs
			timeReportRequest.Filters.IncludeArchivedProjects = &includeArchived
			timeReportRequest.Filters.Include = []projects.TimeReportSideload{sideload}
			timeReportRequest.Filters.Page = 1
			timeReportRequest.Filters.PageSize = timelogSummaryPageSize
			// Only the fields needed to join group names are requested.
			timeReportRequest.Filters.Fields.Users = []projects.UserField{
				projects.UserFieldID, projects.UserFieldFirstName, projects.UserFieldLastName,
			}
			timeReportRequest.Filters.Fields.Projects = []projects.ProjectField{
				projects.ProjectFieldID, projects.ProjectFieldName,
			}

			// The time report API silently scopes rows to what the caller is
			// permitted to see: time on projects the caller cannot access is
			// omitted server-side rather than raising an error. This is accepted
			// with no runtime mitigation (decided) — totals reflect the caller's
			// own visibility.

			// Accumulate rows across pages. The report is grouped server-side, so
			// a group id normally appears once, but rows are still folded into a
			// keyed map (preserving first-seen order) to stay correct even if a
			// group were ever split across pages.
			order := make([]int64, 0)
			byID := make(map[int64]*timelogSummaryAccumulator)
			names := make(map[int64]string)

			accumulate := func(id int64, name string, cols projects.TimeReportColumns) {
				acc, ok := byID[id]
				if !ok {
					acc = &timelogSummaryAccumulator{}
					byID[id] = acc
					order = append(order, id)
				}
				acc.add(cols)
				if name != "" {
					names[id] = name
				}
			}

			page := 0
			for {
				response, err := projects.TimeReportList(ctx, engine, timeReportRequest)
				if err != nil {
					return helpers.HandleAPIError(err, "failed to summarize timelogs")
				}
				page++

				switch groupBy {
				case "project":
					for _, row := range response.TimeReport.Projects {
						id := row.Project.ID
						var name string
						if p, ok := response.Included.Projects[strconv.FormatInt(id, 10)]; ok {
							name = strings.TrimSpace(p.Name)
						}
						accumulate(id, name, row.TimeReportColumns)
					}
				default: // "user"
					for _, row := range response.TimeReport.Users {
						id := row.User.ID
						var name string
						if u, ok := response.Included.Users[strconv.FormatInt(id, 10)]; ok {
							name = strings.TrimSpace(u.FirstName + " " + u.LastName)
						}
						accumulate(id, name, row.TimeReportColumns)
					}
				}

				next := response.Iterate()
				if next == nil {
					break
				}
				if page >= timelogSummaryMaxPages {
					return helpers.NewToolResultTextError(
						"time report exceeded the %d-page limit (page size %d) and would return partial totals; "+
							"narrow the date window or add filters (e.g. project_ids, user_ids, team_ids) and try again",
						timelogSummaryMaxPages, timelogSummaryPageSize,
					), nil
				}
				timeReportRequest = *next
			}

			// Build grouped rows and totals from the same minute sums, so the
			// group columns reconcile against the totals block exactly in minutes.
			var totals timelogSummaryAccumulator
			groups := make([]timelogSummaryGroup, 0, len(order))
			for _, id := range order {
				acc := byID[id]
				name := names[id]
				if name == "" {
					// Sideload entry missing (or blank): fall back to a synthetic
					// name so the row is never dropped.
					name = fmt.Sprintf("%s %d", groupBy, id)
				}
				groups = append(groups, timelogSummaryGroup{
					ID:                    id,
					Name:                  name,
					timelogSummaryColumns: acc.columns(),
				})
				totals.logged += acc.logged
				totals.billable += acc.billable
				totals.nonBillable += acc.nonBillable
				totals.billed += acc.billed
			}

			result := timelogSummaryResult{
				Scope: timelogSummaryScope{
					GroupBy:   groupBy,
					StartDate: startDate.String(),
					EndDate:   endDate.String(),
				},
				Totals: timelogSummaryTotals{
					timelogSummaryColumns: totals.columns(),
					GroupCount:            int64(len(groups)),
				},
				Groups: groups,
			}
			return helpers.NewToolResultJSON(result)
		},
	}
}
