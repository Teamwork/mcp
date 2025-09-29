package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodCompanyCreate)
	toolsets.RegisterMethod(MethodCompanyUpdate)
	toolsets.RegisterMethod(MethodCompanyDelete)
	toolsets.RegisterMethod(MethodCompanyGet)
	toolsets.RegisterMethod(MethodCompanyList)
}

// CompanyCreate creates a company in Teamwork.com.
func CompanyCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyCreate),
			mcp.WithDescription("Create a new company in Teamwork.com. "+companyDescription),
			mcp.WithTitleAnnotation("Create Company"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the company."),
			),
			mcp.WithString("address_one",
				mcp.Description("The first line of the address of the company."),
			),
			mcp.WithString("address_two",
				mcp.Description("The second line of the address of the company."),
			),
			mcp.WithString("city",
				mcp.Description("The city of the company."),
			),
			mcp.WithString("state",
				mcp.Description("The state of the company."),
			),
			mcp.WithString("zip",
				mcp.Description("The ZIP or postal code of the company."),
			),
			mcp.WithString("country_code",
				mcp.Description("The country code of the company, e.g., 'US' for the United States."),
			),
			mcp.WithString("phone",
				mcp.Description("The phone number of the company."),
			),
			mcp.WithString("fax",
				mcp.Description("The fax number of the company."),
			),
			mcp.WithString("email_one",
				mcp.Description("The primary email address of the company."),
			),
			mcp.WithString("email_two",
				mcp.Description("The secondary email address of the company."),
			),
			mcp.WithString("email_three",
				mcp.Description("The tertiary email address of the company."),
			),
			mcp.WithString("website",
				mcp.Description("The website of the company."),
			),
			mcp.WithString("profile",
				mcp.Description("A profile description for the company."),
			),
			mcp.WithNumber("manager_id",
				mcp.Description("The ID of the user who manages the company."),
			),
			mcp.WithNumber("industry_id",
				mcp.Description("The ID of the industry the company belongs to."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the company."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyCreateRequest projects.CompanyCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			companyResponse, err := projects.CompanyCreate(ctx, engine, companyCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create company")
			}

			msg := fmt.Sprintf("Company created successfully with ID %d", companyResponse.Company.ID)
			return mcp.NewToolResultText(msg), nil
		},
	}
}

// CompanyUpdate updates a company in Teamwork.com.
func CompanyUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyUpdate),
			mcp.WithDescription("Update an existing company in Teamwork.com. "+companyDescription),
			mcp.WithTitleAnnotation("Update Company"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the company to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the company."),
			),
			mcp.WithString("address_one",
				mcp.Description("The first line of the address of the company."),
			),
			mcp.WithString("address_two",
				mcp.Description("The second line of the address of the company."),
			),
			mcp.WithString("city",
				mcp.Description("The city of the company."),
			),
			mcp.WithString("state",
				mcp.Description("The state of the company."),
			),
			mcp.WithString("zip",
				mcp.Description("The ZIP or postal code of the company."),
			),
			mcp.WithString("country_code",
				mcp.Description("The country code of the company, e.g., 'US' for the United States."),
			),
			mcp.WithString("phone",
				mcp.Description("The phone number of the company."),
			),
			mcp.WithString("fax",
				mcp.Description("The fax number of the company."),
			),
			mcp.WithString("email_one",
				mcp.Description("The primary email address of the company."),
			),
			mcp.WithString("email_two",
				mcp.Description("The secondary email address of the company."),
			),
			mcp.WithString("email_three",
				mcp.Description("The tertiary email address of the company."),
			),
			mcp.WithString("website",
				mcp.Description("The website of the company."),
			),
			mcp.WithString("profile",
				mcp.Description("A profile description for the company."),
			),
			mcp.WithNumber("manager_id",
				mcp.Description("The ID of the user who manages the company."),
			),
			mcp.WithNumber("industry_id",
				mcp.Description("The ID of the industry the company belongs to."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the company."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyUpdateRequest projects.CompanyUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.CompanyUpdate(ctx, engine, companyUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update company")
			}

			return mcp.NewToolResultText("Company updated successfully"), nil
		},
	}
}

// CompanyDelete deletes a company in Teamwork.com.
func CompanyDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyDelete),
			mcp.WithDescription("Delete an existing company in Teamwork.com. "+companyDescription),
			mcp.WithTitleAnnotation("Delete Company"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the company to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyDeleteRequest projects.CompanyDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&companyDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.CompanyDelete(ctx, engine, companyDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete company")
			}

			return mcp.NewToolResultText("Company deleted successfully"), nil
		},
	}
}

// CompanyGet retrieves a company in Teamwork.com.
func CompanyGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyGet),
			mcp.WithDescription("Get an existing company in Teamwork.com. "+companyDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Company"),
			mcp.WithOutputSchema[projects.CompanyGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the company to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyGetRequest projects.CompanyGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&companyGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			company, err := projects.CompanyGet(ctx, engine, companyGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get company")
			}

			encoded, err := json.Marshal(company)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/clients"),
			))), nil
		},
	}
}

// CompanyList lists companies in Teamwork.com.
func CompanyList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCompanyList),
			mcp.WithDescription("List companies in Teamwork.com. "+companyDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Companies"),
			mcp.WithOutputSchema[projects.CompanyListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter companies by name. "+
					"Each word from the search term is used to match against the company name."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter companies by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match companies that have all the specified tags. "+
					"If false, the search will match companies that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var companyListRequest projects.CompanyListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&companyListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&companyListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&companyListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&companyListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			companyList, err := projects.CompanyList(ctx, engine, companyListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list companies")
			}

			encoded, err := json.Marshal(companyList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/clients"),
			))), nil
		},
	}
}
