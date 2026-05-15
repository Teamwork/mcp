package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

var (
	companyGetOutputSchema  *jsonschema.Schema
	companyListOutputSchema *jsonschema.Schema
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
			var companyCreateRequest projects.CompanyCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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

// CompanyUpdate updates a company in Teamwork.com.
func CompanyUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCompanyUpdate),
			Description: "Update company (aka client).",
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
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				Title:        "List Companies",
				ReadOnlyHint: true,
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
					"tag_ids":        helpers.TagIDsFilterSchema("companies"),
					"match_all_tags": helpers.MatchAllTagsSchema("companies"),
					"page":           helpers.PageSchema(),
					"page_size":      helpers.PageSizeSchema(),
					"verbose":        helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(companyListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyListRequest projects.CompanyListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&companyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&companyListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&companyListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if verbose {
				// Include custom fields and values in the response to provide more
				// context about the company. Custom fields often contain important
				// metadata relevant to the company.
				companyListRequest.Filters.Include = []projects.CompanyRequestSideload{
					projects.CompanyRequestSideloadCustomFields,
					projects.CompanyRequestSideloadCustomFieldValues,
				}
			} else {
				companyListRequest.Filters.Fields.Companies = []projects.CompanyField{
					projects.CompanyFieldID,
					projects.CompanyFieldName,
				}
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
