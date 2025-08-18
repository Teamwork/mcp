package twdesk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
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

func init() {
	toolsets.RegisterMethod(MethodCompanyCreate)
	toolsets.RegisterMethod(MethodCompanyUpdate)
	toolsets.RegisterMethod(MethodCompanyGet)
	toolsets.RegisterMethod(MethodCompanyList)
}

// CompanyGet finds a company in Teamwork Desk.  This will find it by ID
func CompanyGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyGet),
			mcp.WithDescription("Get a company from Teamwork Desk"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the company to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			company, err := client.Companies.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get company: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Company retrieved successfully: %s", company.Company.Name)), nil
		},
	}
}

// CompanyList returns a list of companies that apply to the filters in Teamwork Desk
func CompanyList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithDescription("List all companies in Teamwork Desk"),
		mcp.WithString("name", mcp.Description("The name of the company to filter by.")),
		mcp.WithArray("domains", mcp.Description("The domains of the company to filter by.")),
		mcp.WithString("kind", mcp.Description("The kind of the company to filter by."), mcp.Pattern("^(group|company)$")),
	}

	opts = append(opts, PaginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the company list
			name := request.GetString("name", "")
			domains := request.GetStringSlice("domains", []string{})
			kind := request.GetString("kind", "")

			filter := deskclient.NewFilter()
			if name != "" {
				filter = filter.Eq("name", name)
			}

			if kind != "" {
				filter = filter.Eq("kind", kind)
			}

			if len(domains) > 0 {
				filter = filter.In("domains", domains)
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			SetPagination(&params, request)

			companies, err := client.Companies.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list companies: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Companies retrieved successfully: %v", companies)), nil
		},
	}
}

// CompanyCreate creates a company in Teamwork Desk
func CompanyCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyCreate),
			mcp.WithDescription("Create a new company in Teamwork Desk"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the company."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			company, err := client.Companies.Create(ctx, &deskmodels.CompanyResponse{
				Company: deskmodels.Company{},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Company created successfully with ID %d", company.Company.ID)), nil
		},
	}
}

// CompanyUpdate updates a company in Teamwork Desk
func CompanyUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyUpdate),
			mcp.WithDescription("Update an existing company in Teamwork Desk"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the company to update."),
			),
			mcp.WithString("name",
				mcp.Description("The new name of the company."),
			),
			mcp.WithString("description",
				mcp.Description("The new description of the company."),
			),
			mcp.WithString("details",
				mcp.Description("The new details of the company."),
			),
			mcp.WithString("industry",
				mcp.Description("The new industry of the company."),
			),
			mcp.WithString("website",
				mcp.Description("The new website of the company."),
			),
			mcp.WithString("permission",
				mcp.Description("The new permission level of the company."),
			),
			mcp.WithString("kind",
				mcp.Description("The new kind of the company."),
			),
			mcp.WithString("note",
				mcp.Description("The new note for the company."),
			),
			mcp.WithArray("domains",
				mcp.Description("The new domains for the company."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			domains := request.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}
			_, err := client.Companies.Update(ctx, request.GetInt("id", 0), &deskmodels.CompanyResponse{
				Company: deskmodels.Company{
					Name:        request.GetString("name", ""),
					Description: request.GetString("description", ""),
					Details:     request.GetString("details", ""),
					Industry:    request.GetString("industry", ""),
					Website:     request.GetString("website", ""),
					Permission:  request.GetString("permission", ""),
					Kind:        request.GetString("kind", ""),
					Note:        request.GetString("note", ""),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}

			return mcp.NewToolResultText("Company updated successfully"), nil
		},
	}
}
