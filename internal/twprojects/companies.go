package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCompanyCreate      toolsets.Method = "twprojects-create_company"
	MethodCompanyCreateBatch toolsets.Method = "twprojects-create_companies"
	MethodCompanyUpdate      toolsets.Method = "twprojects-update_company"
	MethodCompanyDelete      toolsets.Method = "twprojects-delete_company"
	MethodCompanyGet         toolsets.Method = "twprojects-get_company"
	MethodCompanyList        toolsets.Method = "twprojects-list_companies"
)

var (
	companyGetOutputSchema  *jsonschema.Schema
	companyListOutputSchema *jsonschema.Schema
)

// companyOrdering is the order-by vocabulary of the companies list endpoint.
var companyOrdering = newOrdering("companies",
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

func init() {
	var err error

	// generate the output schemas only once
	companyGetOutputSchema, err = jsonschema.For[projects.CompanyGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CompanyGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(companyGetOutputSchema)
	companyListOutputSchema, err = jsonschema.For[projects.CompanyListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CompanyListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(companyListOutputSchema)
}

// CompanyCreate creates a company in Teamwork.com.
func CompanyCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyCreate),
			Description: "Create company (aka client).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Company",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the company.",
					},
					"address_one": {
						Description: "The first line of the address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"address_two": {
						Description: "The second line of the address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"city": {
						Description: "The city of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"state": {
						Description: "The state of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"zip": {
						Description: "The ZIP or postal code of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"country_code": {
						Description: "The country code of the company, e.g., 'US' for the United States.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"phone": {
						Description: "The phone number of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"fax": {
						Description: "The fax number of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_one": {
						Description: "The primary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_two": {
						Description: "The secondary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_three": {
						Description: "The tertiary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"website": {
						Description: "The website of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"profile": {
						Description: "A profile description for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"manager_id": {
						Description: "The ID of the user who manages the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"industry_id": {
						Description: "The ID of the industry the company belongs to.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("company"),
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			companyCreateRequest, err := parseCompanyCreateArguments(arguments)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			companyResponse, err := projects.CompanyCreate(ctx, engine, companyCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create company")
			}
			return helpers.NewToolResultText("Company created successfully with ID %d", companyResponse.Company.ID), nil
		},
	}
}

// parseCompanyCreateArguments binds the parameters of twprojects-create_company,
// which twprojects-create_companies takes for each of its items.
func parseCompanyCreateArguments(arguments map[string]any) (projects.CompanyCreateRequest, error) {
	var companyCreateRequest projects.CompanyCreateRequest
	err := helpers.ParamGroup(arguments,
		helpers.RequiredParam(&companyCreateRequest.Name, "name"),
		helpers.OptionalPointerParam(&companyCreateRequest.AddressOne, "address_one"),
		helpers.OptionalPointerParam(&companyCreateRequest.AddressTwo, "address_two"),
		helpers.OptionalPointerParam(&companyCreateRequest.City, "city"),
		helpers.OptionalPointerParam(&companyCreateRequest.State, "state"),
		helpers.OptionalPointerParam(&companyCreateRequest.Zip, "zip"),
		helpers.OptionalPointerParam(&companyCreateRequest.CountryCode, "country_code"),
		helpers.OptionalPointerParam(&companyCreateRequest.Phone, "phone"),
		helpers.OptionalPointerParam(&companyCreateRequest.Fax, "fax"),
		helpers.OptionalPointerParam(&companyCreateRequest.EmailOne, "email_one"),
		helpers.OptionalPointerParam(&companyCreateRequest.EmailTwo, "email_two"),
		helpers.OptionalPointerParam(&companyCreateRequest.EmailThree, "email_three"),
		helpers.OptionalPointerParam(&companyCreateRequest.Website, "website"),
		helpers.OptionalPointerParam(&companyCreateRequest.Profile, "profile"),
		helpers.OptionalNumericPointerParam(&companyCreateRequest.ManagerID, "manager_id"),
		helpers.OptionalNumericPointerParam(&companyCreateRequest.IndustryID, "industry_id"),
		helpers.OptionalNumericListParam(&companyCreateRequest.TagIDs, "tag_ids"),
	)
	return companyCreateRequest, err
}

// companyCreateBatchMaxItems bounds twprojects-create_companies. The whole list
// is one request, and the endpoint creates the companies one after another, so
// a longer list would outlive the tool call.
const companyCreateBatchMaxItems = 50

// CompanyCreateBatch creates many companies in Teamwork.com with a single
// request to the bulk create route, with partial success turned on so one bad
// company does not stop the others.
func CompanyCreateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(CompanyCreate(engine).Tool.InputSchema)
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyCreateBatch),
			Description: fmt.Sprintf("Create many companies (aka clients) in one call, up to %d per call. "+
				"Use instead of calling twprojects-create_company in a loop. Items are validated first and "+
				"nothing is created if one is invalid; after that each company succeeds or fails on its own, "+
				"and the result lists the company ID of each item and the reason for each failure, numbering "+
				"items from 1 in list order. A name already used by an existing company (ignoring case and "+
				"surrounding spaces) is not created again: the item reports that company's ID instead, and "+
				"the same applies to a name repeated in the list.", companyCreateBatchMaxItems),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Companies",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"companies": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(companyCreateBatchMaxItems),
						Description: fmt.Sprintf("The companies to create, at most %d, each with the parameters "+
							"of twprojects-create_company.", companyCreateBatchMaxItems),
					},
				},
				Required: []string{"companies"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "companies", companyCreateBatchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			bulkRequest := projects.CompanyBulkCreateRequest{
				Companies:      make([]projects.CompanyCreateRequest, len(items)),
				PartialSuccess: true,
			}
			labels := make([]string, len(items))
			var invalid []string
			for i, item := range items {
				req, err := parseCompanyCreateArguments(item)
				labels[i] = fmt.Sprintf("item %d %q", i+1, req.Name)
				if err != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, err.Error()))
					continue
				}
				bulkRequest.Companies[i] = req
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			response, err := projects.CompanyBulkCreate(ctx, engine, bulkRequest)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return helpers.NewToolResultTextError("The request was interrupted and its outcome is "+
						"unknown; check with %s before sending the companies again.", MethodCompanyList), nil
				}
				if failures := companyBulkCreateErrorFailures(err); failures != nil {
					return companyBatchReport(labels, nil, nil, failures), nil
				}
				return helpers.HandleAPIError(err, "failed to create companies")
			}
			indexes, err := response.CompanyIndexes(len(items))
			if err != nil {
				return helpers.NewToolResultTextError("The companies were sent but the response could not be "+
					"read (%s); check with %s before sending them again.", err.Error(), MethodCompanyList), nil
			}
			return companyBatchReport(labels, indexes, response.Companies, response.Failed), nil
		},
	}
}

// companyBulkCreateErrorFailures reads the per-company errors of a bulk create
// in which no company could be created: the endpoint then answers 400 or 403
// with each error carrying its company's position under meta.index. It returns
// nil for any other error.
func companyBulkCreateErrorFailures(err error) []projects.CompanyBulkCreateFailure {
	httpErr, ok := errors.AsType[*twapi.HTTPError](err)
	if !ok || (httpErr.StatusCode != http.StatusBadRequest && httpErr.StatusCode != http.StatusForbidden) {
		return nil
	}
	var body struct {
		Errors []twapi.V3Error `json:"errors"`
	}
	if json.Unmarshal([]byte(httpErr.Details), &body) != nil || len(body.Errors) == 0 {
		return nil
	}
	byIndex := make(map[int][]twapi.V3Error)
	for _, entry := range body.Errors {
		index, ok := entry.Meta["index"].(float64)
		if !ok {
			return nil
		}
		byIndex[int(index)] = append(byIndex[int(index)], entry)
	}
	failures := make([]projects.CompanyBulkCreateFailure, 0, len(byIndex))
	for _, index := range slices.Sorted(maps.Keys(byIndex)) {
		failures = append(failures, projects.CompanyBulkCreateFailure{Index: index, Errors: byIndex[index]})
	}
	return failures
}

// companyBatchReport summarises a bulk create in one result: the company ID of
// each item that succeeded and the reasons of each item that failed. It is an
// error result when any item failed, even though the others were created.
func companyBatchReport(
	labels []string,
	indexes []int,
	companies []projects.Company,
	failures []projects.CompanyBulkCreateFailure,
) *mcp.CallToolResult {
	var report strings.Builder
	fmt.Fprintf(&report, "Created %d of %d companies.", len(companies), len(labels))
	if len(companies) > 0 {
		created := make([]string, len(companies))
		for i, company := range companies {
			created[i] = fmt.Sprintf("%s → company %d", labels[indexes[i]], company.ID)
		}
		fmt.Fprintf(&report, "\nCreated: %s.", strings.Join(created, ", "))
	}
	for _, failure := range failures {
		label := fmt.Sprintf("item %d", failure.Index+1)
		if failure.Index >= 0 && failure.Index < len(labels) {
			label = labels[failure.Index]
		}
		reasons := make([]string, len(failure.Errors))
		for i, reason := range failure.Errors {
			reasons[i] = strings.TrimPrefix(reason.Title+": "+reason.Detail, ": ")
		}
		fmt.Fprintf(&report, "\nFailed: %s: %s.", label, strings.Join(reasons, "; "))
	}
	if len(companies) < len(labels) {
		return helpers.NewToolResultTextError("%s", report.String())
	}
	return helpers.NewToolResultText("%s", report.String())
}

// CompanyUpdate updates a company in Teamwork.com.
func CompanyUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyUpdate),
			Description: "Update company (aka client).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Company",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to update.",
					},
					"name": {
						Description: "The name of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"address_one": {
						Description: "The first line of the address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"address_two": {
						Description: "The second line of the address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"city": {
						Description: "The city of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"state": {
						Description: "The state of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"zip": {
						Description: "The ZIP or postal code of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"country_code": {
						Description: "The country code of the company, e.g., 'US' for the United States.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"phone": {
						Description: "The phone number of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"fax": {
						Description: "The fax number of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_one": {
						Description: "The primary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_two": {
						Description: "The secondary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email_three": {
						Description: "The tertiary email address of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"website": {
						Description: "The website of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"profile": {
						Description: "A profile description for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"manager_id": {
						Description: "The ID of the user who manages the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"industry_id": {
						Description: "The ID of the industry the company belongs to.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("company"),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyUpdateRequest projects.CompanyUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&companyUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&companyUpdateRequest.AddressOne, "address_one"),
				helpers.OptionalPointerParam(&companyUpdateRequest.AddressTwo, "address_two"),
				helpers.OptionalPointerParam(&companyUpdateRequest.City, "city"),
				helpers.OptionalPointerParam(&companyUpdateRequest.State, "state"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Zip, "zip"),
				helpers.OptionalPointerParam(&companyUpdateRequest.CountryCode, "country_code"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Phone, "phone"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Fax, "fax"),
				helpers.OptionalPointerParam(&companyUpdateRequest.EmailOne, "email_one"),
				helpers.OptionalPointerParam(&companyUpdateRequest.EmailTwo, "email_two"),
				helpers.OptionalPointerParam(&companyUpdateRequest.EmailThree, "email_three"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Website, "website"),
				helpers.OptionalPointerParam(&companyUpdateRequest.Profile, "profile"),
				helpers.OptionalNumericPointerParam(&companyUpdateRequest.ManagerID, "manager_id"),
				helpers.OptionalNumericPointerParam(&companyUpdateRequest.IndustryID, "industry_id"),
				helpers.OptionalNumericListParam(&companyUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CompanyUpdate(ctx, engine, companyUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update company")
			}
			return helpers.NewToolResultText("Company updated successfully"), nil
		},
	}
}

// CompanyDelete deletes a company in Teamwork.com.
func CompanyDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyDelete),
			Description: "Delete company (aka client).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Company",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyDeleteRequest projects.CompanyDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&companyDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CompanyDelete(ctx, engine, companyDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete company")
			}
			return helpers.NewToolResultText("Company deleted successfully"), nil
		},
	}
}

// CompanyGet retrieves a company in Teamwork.com.
func CompanyGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyGet),
			Description: "Get company (aka client).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Company",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to get.",
					},
					"fields": helpers.FieldsSchema[projects.Company]("company"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(companyGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyGetRequest projects.CompanyGetRequest

			// Always include custom fields and values in the response to provide more
			// context about the company. Custom fields often contain important
			// metadata relevant to the company.
			companyGetRequest.Filters.Include = []projects.CompanyRequestSideload{
				projects.CompanyRequestSideloadCustomFields,
				projects.CompanyRequestSideloadCustomFieldValues,
			}

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&companyGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.Company](&companyGetRequest.Fields.Company, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(companyGetRequest.Fields.Company) > 0 {
				// Drop the sideloads: they are not what the selection named, and
				// they would return the bulk it exists to avoid.
				companyGetRequest.Filters = projects.CompanyRequestFilters{}
				return helpers.NewRawToolResult(ctx, engine, companyGetRequest, "failed to get company",
					helpers.WebLinkerWithIDPathBuilder("/app/clients"),
				)
			}

			company, err := projects.CompanyGet(ctx, engine, companyGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get company")
			}

			encoded, err := json.Marshal(company)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/clients"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, company,
					helpers.WebLinkerWithIDPathBuilder("/app/clients"),
				),
			}, nil
		},
	}
}

// CompanyList lists companies in Teamwork.com.
func CompanyList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyList),
			Description: "List companies (aka clients).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Companies",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter companies by name. " +
							"Each word from the search term is used to match against the company name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids":                  helpers.TagIDsFilterSchema("companies"),
					"match_all_tags":           helpers.MatchAllTagsSchema(),
					"order_by":                 companyOrdering.orderBySchema(),
					"order_mode":               orderModeSchema(),
					"order_by_custom_field_id": orderByFieldIDSchema("companies", "customfield"),
					"page":                     helpers.PageSchema(),
					"page_size":                helpers.PageSizeSchema(),
					"verbose":                  helpers.VerboseSchema(),
					"count_only":               helpers.CountOnlySchema("companies"),
					"fields":                   helpers.FieldsSchema[projects.Company]("company"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(companyListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyListRequest projects.CompanyListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&companyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&companyListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&companyListRequest.Filters.MatchAllTags, "match_all_tags"),
				companyOrdering.param(&companyListRequest.Filters.OrderBy, &companyListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&companyListRequest.Filters.OrderByCustomFieldID, "order_by_custom_field_id"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.Company](&companyListRequest.Filters.Fields.Companies, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			switch {
			case len(companyListRequest.Filters.Fields.Companies) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and sideloading custom
				// fields would smuggle back the bulk the selection exists to avoid.

			case verbose:
				// Include custom fields and values in the response to provide more
				// context about the company. Custom fields often contain important
				// metadata relevant to the company.
				companyListRequest.Filters.Include = []projects.CompanyRequestSideload{
					projects.CompanyRequestSideloadCustomFields,
					projects.CompanyRequestSideloadCustomFieldValues,
				}

			default:
				companyListRequest.Filters.Fields.Companies = []projects.CompanyField{
					projects.CompanyFieldID,
					projects.CompanyFieldName,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, companyListRequest, "failed to count companies")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, companyListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list companies")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list companies"), "failed to list companies")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/clients"))
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
