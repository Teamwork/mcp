package twdesk

import (
	"context"
	"fmt"
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

var (
	companyListOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodCompanyCreate)
	toolsets.RegisterMethod(MethodCompanyUpdate)
	toolsets.RegisterMethod(MethodCompanyGet)
	toolsets.RegisterMethod(MethodCompanyList)

	var err error

	// generate the output schemas only once
	companyListOutputSchema, err = jsonschema.For[deskmodels.CompaniesResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CompanyListResponse: %v", err))
	}
}

// CompanyGet finds a company in Teamwork Desk.  This will find it by ID
func CompanyGet(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Company",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific company in Teamwork Desk by its ID. " +
				"Useful for auditing company records, troubleshooting ticket associations, or " +
				"integrating Desk company data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			company, err := client.Companies.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get company: %w", err)
			}

			return helpers.NewToolResultText("Company retrieved successfully: %s", company.Company.Name), nil
		},
	}
}

// CompanyList returns a list of companies that apply to the filters in Teamwork Desk
func CompanyList(client *deskclient.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Type:        "string",
			Description: "The name of the company to filter by.",
		},
		"domains": {
			Type:        "array",
			Description: "The domains of the company to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"kind": {
			Type:        "string",
			Description: "The kind of the company to filter by.",
			Enum:        []any{"company", "group"},
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
			Description: "List all companies in Teamwork Desk, with optional filters for name, domains, and kind. " +
				"Enables users to audit, analyze, or synchronize company configurations for ticket management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
			OutputSchema: companyListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
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
func CompanyCreate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Company",
			},
			Description: "Create a new company in Teamwork Desk by specifying its name, domains, and other attributes. " +
				"Useful for onboarding new organizations, customizing Desk for business relationships, or " +
				"adapting support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the company.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the company.",
					},
					"details": {
						Type:        "string",
						Description: "The details of the company.",
					},
					"industry": {
						Type:        "string",
						Description: "The industry of the company.",
					},
					"website": {
						Type:        "string",
						Description: "The website of the company.",
					},
					"permission": {
						Type:        "string",
						Description: "The permission level of the company.",
						Enum:        []any{"own", "all"},
					},
					"kind": {
						Type:        "string",
						Description: "The kind of the company.",
						Enum:        []any{"company", "group"},
					},
					"note": {
						Type:        "string",
						Description: "The note for the company.",
					},
					"domains": {
						Type:        "array",
						Description: "The domains for the company.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}

			company, err := client.Companies.Create(ctx, &deskmodels.CompanyResponse{
				Company: deskmodels.Company{
					Name:        arguments.GetString("name", ""),
					Description: arguments.GetString("description", ""),
					Details:     arguments.GetString("details", ""),
					Industry:    arguments.GetString("industry", ""),
					Website:     arguments.GetString("website", ""),
					Permission:  arguments.GetString("permission", ""),
					Kind:        arguments.GetString("kind", ""),
					Note:        arguments.GetString("note", ""),
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
func CompanyUpdate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCompanyUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Company",
			},
			Description: "Update an existing company in Teamwork Desk by ID, allowing changes to its name, domains, and " +
				"other attributes. Supports evolving business relationships, rebranding, or correcting company records for " +
				"improved ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the company to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the company.",
					},
					"description": {
						Type:        "string",
						Description: "The new description of the company.",
					},
					"details": {
						Type:        "string",
						Description: "The new details of the company.",
					},
					"industry": {
						Type:        "string",
						Description: "The new industry of the company.",
					},
					"website": {
						Type:        "string",
						Description: "The new website of the company.",
					},
					"permission": {
						Type:        "string",
						Description: "The new permission level of the company.",
						Enum:        []any{"own", "all"},
					},
					"kind": {
						Type:        "string",
						Description: "The new kind of the company.",
						Enum:        []any{"company", "group"},
					},
					"note": {
						Type:        "string",
						Description: "The new note for the company.",
					},
					"domains": {
						Type:        "array",
						Description: "The new domains for the company.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}
			_, err = client.Companies.Update(ctx, arguments.GetInt("id", 0), &deskmodels.CompanyResponse{
				Company: deskmodels.Company{
					Name:        arguments.GetString("name", ""),
					Description: arguments.GetString("description", ""),
					Details:     arguments.GetString("details", ""),
					Industry:    arguments.GetString("industry", ""),
					Website:     arguments.GetString("website", ""),
					Permission:  arguments.GetString("permission", ""),
					Kind:        arguments.GetString("kind", ""),
					Note:        arguments.GetString("note", ""),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}

			return helpers.NewToolResultText("Company updated successfully"), nil
		},
	}
}
