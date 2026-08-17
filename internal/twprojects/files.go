package twprojects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"

	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodFileCreate toolsets.Method = "twprojects-create_file"
)

// maxAttachmentBytes caps the decoded size of an inline attachment.
//
// The hard ceiling is the HTTP server's maximum request body, which covers the
// whole JSON-RPC envelope: base64 grows by four thirds, so 5 MB of file already
// costs about 6.7 MB on the wire before JSON escaping and the rest of the
// message. Rejecting here rather than at the transport turns an unreadable
// connection reset into a tool result the caller can act on.
//
// In practice the binding limit is far lower. The caller has to emit the base64
// itself, at roughly one output token per three bytes of file, so a megabyte is
// already out of reach. This limit exists to keep the server standing, not to
// describe what is usable.
const maxAttachmentBytes = 5 << 20

// maxFileNameBytes bounds the stored name. Most filesystems stop around 255
// bytes; the extension is preserved when truncating so the file still opens
// with the right application.
const maxFileNameBytes = 200

// attachmentRefsSchema returns the schema for the pending file references
// parameter. The entity is named in the description so the caller knows what it
// is attaching to.
//
// There is no minimum length: a caller that sends an empty array to mean "none"
// gets a no-op rather than a validation failure.
func attachmentRefsSchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf(
			"References of files to attach to the %s, as returned by %s. Each looks like "+
				"\"tf_1a2b\" and can only be used once, so upload a file for each place it "+
				"should go. Files are added to whatever is already attached; nothing is removed.",
			entity, MethodFileCreate),
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
			{Type: "null"},
		},
	}
}

// fileCreateResult is what the tool hands back. Reference is the value the
// attachment parameters take.
type fileCreateResult struct {
	Reference string `json:"reference"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Usage     string `json:"usage"`
}

// FileCreate uploads a file to Teamwork.com so that it can be attached to
// something.
func FileCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodFileCreate),
			Description: fmt.Sprintf("Upload a file so it can be attached to a task, comment or message. "+
				"Returns a single-use reference like \"tf_1a2b\"; pass it in attachment_refs on %s, %s, "+
				"%s or %s. Content is sent inline as base64, so this suits text you generated, such as "+
				"plans, specs or CSV, rather than large binaries.",
				MethodTaskCreate, MethodTaskUpdate, MethodCommentCreate, MethodMessageCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create File",
				// The contents go straight to storage rather than through the API,
				// but that storage is Teamwork.com's and the file lands in the
				// caller's own account, so the world here is still closed.
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:      "string",
						MinLength: new(1),
						Description: "The file name, including its extension, for example \"plan.md\". " +
							"Teamwork.com works out how to display the file from the extension, so a name " +
							"without one is harder to open. Any directory part is removed.",
					},
					"data": {
						Type:      "string",
						MaxLength: new(base64.StdEncoding.EncodedLen(maxAttachmentBytes)),
						Description: fmt.Sprintf("The file content, base64-encoded with the standard "+
							"alphabet. It must decode to between 1 byte and %d bytes.", maxAttachmentBytes),
					},
				},
				Required: []string{"name", "data"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var name, data string
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&name, "name"),
				helpers.RequiredParam(&data, "data"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			name, err := sanitizeFileName(name)
			if err != nil {
				return helpers.NewToolResultTextError("invalid name: %s", err.Error()), nil
			}

			// Check the encoded length first: decoding allocates the payload a
			// second time, and there is no reason to pay for that only to reject
			// the result.
			if len(data) > base64.StdEncoding.EncodedLen(maxAttachmentBytes) {
				return helpers.NewToolResultTextError(
					"file is too large: %d base64 characters exceed the %d byte limit on decoded "+
						"content. Upload a smaller file, or split the content across several.",
					len(data), maxAttachmentBytes), nil
			}

			// Not an API failure: the caller supplied malformed base64, so report
			// it as a tool result it can correct rather than as a transport error.
			content, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return helpers.NewToolResultTextError("failed to decode base64 data: %s", err.Error()), nil
			}
			switch {
			case len(content) == 0:
				return helpers.NewToolResultTextError(
					"the data decoded to zero bytes, so there is nothing to upload"), nil
			case len(content) > maxAttachmentBytes:
				return helpers.NewToolResultTextError(
					"file is too large: %d bytes exceed the %d byte limit. Upload a smaller file, "+
						"or split the content across several.",
					len(content), maxAttachmentBytes), nil
			}

			pendingFile, err := projects.PendingFileCreate(ctx, engine,
				projects.NewPendingFileCreateRequest(name, content))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to upload file")
			}

			return helpers.NewToolResultJSON(fileCreateResult{
				Reference: string(pendingFile.Ref),
				Name:      name,
				Size:      int64(len(content)),
				Usage: fmt.Sprintf("Pass %q in attachment_refs on %s, %s, %s or %s.",
					pendingFile.Ref, MethodTaskCreate, MethodTaskUpdate,
					MethodCommentCreate, MethodMessageCreate),
			})
		},
	}
}

// parseAttachmentRefs reads the attachment references shared by the tools that
// can attach a file. It returns nil when the caller named none, so that the
// request omits the field entirely rather than sending an empty set.
func parseAttachmentRefs(arguments map[string]any) ([]projects.PendingFileRef, *mcp.CallToolResult) {
	var refs []projects.PendingFileRef
	if err := helpers.ParamGroup(arguments,
		helpers.OptionalListParam(&refs, "attachment_refs"),
	); err != nil {
		return nil, helpers.NewToolResultTextError("invalid attachment_refs: %s", err.Error())
	}

	cleaned := make([]projects.PendingFileRef, 0, len(refs))
	for _, ref := range refs {
		if trimmed := projects.PendingFileRef(strings.TrimSpace(string(ref))); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return nil, nil
	}
	return cleaned, nil
}

// parseTaskAttachments reads the attachment references for the task tools,
// which take the structured form rather than a plain list.
func parseTaskAttachments(arguments map[string]any) (*projects.TaskAttachments, *mcp.CallToolResult) {
	refs, toolResult := parseAttachmentRefs(arguments)
	if toolResult != nil {
		return nil, toolResult
	}
	if len(refs) == 0 {
		return nil, nil
	}

	attachments := projects.TaskAttachments{
		PendingFiles: make([]projects.TaskAttachmentPendingFile, 0, len(refs)),
	}
	for _, ref := range refs {
		attachments.PendingFiles = append(attachments.PendingFiles,
			projects.TaskAttachmentPendingFile{Reference: ref})
	}
	return &attachments, nil
}

// sanitizeFileName reduces a caller-supplied name to something safe to store as
// an attachment name.
//
// Models emit paths rather than names, and both "docs/plan.md" and a Windows
// style path turn up. path.Base only understands the forward slash, so the
// backslash form has to be removed explicitly.
func sanitizeFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if index := strings.LastIndexAny(name, `/\`); index >= 0 {
		name = name[index+1:]
	}
	// Control characters would break any downstream header carrying the name.
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)

	switch name {
	case "", ".", "..":
		return "", fmt.Errorf("name must be a file name, not a path")
	}

	if len(name) > maxFileNameBytes {
		// Keep the extension so the file still opens with the right application,
		// and cut the stem on a rune boundary so the name stays valid UTF-8.
		extension := path.Ext(name)
		if len(extension) > maxFileNameBytes/2 {
			extension = "" // a late dot, not an extension
		}
		stem := name[:len(name)-len(extension)]
		stem = strings.ToValidUTF8(stem[:maxFileNameBytes-len(extension)], "")
		name = stem + extension
	}
	return name, nil
}
