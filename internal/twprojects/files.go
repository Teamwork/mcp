package twprojects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"

	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodFileCreate      toolsets.Method = "twprojects-create_file"
	MethodUploadURLCreate toolsets.Method = "twprojects-create_upload_url"
	MethodProjectFileAdd  toolsets.Method = "twprojects-add_project_file"
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
			"References of files to attach to the %s, as returned by %s or %s. Each is \"tf_\" "+
				"followed by a UUID and the file extension, and can only be used once, so reserve "+
				"an upload for each place a file should go. Files are added to whatever is already "+
				"attached; nothing is removed.",
			entity, MethodUploadURLCreate, MethodFileCreate),
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
			Description: fmt.Sprintf("Upload short text you are generating yourself, such as a plan, a "+
				"spec or a CSV, so it can be attached to a task, comment or message. Returns a "+
				"single-use reference; pass it in attachment_refs on %s, %s, %s or %s. "+
				"Content is sent inline as base64, which means you have to emit the whole file as text, "+
				"so use %s instead for anything that already exists as a file — it hands back a URL to "+
				"send the bytes to directly, and is the only safe option for a document that must stay "+
				"byte-for-byte identical.",
				MethodTaskCreate, MethodTaskUpdate, MethodCommentCreate, MethodMessageCreate,
				MethodUploadURLCreate),
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

// uploadURLCreateResult is what the tool hands back. It is everything the
// caller needs to send the contents itself; nothing further is required of this
// server between the reservation and the attachment.
type uploadURLCreateResult struct {
	Reference string            `json:"reference"`
	Name      string            `json:"name"`
	Size      int64             `json:"size"`
	Method    string            `json:"method"`
	UploadURL string            `json:"upload_url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at,omitempty"`
	Usage     string            `json:"usage"`
}

// UploadURLCreate reserves space for a file and hands back where to send it, so
// the contents never pass through this server or the model.
func UploadURLCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodUploadURLCreate),
			Description: fmt.Sprintf("Reserve an upload for a file and get back a short-lived URL to "+
				"send its bytes to, plus a single-use reference. Use this for any file "+
				"that already exists — a PDF, an image, a signed document — because the bytes go straight "+
				"from you to storage and are never read into the conversation. Send the file with the "+
				"returned method and URL, setting exactly the headers returned and no "+
				"authorization of your own. Then pass the reference in attachment_refs on %s, %s, %s or "+
				"%s. Prefer %s only for short text you are generating yourself.",
				MethodTaskCreate, MethodTaskUpdate, MethodCommentCreate, MethodMessageCreate,
				MethodFileCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Upload URL",
				// Nothing is destroyed by reserving space, and the URL addresses
				// Teamwork.com's own storage for the caller's own account, so the
				// world here is as closed as it is for the upload itself.
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:      "string",
						MinLength: new(1),
						Description: "The file name, including its extension, for example \"contract.pdf\". " +
							"Teamwork.com works out how to display the file from the extension, so a name " +
							"without one is harder to open. Any directory part is removed.",
					},
					"size": {
						Type:    "integer",
						Minimum: new(1.0),
						Description: "The exact size of the file in bytes. The reservation is signed against " +
							"this number, so an upload of any other length is rejected.",
					},
				},
				Required: []string{"name", "size"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var name string
			var size int64
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&name, "name"),
				helpers.RequiredNumericParam(&size, "size"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			name, err := sanitizeFileName(name)
			if err != nil {
				return helpers.NewToolResultTextError("invalid name: %s", err.Error()), nil
			}
			if size <= 0 {
				return helpers.NewToolResultTextError(
					"size must be greater than zero, so there is something to upload"), nil
			}

			presigned, err := projects.PendingFilePresignedURL(ctx, engine,
				projects.NewPendingFilePresignedURLRequest(name, size))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to reserve the upload")
			}

			// The rules for the PUT come from the SDK rather than from a second
			// guess here: the signature covers the headers it lists, and only the
			// storage service would notice one wrong.
			plan, err := projects.NewPendingFileUploadPlan(presigned.URL, name, "")
			if err != nil {
				return helpers.NewToolResultTextError("failed to describe the upload: %s", err.Error()), nil
			}

			headers := make(map[string]string, len(plan.Headers)+1)
			for key := range plan.Headers {
				headers[key] = plan.Headers.Get(key)
			}
			// Not one of the signed headers, but the storage service rejects a
			// chunked body, so the caller has to send it.
			headers["Content-Length"] = strconv.FormatInt(size, 10)

			result := uploadURLCreateResult{
				Reference: string(presigned.Ref),
				Name:      name,
				Size:      size,
				Method:    plan.Method,
				UploadURL: plan.URL,
				Headers:   headers,
				Usage: fmt.Sprintf("Send the bytes with %s to upload_url using exactly these headers, "+
					"then pass %q in attachment_refs on %s, %s, %s or %s.",
					plan.Method, presigned.Ref, MethodTaskCreate, MethodTaskUpdate,
					MethodCommentCreate, MethodMessageCreate),
			}
			if !plan.ExpiresAt.IsZero() {
				result.ExpiresAt = plan.ExpiresAt.UTC().Format(time.RFC3339)
			}
			return helpers.NewToolResultJSON(result)
		},
	}
}

// projectFileAddResult is what the tool hands back. ID is the durable handle: a
// reference is spent once, this can be attached as often as it is needed.
type projectFileAddResult struct {
	ID    int64  `json:"id"`
	Usage string `json:"usage"`
}

// ProjectFileAdd stores an uploaded file in a project's files area, which is
// what turns a single-use reference into a file that can be attached more than
// once.
func ProjectFileAdd(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodProjectFileAdd),
			Description: fmt.Sprintf("Store an uploaded file in a project's files area, where people "+
				"find it outside any one task or comment. Upload it first with %s, then pass the "+
				"reference here. Returns a numeric file ID which, unlike a reference, survives being "+
				"used: pass it in attachment_file_ids on %s or %s to attach the same file to as many "+
				"tasks as needed. Attaching a reference to a task, comment or message already files it "+
				"here, so use this tool to store a file on its own, to describe or categorise it, or "+
				"when it has to reach more than one place.",
				MethodUploadURLCreate, MethodTaskCreate, MethodTaskUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Add Project File",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project whose files area will hold the file.",
					},
					"reference": {
						Type:      "string",
						MinLength: new(1),
						Description: fmt.Sprintf("The reference of an uploaded file, as returned by %s or "+
							"%s. It is \"tf_\" followed by a UUID and the file extension, and can only be "+
							"used once.",
							MethodUploadURLCreate, MethodFileCreate),
					},
					"name": {
						Description: "Override the name the file was uploaded with, including its extension.",
						AnyOf:       []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"description": {
						Description: "A description of the file.",
						AnyOf:       []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"category_id": {
						Description: "File it under an existing file category. Wins over category_name.",
						AnyOf:       []*jsonschema.Schema{{Type: "integer"}, {Type: "null"}},
					},
					"category_name": {
						Description: "File it under a category with this name, creating one when the " +
							"project has none. Ignored when category_id is given.",
						AnyOf: []*jsonschema.Schema{{Type: "string"}, {Type: "null"}},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("file"),
					"private": {
						Description: "Hide the file from client users.",
						AnyOf:       []*jsonschema.Schema{{Type: "boolean"}, {Type: "null"}},
					},
					"auto_new_version": {
						Description: "Store it as a new version of an existing file with the same name in " +
							"the project, rather than as a separate file.",
						AnyOf: []*jsonschema.Schema{{Type: "boolean"}, {Type: "null"}},
					},
					"notify_current_user": {
						Description: "Notify the user adding the file. Defaults to false.",
						AnyOf:       []*jsonschema.Schema{{Type: "boolean"}, {Type: "null"}},
					},
				},
				Required: []string{"project_id", "reference"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var fileCreateRequest projects.FileCreateRequest
			var reference string
			// TagIDs is a LegacyNumericList, which the numeric list binder cannot
			// infer through, so it is bound as its underlying type and converted.
			var tagIDs []int64
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&fileCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&reference, "reference"),
				helpers.OptionalPointerParam(&fileCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&fileCreateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&fileCreateRequest.CategoryID, "category_id"),
				helpers.OptionalPointerParam(&fileCreateRequest.CategoryName, "category_name"),
				helpers.OptionalNumericListParam(&tagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&fileCreateRequest.Private, "private"),
				helpers.OptionalPointerParam(&fileCreateRequest.AutoNewVersion, "auto_new_version"),
				helpers.OptionalPointerParam(&fileCreateRequest.NotifyCurrentUser, "notify_current_user"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			fileCreateRequest.TagIDs = projects.LegacyNumericList(tagIDs)

			// A reference surrounded by whitespace is the model's, not the API's,
			// and the endpoint would reject it as unknown rather than as untidy.
			fileCreateRequest.PendingFileRef = projects.PendingFileRef(strings.TrimSpace(reference))
			if fileCreateRequest.PendingFileRef == "" {
				return helpers.NewToolResultTextError("reference must not be empty"), nil
			}

			file, err := projects.FileCreate(ctx, engine, fileCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to add the file to the project")
			}

			return helpers.NewToolResultJSON(projectFileAddResult{
				ID: int64(file.ID),
				Usage: fmt.Sprintf("Pass %d in attachment_file_ids on %s or %s. Unlike a reference, "+
					"it can be used more than once.", file.ID, MethodTaskCreate, MethodTaskUpdate),
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

// attachmentFileIDsSchema returns the schema for the file identifier parameter,
// which is what a caller uses to attach a file that already exists.
//
// It is a separate parameter rather than a value attachment_refs also accepts,
// because the two are not interchangeable: a reference is spent by the first
// attachment and an identifier is not. Telling them apart by the "tf_" prefix
// would work today and encode a guess about the API's identifier format that
// even the SDK declines to make.
func attachmentFileIDsSchema(entity string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: fmt.Sprintf(
			"IDs of files already in a project's files area to attach to the %s, as returned by "+
				"%s. Unlike a reference these can be used repeatedly, so this is how one file "+
				"reaches several tasks. Files are added to whatever is already attached; nothing "+
				"is removed.",
			entity, MethodProjectFileAdd),
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			{Type: "null"},
		},
	}
}

// parseTaskAttachments reads the attachments for the task tools, which take the
// structured form rather than a plain list, and are the only ones that can
// attach a file that already exists as well as a freshly uploaded one.
func parseTaskAttachments(arguments map[string]any) (*projects.TaskAttachments, *mcp.CallToolResult) {
	refs, toolResult := parseAttachmentRefs(arguments)
	if toolResult != nil {
		return nil, toolResult
	}

	var fileIDs []int64
	if err := helpers.ParamGroup(arguments,
		helpers.OptionalNumericListParam(&fileIDs, "attachment_file_ids"),
	); err != nil {
		return nil, helpers.NewToolResultTextError("invalid attachment_file_ids: %s", err.Error())
	}

	if len(refs) == 0 && len(fileIDs) == 0 {
		return nil, nil
	}

	var attachments projects.TaskAttachments
	if len(refs) > 0 {
		attachments.PendingFiles = make([]projects.TaskAttachmentPendingFile, 0, len(refs))
		for _, ref := range refs {
			attachments.PendingFiles = append(attachments.PendingFiles,
				projects.TaskAttachmentPendingFile{Reference: ref})
		}
	}
	if len(fileIDs) > 0 {
		attachments.Files = make([]projects.TaskAttachmentFile, 0, len(fileIDs))
		for _, fileID := range fileIDs {
			attachments.Files = append(attachments.Files, projects.TaskAttachmentFile{ID: fileID})
		}
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
