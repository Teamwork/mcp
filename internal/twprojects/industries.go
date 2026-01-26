package twprojects

import (
	"context"
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
	MethodIndustryList toolsets.Method = "twprojects-list_industries"
)

const industryDescription = "Industry refers to the business sector or market category that a company belongs to, " +
	"such as technology, healthcare, finance, or education. It helps provide context about the nature of a company's " +
	"work and can be used to better organize and filter data across the platform. By associating companies and " +
	"projects with specific industries, Teamwork.com allows teams to gain clearer insights, tailor communication, " +
	"and segment information in ways that make it easier to manage relationships and understand the broader business " +
	"landscape in which their clients and partners operate."

var (
	industryListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodIndustryList)

	var err error

	// generate the output schemas only once
	industryListOutputSchema, err = jsonschema.For[projects.IndustryListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for IndustryListResponse: %v", err))
	}
}

// IndustryList lists projects in Teamwork.com.
func IndustryList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodIndustryList),
			Description: "List industries in Teamwork.com. " + industryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Industries",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
			OutputSchema: industryListOutputSchema,
		},
		Handler: func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var industryListRequest projects.IndustryListRequest

			industryList, err := projects.IndustryList(ctx, engine, industryListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list industries")
			}
			return helpers.NewToolResultJSON(industryList)
		},
	}
}
