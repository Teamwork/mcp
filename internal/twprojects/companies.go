package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCompanyCreate toolsets.Method = "twprojects-create_company"
	MethodCompanyUpdate toolsets.Method = "twprojects-update_company"
	MethodCompanyDelete toolsets.Method = "twprojects-delete_company"
	MethodCompanyGet    toolsets.Method = "twprojects-get_company"
	MethodCompanyList   toolsets.Method = "twprojects-list_companies"
)

const companyDescription = "In the context of Teamwork.com, a company represents an organization or business entity " +
	"that can be associated with users, projects, and tasks within the platform, and it is often referred to as a " +
	"“client.” It serves as a way to group related users and projects under a single organizational umbrella, making " +
	"it easier to manage permissions, assign responsibilities, and organize work. Companies (or clients) are " +
	"frequently used to distinguish between internal teams and external collaborators, enabling teams to work " +
	"efficiently while maintaining clear boundaries around ownership, visibility, and access levels across different " +
	"projects."

var (
	companyGetOutputSchema  *jsonschema.Schema
	companyListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodCompanyCreate)
	toolsets.RegisterMethod(MethodCompanyUpdate)
	toolsets.RegisterMethod(MethodCompanyDelete)
	toolsets.RegisterMethod(MethodCompanyGet)
	toolsets.RegisterMethod(MethodCompanyList)

	var err error

	// generate the output schemas only once
	companyGetOutputSchema, err = jsonschema.For[projects.CompanyGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CompanyGetResponse: %v", err))
	}
	companyListOutputSchema, err = jsonschema.For[projects.CompanyListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CompanyListResponse: %v", err))
	}
}

// CompanyCreate creates a company in Teamwork.com.
func CompanyCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyCreate),
			Description: "Create a new company in Teamwork.com. " + companyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Company",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the company.",
					},
					"address_one": {
						Type:        "string",
						Description: "The first line of the address of the company.",
					},
					"address_two": {
						Type:        "string",
						Description: "The second line of the address of the company.",
					},
					"city": {
						Type:        "string",
						Description: "The city of the company.",
					},
					"state": {
						Type:        "string",
						Description: "The state of the company.",
					},
					"zip": {
						Type:        "string",
						Description: "The ZIP or postal code of the company.",
					},
					"country_code": {
						Type:        "string",
						Description: "The country code of the company, e.g., 'US' for the United States.",
					},
					"phone": {
						Type:        "string",
						Description: "The phone number of the company.",
					},
					"fax": {
						Type:        "string",
						Description: "The fax number of the company.",
					},
					"email_one": {
						Type:        "string",
						Description: "The primary email address of the company.",
					},
					"email_two": {
						Type:        "string",
						Description: "The secondary email address of the company.",
					},
					"email_three": {
						Type:        "string",
						Description: "The tertiary email address of the company.",
					},
					"website": {
						Type:        "string",
						Description: "The website of the company.",
					},
					"profile": {
						Type:        "string",
						Description: "A profile description for the company.",
					},
					"manager_id": {
						Type:        "integer",
						Description: "The ID of the user who manages the company.",
					},
					"industry_id": {
						Type:        "integer",
						Description: "The ID of the industry the company belongs to.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the company.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyCreateRequest projects.CompanyCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
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
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			companyResponse, err := projects.CompanyCreate(ctx, engine, companyCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create company")
			}
			return helpers.NewToolResultText("Company created successfully with ID %d", companyResponse.Company.ID), nil
		},
	}
}

// CompanyUpdate updates a company in Teamwork.com.
func CompanyUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyUpdate),
			Description: "Update an existing company in Teamwork.com. " + companyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Company",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the company.",
					},
					"address_one": {
						Type:        "string",
						Description: "The first line of the address of the company.",
					},
					"address_two": {
						Type:        "string",
						Description: "The second line of the address of the company.",
					},
					"city": {
						Type:        "string",
						Description: "The city of the company.",
					},
					"state": {
						Type:        "string",
						Description: "The state of the company.",
					},
					"zip": {
						Type:        "string",
						Description: "The ZIP or postal code of the company.",
					},
					"country_code": {
						Type:        "string",
						Description: "The country code of the company, e.g., 'US' for the United States.",
					},
					"phone": {
						Type:        "string",
						Description: "The phone number of the company.",
					},
					"fax": {
						Type:        "string",
						Description: "The fax number of the company.",
					},
					"email_one": {
						Type:        "string",
						Description: "The primary email address of the company.",
					},
					"email_two": {
						Type:        "string",
						Description: "The secondary email address of the company.",
					},
					"email_three": {
						Type:        "string",
						Description: "The tertiary email address of the company.",
					},
					"website": {
						Type:        "string",
						Description: "The website of the company.",
					},
					"profile": {
						Type:        "string",
						Description: "A profile description for the company.",
					},
					"manager_id": {
						Type:        "integer",
						Description: "The ID of the user who manages the company.",
					},
					"industry_id": {
						Type:        "integer",
						Description: "The ID of the industry the company belongs to.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the company.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyUpdateRequest projects.CompanyUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Delete an existing company in Teamwork.com. " + companyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Company",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&companyDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Get an existing company in Teamwork.com. " + companyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Company",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: companyGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyGetRequest projects.CompanyGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&companyGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
				StructuredContent: company,
			}, nil
		},
	}
}

// CompanyList lists companies in Teamwork.com.
func CompanyList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyList),
			Description: "List companies in Teamwork.com. " + companyDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Companies",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter companies by name. " +
							"Each word from the search term is used to match against the company name.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter companies by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match companies that have all the specified tags. " +
							"If false, the search will match companies that have any of the specified tags. " +
							"Defaults to false.",
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination of results.",
					},
					"page_size": {
						Type:        "integer",
						Description: "Number of results per page for pagination.",
					},
				},
			},
			OutputSchema: companyListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyListRequest projects.CompanyListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&companyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&companyListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&companyListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			companyList, err := projects.CompanyList(ctx, engine, companyListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list companies")
			}

			encoded, err := json.Marshal(companyList)
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
				StructuredContent: companyList,
			}, nil
		},
	}
}
