package twprojects

import (
	"context"
	"encoding/json"

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
	MethodIndustryList toolsets.Method = "twprojects-list_industries"
)

const industryDescription = "Industry refers to the business sector or market category that a company belongs to, " +
	"such as technology, healthcare, finance, or education. It helps provide context about the nature of a company's " +
	"work and can be used to better organize and filter data across the platform. By associating companies and " +
	"projects with specific industries, Teamwork.com allows teams to gain clearer insights, tailor communication, " +
	"and segment information in ways that make it easier to manage relationships and understand the broader business " +
	"landscape in which their clients and partners operate."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodIndustryList)
}

// IndustryList lists projects in Teamwork.com.
func IndustryList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodIndustryList),
			mcp.WithDescription("List industries in Teamwork.com. "+industryDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Industries"),
			mcp.WithOutputSchema[projects.IndustryListResponse](),
		),
		Handler: func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var industryListRequest projects.IndustryListRequest

			industryList, err := projects.IndustryList(ctx, engine, industryListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list industries")
			}

			encoded, err := json.Marshal(industryList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}
