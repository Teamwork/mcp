package twdesk

import (
	"context"
	"encoding/base64"
	"fmt"

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
	MethodFileCreate toolsets.Method = "twdesk-create_file"
)

func init() {
	toolsets.RegisterMethod(MethodFileCreate)
}

// FileCreate creates a file in Teamwork Desk
func FileCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodFileCreate),
			mcp.WithTitleAnnotation("Create File"),
			mcp.WithOutputSchema[deskmodels.FileResponse](),
			mcp.WithDescription(
				"Upload a new file to Teamwork Desk, enabling attachment to tickets, articles, or "+
					"other resources."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the file."),
			),
			mcp.WithString("mimeType",
				mcp.Required(),
				mcp.Description("The MIME type of the file."),
			),
			mcp.WithString("disposition",
				mcp.Description("The disposition of the file."),
				mcp.Enum(
					string(deskmodels.DispositionAttachment),
					string(deskmodels.DispositionAttachmentInline),
				),
			),
			mcp.WithString("data",
				mcp.Required(),
				mcp.Description("The content of the file as a base64-encoded string."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			file, err := client.Files.Create(ctx, &deskmodels.FileResponse{
				File: deskmodels.File{
					Filename: request.GetString("name", ""),
					MIMEType: request.GetString("mimeType", "application/octet-stream"),
					Disposition: deskmodels.Disposition(
						request.GetString(
							"disposition",
							string(deskmodels.DispositionAttachment),
						),
					),
					Type: deskmodels.FileTypeAttachment,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create file: %w", err)
			}

			dataStr := request.GetString("data", "")
			if dataStr == "" {
				return nil, fmt.Errorf("file data (base64 encoded) is required")
			}

			fileData, err := base64.StdEncoding.DecodeString(dataStr)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 data: %w", err)
			}

			err = client.Files.Upload(ctx, file, fileData)
			if err != nil {
				return nil, fmt.Errorf("failed to upload file: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("File created successfully with ID %d", file.File.ID)), nil
		},
	}
}
