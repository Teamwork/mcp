package twdesk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCompanyCreate toolsets.Method = "twdesk-create_company"
	MethodCompanyUpdate toolsets.Method = "twdesk-update_company"
	MethodCompanyGet    toolsets.Method = "twdesk-get_company"
	MethodCompanyList   toolsets.Method = "twdesk-list_companies"
)

// CompanyGet finds a company in Teamwork Desk.  This will find it by ID
func CompanyGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Company",
				ReadOnlyHint: true,
			},
			Description: "Get Desk company (customer organization).",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			company, err := client.Companies.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return nil, fmt.Errorf("failed to get company: %w", err)
			}

			return helpers.NewToolResultJSON(company)
		},
	}
}

// CompanyList returns a list of companies that apply to the filters in Teamwork Desk
func CompanyList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the company to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "null"},
			},
		},
		"domains": {
			Description: "The domains of the company to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"kind": {
			Description: "The kind of the company to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string", Enum: []any{"company", "group"}},
				{Type: "null"},
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Companies",
				ReadOnlyHint: true,
			},
			Description: "List Desk companies. Filter by name, domains, or kind.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required:   []string{},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Apply filters to the company list
			name := arguments.GetString("name", "")
			domains := arguments.GetStringSlice("domains", []string{})
			kind := arguments.GetString("kind", "")

			filter := deskclient.NewFilter()
			if name != "" {
				filter = filter.Eq("name", name)
			}

			if kind != "" {
				filter = filter.Eq("kind", kind)
			}

			if len(domains) > 0 {
				filter = filter.In("domains", helpers.SliceToAny(domains))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			companies, err := client.Companies.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list companies: %w", err)
			}
			return helpers.NewToolResultJSON(companies)
		},
	}
}

// CompanyCreate creates a company in Teamwork Desk
func CompanyCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Company",
			},
			Description: "Create Desk company.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the company.",
					},
					"description": {
						Description: "The description of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"details": {
						Description: "The details of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"industry": {
						Description: "The industry of the company.",
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
					"permission": {
						Description: "The permission level of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"own", "all"}},
							{Type: "null"},
						},
					},
					"kind": {
						Description: "The kind of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"company", "group"}},
							{Type: "null"},
						},
					},
					"note": {
						Description: "The note for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"domains": {
						Description: "The domains for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				d := domain
				domainEntities[i] = deskmodels.Domain{
					Name: &d,
				}
			}

			name := arguments.GetString("name", "")
			company, err := client.Companies.Create(ctx, &deskmodels.CompanyResponse{
				Company: deskmodels.Company{
					Name:        &name,
					Description: strPtr(arguments.GetString("description", "")),
					Details:     strPtr(arguments.GetString("details", "")),
					Industry:    strPtr(arguments.GetString("industry", "")),
					Website:     strPtr(arguments.GetString("website", "")),
					Permission:  strPtr(arguments.GetString("permission", "")),
					Kind:        strPtr(arguments.GetString("kind", "")),
					Note:        strPtr(arguments.GetString("note", "")),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}
			return helpers.NewToolResultText("Company created successfully with ID %d", company.Company.ID), nil
		},
	}
}

// CompanyUpdate updates a company in Teamwork Desk
func CompanyUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Company",
			},
			Description: "Update Desk company.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to update.",
					},
					"name": {
						Description: "The new name of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The new description of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"details": {
						Description: "The new details of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"industry": {
						Description: "The new industry of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"website": {
						Description: "The new website of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"permission": {
						Description: "The new permission level of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"own", "all"}},
							{Type: "null"},
						},
					},
					"kind": {
						Description: "The new kind of the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"company", "group"}},
							{Type: "null"},
						},
					},
					"note": {
						Description: "The new note for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"domains": {
						Description: "The new domains for the company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				d := domain
				domainEntities[i] = deskmodels.Domain{
					Name: &d,
				}
			}
			_, err = client.Companies.Update(ctx, arguments.GetInt("id", 0), &deskmodels.CompanyResponse{
				Company: deskmodels.Company{
					Name:        strPtr(arguments.GetString("name", "")),
					Description: strPtr(arguments.GetString("description", "")),
					Details:     strPtr(arguments.GetString("details", "")),
					Industry:    strPtr(arguments.GetString("industry", "")),
					Website:     strPtr(arguments.GetString("website", "")),
					Permission:  strPtr(arguments.GetString("permission", "")),
					Kind:        strPtr(arguments.GetString("kind", "")),
					Note:        strPtr(arguments.GetString("note", "")),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update company: %w", err)
			}

			return helpers.NewToolResultText("Company updated successfully"), nil
		},
	}
}
