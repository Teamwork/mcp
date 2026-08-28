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

// timelogSummaryGroupBys lists the group_by values in published order.
var timelogSummaryGroupBys = []string{"user", "project", "task", "day", "week", "month"}

// timelogSummaryPeriods maps period group_by values onto the totals endpoint's
// buckets. A group_by absent from it is an entity dimension.
var timelogSummaryPeriods = map[string]projects.TimeReportGroupBy{
	"day":   projects.TimeReportGroupByDay,
	"week":  projects.TimeReportGroupByWeek,
	"month": projects.TimeReportGroupByMonth,
}

// timelogSummaryColumns holds the ten aggregate columns shared by the totals
// block and every row. Minutes are authoritative; hours are minutes ÷ 60
// rounded to two decimals.
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

// timelogSummaryTotals is the roll-up across every row.
type timelogSummaryTotals struct {
	timelogSummaryColumns
	GroupCount int64 `json:"groupCount"`
}

// timelogSummaryGroup is one entity row (a user, a project or a task).
type timelogSummaryGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	timelogSummaryColumns
}

// timelogSummaryPeriod is one period row (a day, a week or a month). Both
// dates are inclusive; a day has StartDate equal to EndDate.
type timelogSummaryPeriod struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	timelogSummaryColumns
}

// timelogSummaryResult is the full tool response. Exactly one of Groups and
// Periods carries rows, matching the group_by dimension; the other is empty.
type timelogSummaryResult struct {
	Scope   timelogSummaryScope    `json:"scope"`
	Totals  timelogSummaryTotals   `json:"totals"`
	Groups  []timelogSummaryGroup  `json:"groups"`
	Periods []timelogSummaryPeriod `json:"periods"`
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

func (a *timelogSummaryAccumulator) merge(other timelogSummaryAccumulator) {
	a.logged += other.logged
	a.billable += other.billable
	a.nonBillable += other.nonBillable
	a.billed += other.billed
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

// timelogSummaryFilters carries the caller's filters, which both endpoints
// take under the same names.
type timelogSummaryFilters struct {
	projectIDs      []int64
	userIDs         []int64
	taskIDs         []int64
	tasklistIDs     []int64
	companyIDs      []int64
	teamIDs         []int64
	timelogTagIDs   []int64
	includeArchived bool
}

var timelogSummaryOutputSchema *jsonschema.Schema

// timeReportOrdering is the order-by vocabulary of the time report endpoint
// behind twprojects-summarize_timelogs.
var timeReportOrdering = newOrdering("summary rows",
	projects.TimeReportOrderByName,
	projects.TimeReportOrderByLoggedTime,
	projects.TimeReportOrderByBillableTime,
	projects.TimeReportOrderByNonBillableTime,
	projects.TimeReportOrderByBilledTime,
	projects.TimeReportOrderByBudget,
)

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

	properties := timelogSummaryOutputSchema.Properties
	properties["groups"].Description = "Entity rows; empty for a day, week or month grouping."
	properties["periods"].Description = "Period rows, chronological; empty for an entity grouping."
	properties["totals"].Properties["groupCount"].Description = "Number of rows returned."
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

// timelogSummaryOrderingSchema notes the period exclusion on the parameter
// itself, since a model reading one parameter never sees another's text.
func timelogSummaryOrderingSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	schema.Description += " Rejected for day, week and month: period rows are chronological."
	return schema
}

// SummarizeTimelogs returns complete time aggregates for a date window: entity
// rows from the grouped time report, period rows from the report totals.
func SummarizeTimelogs(engine *twapi.Engine) toolsets.ToolWrapper {
	groupByEnum := make([]any, len(timelogSummaryGroupBys))
	for i, value := range timelogSummaryGroupBys {
		groupByEnum[i] = value
	}

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSummarizeTimelogs),
			Description: "Complete time totals for a date window, grouped by user, project or task (rows in " +
				"groups) or by day, week or month (rows in periods). One call returns every row, in exact " +
				"minutes and hours to 2 decimals. Prefer it over twprojects-list_timelogs for any total, sum " +
				"or breakdown; list_timelogs is for individual entries. Minutes are authoritative, " +
				"unbilledBillable = billable − billed, and rows sum to totals. One dimension per call: for " +
				"hours per user per week, call once per user with user_ids. Task rows omit project-level and " +
				"subtask time, so never add totals across group_by values. Period rows cover every period in " +
				"order, zeros included, first and last clipped to the window; weeks follow the caller's " +
				"start-of-week setting, so buckets can differ per user, and a weekend-only week with no time " +
				"is dropped.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Summarize Timelogs",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Type:    "string",
						Enum:    groupByEnum,
						Default: []byte(`"user"`),
						Description: "Dimension to group by. Defaults to user. user, project and task fill " +
							"groups; day, week and month fill periods. Filter a task grouping on a busy account.",
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
					// Ordering reaches the report request. The handler errors
					// rather than truncating, so it changes the order of the
					// rows, never which rows come back.
					"order_by":   timelogSummaryOrderingSchema(timeReportOrdering.orderBySchema()),
					"order_mode": timelogSummaryOrderingSchema(orderModeSchema()),
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
				startDate twapi.Date
				endDate   twapi.Date
				groupBy   = "user"
				filters   timelogSummaryFilters
				orderBy   projects.TimeReportOrderBy
				orderMode twapi.OrderMode
			)

			err := helpers.ParamGroup(arguments,
				helpers.RequiredDateParam(&startDate, "start_date"),
				helpers.RequiredDateParam(&endDate, "end_date"),
				helpers.OptionalParam(&groupBy, "group_by", helpers.RestrictValues(timelogSummaryGroupBys...)),
				helpers.OptionalNumericListParam(&filters.projectIDs, "project_ids"),
				helpers.OptionalNumericListParam(&filters.userIDs, "user_ids"),
				helpers.OptionalNumericListParam(&filters.taskIDs, "task_ids"),
				helpers.OptionalNumericListParam(&filters.tasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&filters.companyIDs, "company_ids"),
				helpers.OptionalNumericListParam(&filters.teamIDs, "team_ids"),
				helpers.OptionalNumericListParam(&filters.timelogTagIDs, "timelog_tag_ids"),
				timeReportOrdering.param(&orderBy, &orderMode),
				helpers.OptionalParam(&filters.includeArchived, "include_archived_projects"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// An empty window is valid and returns zeros, so only start > end fails.
			if time.Time(startDate).After(time.Time(endDate)) {
				return helpers.NewToolResultTextError(
					"invalid parameters: start_date (%s) must be on or before end_date (%s)",
					startDate.String(), endDate.String(),
				), nil
			}

			// The API silently omits time on projects the caller cannot see, so
			// totals reflect the caller's own visibility. Accepted (decided).

			if period, ok := timelogSummaryPeriods[groupBy]; ok {
				// Period rows are chronological; an ordering would do nothing.
				if orderBy != "" || orderMode != "" {
					return helpers.NewToolResultTextError(
						"invalid parameters: order_by and order_mode are not accepted when group_by is %s",
						groupBy,
					), nil
				}
				return summarizeTimelogsByPeriod(ctx, engine, groupBy, period, startDate, endDate, filters)
			}
			return summarizeTimelogsByEntity(ctx, engine, groupBy, startDate, endDate, filters, orderBy, orderMode)
		},
	}
}

// summarizeTimelogsByPeriod answers a period group_by through the report
// totals. The endpoint keys buckets by day of year or month number with no
// year, so the window goes one calendar year per request.
func summarizeTimelogsByPeriod(
	ctx context.Context,
	engine *twapi.Engine,
	groupBy string,
	period projects.TimeReportGroupBy,
	startDate, endDate twapi.Date,
	filters timelogSummaryFilters,
) (*mcp.CallToolResult, error) {
	var rows []timelogSummaryPeriodRow
	for _, window := range calendarYearWindows(startDate, endDate) {
		totalsRequest := projects.NewTimeReportTotalsRequest(period, window.start, window.end)
		totalsRequest.Filters.ProjectIDs = filters.projectIDs
		totalsRequest.Filters.UserIDs = filters.userIDs
		totalsRequest.Filters.TaskIDs = filters.taskIDs
		totalsRequest.Filters.TasklistIDs = filters.tasklistIDs
		totalsRequest.Filters.CompanyIDs = filters.companyIDs
		totalsRequest.Filters.TeamIDs = filters.teamIDs
		totalsRequest.Filters.TimelogTagIDs = filters.timelogTagIDs
		totalsRequest.Filters.IncludeArchivedProjects = &filters.includeArchived

		response, err := projects.TimeReportTotals(ctx, engine, totalsRequest)
		if err != nil {
			return helpers.HandleAPIError(err, "failed to summarize timelogs")
		}
		for _, entry := range response.Dates {
			row := timelogSummaryPeriodRow{start: time.Time(entry.StartDate), end: time.Time(entry.EndDate)}
			row.acc.add(entry.TimeReportColumns)
			rows = appendPeriodRow(rows, period, row)
		}
	}

	// Summed from the rows, not read off the responses, so they reconcile.
	var totals timelogSummaryAccumulator
	periods := make([]timelogSummaryPeriod, 0, len(rows))
	for _, row := range rows {
		totals.merge(row.acc)
		periods = append(periods, timelogSummaryPeriod{
			StartDate:             twapi.Date(row.start).String(),
			EndDate:               twapi.Date(row.end).String(),
			timelogSummaryColumns: row.acc.columns(),
		})
	}

	return helpers.NewToolResultJSON(timelogSummaryResult{
		Scope: timelogSummaryScope{
			GroupBy:   groupBy,
			StartDate: startDate.String(),
			EndDate:   endDate.String(),
		},
		Totals: timelogSummaryTotals{
			timelogSummaryColumns: totals.columns(),
			GroupCount:            int64(len(periods)),
		},
		Groups:  []timelogSummaryGroup{},
		Periods: periods,
	})
}

// timelogSummaryPeriodRow is a period row before projection, kept as dates and
// raw minutes so a row split across requests can be merged.
type timelogSummaryPeriodRow struct {
	start time.Time
	end   time.Time
	acc   timelogSummaryAccumulator
}

// dateWindow is an inclusive date range.
type dateWindow struct {
	start twapi.Date
	end   twapi.Date
}

// calendarYearWindows splits an inclusive window at every 1 January.
func calendarYearWindows(startDate, endDate twapi.Date) []dateWindow {
	start, end := time.Time(startDate), time.Time(endDate)
	var windows []dateWindow
	for year := start.Year(); year <= end.Year(); year++ {
		window := dateWindow{start: startDate, end: endDate}
		if year > start.Year() {
			window.start = twapi.Date(time.Date(year, time.January, 1, 0, 0, 0, 0, start.Location()))
		}
		if year < end.Year() {
			window.end = twapi.Date(time.Date(year, time.December, 31, 0, 0, 0, 0, end.Location()))
		}
		windows = append(windows, window)
	}
	return windows
}

// appendPeriodRow appends row, merging it into the previous one when both are
// halves of a week the 1 January split cut in two: previous ends 31 December,
// this one starts 1 January, and together they span under seven days — so a
// full week ending 31 December never merges. Days and months never straddle it.
func appendPeriodRow(
	rows []timelogSummaryPeriodRow,
	period projects.TimeReportGroupBy,
	row timelogSummaryPeriodRow,
) []timelogSummaryPeriodRow {
	if period == projects.TimeReportGroupByWeek && len(rows) > 0 {
		last := &rows[len(rows)-1]
		endsYear := last.end.Month() == time.December && last.end.Day() == 31
		continues := row.start.Year() == last.end.Year()+1 && row.start.YearDay() == 1
		if endsYear && continues && row.end.Sub(last.start) < 7*24*time.Hour {
			last.end = row.end
			last.acc.merge(row.acc)
			return rows
		}
	}
	return append(rows, row)
}

// summarizeTimelogsByEntity answers an entity group_by, paginating the grouped
// time report internally.
func summarizeTimelogsByEntity(
	ctx context.Context,
	engine *twapi.Engine,
	groupBy string,
	startDate, endDate twapi.Date,
	filters timelogSummaryFilters,
	orderBy projects.TimeReportOrderBy,
	orderMode twapi.OrderMode,
) (*mcp.CallToolResult, error) {
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
	case "task":
		dimension = projects.TimeReportTypeTask
		// The logged-time variant drops tasks carrying only an
		// estimate; otherwise they arrive as rows of zeros.
		reportType = projects.TimeReportReportTypeLoggedTime
		sideload = projects.TimeReportSideloadTasks
	default: // "user"
		dimension = projects.TimeReportTypeUser
		reportType = projects.TimeReportReportTypeUserLoggedTime
		sideload = projects.TimeReportSideloadUsers
	}

	timeReportRequest := projects.NewTimeReportListRequest(dimension, startDate, endDate)
	timeReportRequest.Filters.ReportType = reportType
	timeReportRequest.Filters.ProjectIDs = filters.projectIDs
	timeReportRequest.Filters.UserIDs = filters.userIDs
	timeReportRequest.Filters.TaskIDs = filters.taskIDs
	timeReportRequest.Filters.TasklistIDs = filters.tasklistIDs
	timeReportRequest.Filters.CompanyIDs = filters.companyIDs
	timeReportRequest.Filters.TeamIDs = filters.teamIDs
	timeReportRequest.Filters.TimelogTagIDs = filters.timelogTagIDs
	timeReportRequest.Filters.IncludeArchivedProjects = &filters.includeArchived
	timeReportRequest.Filters.Include = []projects.TimeReportSideload{sideload}
	timeReportRequest.Filters.OrderBy = orderBy
	timeReportRequest.Filters.OrderMode = orderMode
	timeReportRequest.Filters.Page = 1
	timeReportRequest.Filters.PageSize = timelogSummaryPageSize
	// Only the fields needed to join group names are requested.
	timeReportRequest.Filters.Fields.Users = []projects.UserField{
		projects.UserFieldID, projects.UserFieldFirstName, projects.UserFieldLastName,
	}
	timeReportRequest.Filters.Fields.Projects = []projects.ProjectField{
		projects.ProjectFieldID, projects.ProjectFieldName,
	}
	timeReportRequest.Filters.Fields.Tasks = []projects.TaskField{
		projects.TaskFieldID, projects.TaskFieldName,
	}

	// Folded into a keyed map in first-seen order: the report groups
	// server-side, but this stays correct if a group ever spans two pages.
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
		case "task":
			for _, row := range response.TimeReport.Tasks {
				id := row.Task.ID
				var name string
				if task, ok := response.Included.Tasks[strconv.FormatInt(id, 10)]; ok {
					name = strings.TrimSpace(task.Name)
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

	// Rows and totals come from the same minute sums, so they reconcile.
	var totals timelogSummaryAccumulator
	groups := make([]timelogSummaryGroup, 0, len(order))
	for _, id := range order {
		acc := byID[id]
		name := names[id]
		if name == "" {
			// Sideload missing or blank: name it rather than drop the row.
			name = fmt.Sprintf("%s %d", groupBy, id)
		}
		groups = append(groups, timelogSummaryGroup{
			ID:                    id,
			Name:                  name,
			timelogSummaryColumns: acc.columns(),
		})
		totals.merge(*acc)
	}

	return helpers.NewToolResultJSON(timelogSummaryResult{
		Scope: timelogSummaryScope{
			GroupBy:   groupBy,
			StartDate: startDate.String(),
			EndDate:   endDate.String(),
		},
		Totals: timelogSummaryTotals{
			timelogSummaryColumns: totals.columns(),
			GroupCount:            int64(len(groups)),
		},
		Groups:  groups,
		Periods: []timelogSummaryPeriod{},
	})
}
