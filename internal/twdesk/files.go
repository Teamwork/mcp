package twdesk

import (
	"context"
	"encoding/base64"
	"fmt"

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
	MethodFileCreate toolsets.Method = "twdesk-create_file"
)

func init() {
	toolsets.RegisterMethod(MethodFileCreate)
}

// FileCreate creates a file in Teamwork Desk
func FileCreate(client *deskclient.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodFileCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create File",
			},
			Description: "Upload a new file to Teamwork Desk, enabling attachment to tickets, articles, or " +
				"other resources.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the file.",
					},
					"mimeType": {
						Type:        "string",
						Description: "The MIME type of the file.",
					},
					"disposition": {
						Type:        "string",
						Description: "The disposition of the file.",
						Enum: []any{
							string(deskmodels.DispositionAttachment),
							string(deskmodels.DispositionAttachmentInline),
						},
					},
					"data": {
						Type:        "string",
						Description: "The content of the file as a base64-encoded string.",
					},
				},
				Required: []string{"name", "mimeType", "data"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			file, err := client.Files.Create(ctx, &deskmodels.FileResponse{
				File: deskmodels.File{
					Filename: arguments.GetString("name", ""),
					MIMEType: arguments.GetString("mimeType", "application/octet-stream"),
					Disposition: deskmodels.Disposition(
						arguments.GetString(
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

			dataStr := arguments.GetString("data", "")
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
			return helpers.NewToolResultText("File created successfully with ID %d", file.File.ID), nil
		},
	}
}
