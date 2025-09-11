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

const (
	// Read operations
	MethodRateUserGet               toolsets.Method = "twprojects-get_user_rates"
	MethodRateInstallationUserList  toolsets.Method = "twprojects-list_installation_user_rates"
	MethodRateInstallationUserGet   toolsets.Method = "twprojects-get_installation_user_rate"
	MethodRateProjectGet            toolsets.Method = "twprojects-get_project_rate"
	MethodRateProjectUserList       toolsets.Method = "twprojects-list_project_user_rates"
	MethodRateProjectUserGet        toolsets.Method = "twprojects-get_project_user_rate"
	MethodRateProjectUserHistoryGet toolsets.Method = "twprojects-get_project_user_rate_history"

	// Write operations
	MethodRateInstallationUserUpdate     toolsets.Method = "twprojects-update_installation_user_rate"
	MethodRateInstallationUserBulkUpdate toolsets.Method = "twprojects-bulk_update_installation_user_rates"
	MethodRateProjectUpdate              toolsets.Method = "twprojects-update_project_rate"
	MethodRateProjectAndUsersUpdate      toolsets.Method = "twprojects-update_project_and_user_rates"
	MethodRateProjectUserUpdate          toolsets.Method = "twprojects-update_project_user_rate"
)

const ratesDescription = "The rates feature in Teamwork.com enables organizations to manage billing and cost " +
	"rates for users across projects. Rates can be configured at multiple levels: installation-wide default rates, " +
	"project-specific rates, and individual user rates. This hierarchical system allows for flexible rate management " +
	"where project-specific rates override installation defaults, and user-specific rates take precedence over " +
	"both. Rates support multi-currency configurations and maintain historical tracking for accurate financial " +
	"reporting and billing. Both billable rates (for client billing) and cost rates (for internal cost tracking) " +
	"are supported, providing comprehensive financial oversight of project work."

func init() {
	toolsets.RegisterMethod(MethodRateUserGet)
	toolsets.RegisterMethod(MethodRateInstallationUserList)
	toolsets.RegisterMethod(MethodRateInstallationUserGet)
	toolsets.RegisterMethod(MethodRateProjectGet)
	toolsets.RegisterMethod(MethodRateProjectUserList)
	toolsets.RegisterMethod(MethodRateProjectUserGet)
	toolsets.RegisterMethod(MethodRateProjectUserHistoryGet)
	toolsets.RegisterMethod(MethodRateInstallationUserUpdate)
	toolsets.RegisterMethod(MethodRateInstallationUserBulkUpdate)
	toolsets.RegisterMethod(MethodRateProjectUpdate)
	toolsets.RegisterMethod(MethodRateProjectAndUsersUpdate)
	toolsets.RegisterMethod(MethodRateProjectUserUpdate)
}

// RateUserGet retrieves all rates for a specific user.
func RateUserGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateUserGet),
			mcp.WithDescription("Get all rates for a specific user in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the user to get rates for."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results. Defaults to 1."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination. Defaults to 50."),
			),
			mcp.WithBoolean("include_installation_rate",
				mcp.Description("Include the installation rate in the response. Defaults to false."),
			),
			mcp.WithBoolean("include_user_cost",
				mcp.Description("Include the user cost in the response. Defaults to false."),
			),
			mcp.WithBoolean("include_archived_projects",
				mcp.Description("Include archived projects in the response. Defaults to false."),
			),
			mcp.WithBoolean("include_deleted_projects",
				mcp.Description("Include deleted projects in the response. Defaults to false."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateUserGetRequest projects.RateUserGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateUserGetRequest.Path.ID, "id"),
				helpers.OptionalNumericParam(&rateUserGetRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&rateUserGetRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&rateUserGetRequest.Filters.IncludeInstallationRate, "include_installation_rate"),
				helpers.OptionalParam(&rateUserGetRequest.Filters.IncludeUserCost, "include_user_cost"),
				helpers.OptionalParam(&rateUserGetRequest.Filters.IncludeArchivedProjects, "include_archived_projects"),
				helpers.OptionalParam(&rateUserGetRequest.Filters.IncludeDeletedProjects, "include_deleted_projects"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			// Set defaults if not provided
			if rateUserGetRequest.Filters.Page == 0 {
				rateUserGetRequest.Filters.Page = 1
			}
			if rateUserGetRequest.Filters.PageSize == 0 {
				rateUserGetRequest.Filters.PageSize = 50
			}

			userRates, err := projects.RateUserGet(ctx, engine, rateUserGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get user rates")
			}

			encoded, err := json.Marshal(userRates)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// RateInstallationUserList lists all users' installation rates.
func RateInstallationUserList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateInstallationUserList),
			mcp.WithDescription("List all users' installation rates in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results. Defaults to 1."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination. Defaults to 50."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateInstallationUserListRequest projects.RateInstallationUserListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericParam(&rateInstallationUserListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&rateInstallationUserListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			// Set defaults if not provided
			if rateInstallationUserListRequest.Filters.Page == 0 {
				rateInstallationUserListRequest.Filters.Page = 1
			}
			if rateInstallationUserListRequest.Filters.PageSize == 0 {
				rateInstallationUserListRequest.Filters.PageSize = 50
			}

			installationUserRates, err := projects.RateInstallationUserList(ctx, engine, rateInstallationUserListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list installation user rates")
			}

			encoded, err := json.Marshal(installationUserRates)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// RateInstallationUserGet retrieves a user's default installation rate.
func RateInstallationUserGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateInstallationUserGet),
			mcp.WithDescription("Get a user's default installation rate in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("user_id",
				mcp.Required(),
				mcp.Description("The ID of the user to get the installation rate for."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateInstallationUserGetRequest projects.RateInstallationUserGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateInstallationUserGetRequest.Path.UserID, "user_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			installationUserRate, err := projects.RateInstallationUserGet(ctx, engine, rateInstallationUserGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get installation user rate")
			}

			encoded, err := json.Marshal(installationUserRate)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// RateProjectGet retrieves a project's default rate.
func RateProjectGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectGet),
			mcp.WithDescription("Get a project's default rate in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get the rate for."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectGetRequest projects.RateProjectGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectGetRequest.Path.ProjectID, "project_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			projectRate, err := projects.RateProjectGet(ctx, engine, rateProjectGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project rate")
			}

			encoded, err := json.Marshal(projectRate)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}

// RateProjectUserList lists all users' rates for a project.
func RateProjectUserList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectUserList),
			mcp.WithDescription("List all users' rates for a project in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to get user rates for."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter users by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results. Defaults to 1."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination. Defaults to 50."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectUserListRequest projects.RateProjectUserListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectUserListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&rateProjectUserListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&rateProjectUserListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&rateProjectUserListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			// Set defaults if not provided
			if rateProjectUserListRequest.Filters.Page == 0 {
				rateProjectUserListRequest.Filters.Page = 1
			}
			if rateProjectUserListRequest.Filters.PageSize == 0 {
				rateProjectUserListRequest.Filters.PageSize = 50
			}

			projectUserRates, err := projects.RateProjectUserList(ctx, engine, rateProjectUserListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list project user rates")
			}

			encoded, err := json.Marshal(projectUserRates)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}

// RateProjectUserGet retrieves a specific user's rate for a project.
func RateProjectUserGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectUserGet),
			mcp.WithDescription("Get a specific user's rate for a project in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project."),
			),
			mcp.WithNumber("user_id",
				mcp.Required(),
				mcp.Description("The ID of the user to get the rate for."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectUserGetRequest projects.RateProjectUserGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectUserGetRequest.Path.ProjectID, "project_id"),
				helpers.RequiredNumericParam(&rateProjectUserGetRequest.Path.UserID, "user_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			projectUserRate, err := projects.RateProjectUserGet(ctx, engine, rateProjectUserGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project user rate")
			}

			encoded, err := json.Marshal(projectUserRate)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}

// RateProjectUserHistoryGet retrieves a user's rate history for a project.
func RateProjectUserHistoryGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectUserHistoryGet),
			mcp.WithDescription("Get a user's rate history for a project in Teamwork.com. "+ratesDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project."),
			),
			mcp.WithNumber("user_id",
				mcp.Required(),
				mcp.Description("The ID of the user to get the rate history for."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results. Defaults to 1."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination. Defaults to 50."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectUserHistoryGetRequest projects.RateProjectUserHistoryGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectUserHistoryGetRequest.Path.ProjectID, "project_id"),
				helpers.RequiredNumericParam(&rateProjectUserHistoryGetRequest.Path.UserID, "user_id"),
				helpers.OptionalNumericParam(&rateProjectUserHistoryGetRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&rateProjectUserHistoryGetRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			// Set defaults if not provided
			if rateProjectUserHistoryGetRequest.Filters.Page == 0 {
				rateProjectUserHistoryGetRequest.Filters.Page = 1
			}
			if rateProjectUserHistoryGetRequest.Filters.PageSize == 0 {
				rateProjectUserHistoryGetRequest.Filters.PageSize = 50
			}

			projectUserRateHistory, err := projects.RateProjectUserHistoryGet(ctx, engine, rateProjectUserHistoryGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project user rate history")
			}

			encoded, err := json.Marshal(projectUserRateHistory)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}

// RateInstallationUserUpdate sets a user's default installation rate.
func RateInstallationUserUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateInstallationUserUpdate),
			mcp.WithDescription("Set a user's default installation rate in Teamwork.com. "+ratesDescription),
			mcp.WithNumber("user_id",
				mcp.Required(),
				mcp.Description("The ID of the user to set the installation rate for."),
			),
			mcp.WithNumber("user_rate",
				mcp.Required(),
				mcp.Description("The rate amount for the user."),
			),
			mcp.WithNumber("currency_id",
				mcp.Description("The ID of the currency for the rate (optional, only used in multi-currency mode)."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateInstallationUserUpdateRequest projects.RateInstallationUserUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateInstallationUserUpdateRequest.Path.UserID, "user_id"),
				helpers.OptionalNumericPointerParam(&rateInstallationUserUpdateRequest.UserRate, "user_rate"),
				helpers.OptionalNumericPointerParam(&rateInstallationUserUpdateRequest.CurrencyID, "currency_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.RateInstallationUserUpdate(ctx, engine, rateInstallationUserUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update installation user rate")
			}

			return mcp.NewToolResultText("Installation user rate updated successfully"), nil
		},
	}
}

// RateInstallationUserBulkUpdate performs bulk update of user installation rates.
func RateInstallationUserBulkUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateInstallationUserBulkUpdate),
			mcp.WithDescription("Bulk update installation rates for users in Teamwork.com. "+ratesDescription),
			mcp.WithNumber("user_rate",
				mcp.Required(),
				mcp.Description("The rate amount to set for users."),
			),
			mcp.WithBoolean("all",
				mcp.Description("Whether to update all users. Defaults to false."),
			),
			mcp.WithArray("ids",
				mcp.Description("Array of user IDs to update (if all is false)."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("exclude_ids",
				mcp.Description("Array of user IDs to exclude (if all is true)."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithNumber("currency_id",
				mcp.Description("The ID of the currency for the rate (optional, only used in multi-currency mode)."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateInstallationUserBulkUpdateRequest projects.RateInstallationUserBulkUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericPointerParam(&rateInstallationUserBulkUpdateRequest.UserRate, "user_rate"),
				helpers.OptionalParam(&rateInstallationUserBulkUpdateRequest.All, "all"),
				helpers.OptionalNumericListParam(&rateInstallationUserBulkUpdateRequest.IDs, "ids"),
				helpers.OptionalNumericListParam(&rateInstallationUserBulkUpdateRequest.ExcludeIDs, "exclude_ids"),
				helpers.OptionalNumericPointerParam(&rateInstallationUserBulkUpdateRequest.CurrencyID, "currency_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.RateInstallationUserBulkUpdate(ctx, engine, rateInstallationUserBulkUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to bulk update installation user rates")
			}

			return mcp.NewToolResultText("Bulk updated installation user rates successfully"), nil
		},
	}
}

// RateProjectUpdate sets a project's default rate.
func RateProjectUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectUpdate),
			mcp.WithDescription("Set a project's default rate in Teamwork.com. "+ratesDescription),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to set the rate for."),
			),
			mcp.WithNumber("project_rate",
				mcp.Required(),
				mcp.Description("The rate amount for the project."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectUpdateRequest projects.RateProjectUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectUpdateRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericPointerParam(&rateProjectUpdateRequest.ProjectRate, "project_rate"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.RateProjectUpdate(ctx, engine, rateProjectUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project rate")
			}

			return mcp.NewToolResultText("Project rate updated successfully"), nil
		},
	}
}

// RateProjectAndUsersUpdate sets project rate and user rates together.
func RateProjectAndUsersUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectAndUsersUpdate),
			mcp.WithDescription("Set project rate and user rates together in Teamwork.com. "+ratesDescription),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project."),
			),
			mcp.WithNumber("project_rate",
				mcp.Description("The project's default rate amount."),
			),
			mcp.WithArray("user_rates",
				mcp.Description("Array of user rate objects to set for the project."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"user_id": map[string]any{
							"type":        "integer",
							"description": "The ID of the user.",
						},
						"user_rate": map[string]any{
							"type":        "number",
							"description": "The rate amount for the user.",
						},
					},
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectAndUsersUpdateRequest projects.RateProjectAndUsersUpdateRequest

			args := request.GetArguments()

			err := helpers.ParamGroup(args,
				helpers.RequiredNumericParam(&rateProjectAndUsersUpdateRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericParam(&rateProjectAndUsersUpdateRequest.ProjectRate, "project_rate"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			// Parse user_rates if provided
			if userRatesRaw, exists := args["user_rates"]; exists && userRatesRaw != nil {
				userRatesArray, ok := userRatesRaw.([]interface{})
				if !ok {
					return mcp.NewToolResultErrorFromErr("invalid parameters", fmt.Errorf("user_rates must be an array")), nil
				}

				for _, userRateRaw := range userRatesArray {
					userRateMap, ok := userRateRaw.(map[string]interface{})
					if !ok {
						return mcp.NewToolResultErrorFromErr("invalid parameters", fmt.Errorf("each user_rate must be an object")), nil
					}

					var userRate projects.ProjectUserRateRequest
					if userID, exists := userRateMap["user_id"]; exists {
						if userIDFloat, ok := userID.(float64); ok {
							userRate.User = twapi.Relationship{ID: int64(userIDFloat)}
						}
					}
					if userRateVal, exists := userRateMap["user_rate"]; exists {
						if userRateFloat, ok := userRateVal.(float64); ok {
							userRate.UserRate = int64(userRateFloat)
						}
					}

					rateProjectAndUsersUpdateRequest.UserRates = append(rateProjectAndUsersUpdateRequest.UserRates, userRate)
				}
			}

			_, err = projects.RateProjectAndUsersUpdate(ctx, engine, rateProjectAndUsersUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project and user rates")
			}

			userCount := len(rateProjectAndUsersUpdateRequest.UserRates)
			if userCount > 0 {
				return mcp.NewToolResultText(fmt.Sprintf("Project rate and %d user rates updated successfully", userCount)), nil
			}
			return mcp.NewToolResultText("Project rate updated successfully"), nil
		},
	}
}

// RateProjectUserUpdate sets a user's rate for a specific project.
func RateProjectUserUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodRateProjectUserUpdate),
			mcp.WithDescription("Set a user's rate for a specific project in Teamwork.com. "+ratesDescription),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project."),
			),
			mcp.WithNumber("user_id",
				mcp.Required(),
				mcp.Description("The ID of the user to set the rate for."),
			),
			mcp.WithNumber("user_rate",
				mcp.Required(),
				mcp.Description("The rate amount for the user."),
			),
			mcp.WithNumber("currency_id",
				mcp.Description("The ID of the currency for the rate (optional, only used in multi-currency mode)."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var rateProjectUserUpdateRequest projects.RateProjectUserUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&rateProjectUserUpdateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredNumericParam(&rateProjectUserUpdateRequest.Path.UserID, "user_id"),
				helpers.OptionalNumericPointerParam(&rateProjectUserUpdateRequest.UserRate, "user_rate"),
				helpers.OptionalNumericPointerParam(&rateProjectUserUpdateRequest.CurrencyID, "currency_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.RateProjectUserUpdate(ctx, engine, rateProjectUserUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project user rate")
			}

			return mcp.NewToolResultText("Project user rate updated successfully"), nil
		},
	}
}
