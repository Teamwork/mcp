package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

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
	MethodCommentCreate toolsets.Method = "twprojects-create_comment"
	MethodCommentUpdate toolsets.Method = "twprojects-update_comment"
	MethodCommentDelete toolsets.Method = "twprojects-delete_comment"
	MethodCommentGet    toolsets.Method = "twprojects-get_comment"
	MethodCommentList   toolsets.Method = "twprojects-list_comments"
)

var (
	commentGetOutputSchema  *jsonschema.Schema
	commentListOutputSchema *jsonschema.Schema
)

// commentListFields is the attribute set list_comments asks for when the caller
// names none: every attribute the SDK models except htmlBody.
//
// htmlBody is the same content as body a second time, and the larger of the
// two — measured across one real account it came to 130% of the body payload —
// so a list response carried the text twice for no extra meaning. Selecting the
// rest server-side rather than stripping htmlBody from the response means we do
// not pay to transfer it either. get_comment still returns it, and a caller that
// wants it back in a list can name it in `fields`.
//
// Derived from the entity struct rather than restated as constants so an SDK
// upgrade that adds an attribute picks it up here instead of quietly excluding
// it.
var commentListFields = slices.DeleteFunc(
	helpers.SparseFieldNames[projects.CommentField, projects.Comment](),
	func(field projects.CommentField) bool {
		return field == projects.CommentFieldHTMLBody
	},
)

// commentBodyLimit is how many characters of a comment body list_comments
// returns before truncating it.
//
// Comment bodies are not short in practice: bots post build summaries and
// agents post long reports, so a list of a few hundred routinely ran to
// hundreds of thousands of tokens. The distribution is what makes a cap work —
// across one real account the median body was 484 characters while the top 5%
// of records held 52.8% of all body text. A 500-character cap therefore leaves
// roughly three quarters of comments untouched and still removes ~90% of the
// payload. Full text stays one get_comment away.
const commentBodyLimit = 500

func init() {
	var err error

	// generate the output schemas only once
	commentGetOutputSchema, err = jsonschema.For[projects.CommentGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(commentGetOutputSchema)
	commentListOutputSchema, err = jsonschema.For[projects.CommentListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CommentListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(commentListOutputSchema)
}

// CommentCreate creates a comment in Teamwork.com.
func CommentCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentCreate),
			Description: "Create comment on a task, milestone, notebook, file, or link.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Comment",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"object": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"type": {
								Type:        "string",
								Description: "The type of object to create the comment for.",
								Enum: []any{
									"tasks",
									"milestones",
									"files",
									"notebooks",
									"links",
								},
							},
							"id": {
								Type:        "integer",
								Description: "The ID of the object to create the comment for.",
							},
						},
						Required:    []string{"type", "id"},
						Description: "The object to create the comment for. It can be a tasks, milestones, files or notebooks.",
					},
					"body": {
						Type:        "string",
						Description: "The content of the comment. The content can be added as text or HTML.",
					},
					"content_type": {
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"TEXT", "HTML"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the new comment.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": helpers.NotifySchema("Who to notify of the new comment.", true),
				},
				Required: []string{"object", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentCreateRequest projects.CommentCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&commentCreateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentCreateRequest.ContentType, "content_type"),
				helpers.OptionalPointerParam(&commentCreateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			notifyChosen, notifiers, toolResult := parseNotify(arguments, true)
			if toolResult != nil {
				return toolResult, nil
			}
			switch notifyChosen {
			case notifyChoiceAll:
				commentCreateRequest.Notify = projects.NewCommentNotifyAll()
			case notifyChoiceGroup:
				commentCreateRequest.Notify = projects.NewCommentNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				commentCreateRequest.Notify = projects.NewCommentNotifyFollowers()
			}

			var objectType string
			var objectID int64
			object, ok := arguments["object"]
			if !ok {
				return helpers.NewToolResultTextError("missing required parameter: object"), nil
			}
			objectMap, ok := object.(map[string]any)
			if !ok {
				return helpers.NewToolResultTextError("invalid object: expected an object, got %T", object), nil
			} else if objectMap == nil {
				return helpers.NewToolResultTextError("object cannot be nil"), nil
			}
			err = helpers.ParamGroup(objectMap,
				helpers.RequiredParam(&objectType, "type"),
				helpers.RequiredNumericParam(&objectID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid object: %s", err.Error()), nil
			}

			switch strings.ToLower(objectType) {
			case "tasks":
				commentCreateRequest.Path.TaskID = objectID
			case "milestones":
				commentCreateRequest.Path.MilestoneID = objectID
			case "files":
				commentCreateRequest.Path.FileVersionID = objectID
			case "notebooks":
				commentCreateRequest.Path.NotebookID = objectID
			case "links":
				commentCreateRequest.Path.LinkID = objectID
			default:
				return helpers.NewToolResultTextError("invalid object type: %s", objectType), nil
			}

			comment, err := projects.CommentCreate(ctx, engine, commentCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create comment")
			}
			return helpers.NewToolResultText("Comment created successfully with ID %d", comment.ID), nil
		},
	}
}

// CommentUpdate updates a comment in Teamwork.com.
func CommentUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentUpdate),
			Description: "Update comment.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Comment",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to update.",
					},
					"body": {
						Type:        "string",
						Description: "The content of the comment. The content can be added as text or HTML.",
					},
					"content_type": {
						Description: "The content type of the comment. It can be either 'TEXT' or 'HTML'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"TEXT", "HTML"}},
							{Type: "null"},
						},
					},
					"notify_current_user": {
						Description: "Whether the current user should be notified about the comment change.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"notify": helpers.NotifySchema("Who to notify of the comment change.", true),
				},
				Required: []string{"id", "body"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentUpdateRequest projects.CommentUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentUpdateRequest.Path.ID, "id"),
				helpers.RequiredParam(&commentUpdateRequest.Body, "body"),
				helpers.OptionalPointerParam(&commentUpdateRequest.ContentType, "content_type"),
				helpers.OptionalPointerParam(&commentUpdateRequest.NotifyCurrentUser, "notify_current_user"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			notifyChosen, notifiers, toolResult := parseNotify(arguments, true)
			if toolResult != nil {
				return toolResult, nil
			}
			switch notifyChosen {
			case notifyChoiceAll:
				commentUpdateRequest.Notify = projects.NewCommentNotifyAll()
			case notifyChoiceGroup:
				commentUpdateRequest.Notify = projects.NewCommentNotifyGroup(*notifiers)
			case notifyChoiceNone:
				// leave Notify unset: the API sends no notifications
			default:
				commentUpdateRequest.Notify = projects.NewCommentNotifyFollowers()
			}

			_, err = projects.CommentUpdate(ctx, engine, commentUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update comment")
			}
			return helpers.NewToolResultText("Comment updated successfully"), nil
		},
	}
}

// CommentDelete deletes a comment in Teamwork.com.
func CommentDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentDelete),
			Description: "Delete comment.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Comment",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentDeleteRequest projects.CommentDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CommentDelete(ctx, engine, commentDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete comment")
			}
			return helpers.NewToolResultText("Comment deleted successfully"), nil
		},
	}
}

// CommentGet retrieves a comment in Teamwork.com.
func CommentGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCommentGet),
			Description: "Get comment.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Comment",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the comment to get.",
					},
					"fields": helpers.FieldsSchema("comment"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(commentGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentGetRequest projects.CommentGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&commentGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.Comment](&commentGetRequest.Fields.Comment, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(commentGetRequest.Fields.Comment) > 0 {
				return helpers.NewRawToolResult(ctx, engine, commentGetRequest, "failed to get comment",
					commentPathBuilder,
				)
			}

			comment, err := projects.CommentGet(ctx, engine, commentGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get comment")
			}

			encoded, err := json.Marshal(comment)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, commentPathBuilder)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, comment, commentPathBuilder),
			}, nil
		},
	}
}

// CommentList lists comments in Teamwork.com.
func CommentList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCommentList),
			Description: "List comments. Scope by one of task_id, milestone_id, notebook_id, link_id, or file_version_id; " +
				"omit all for site-wide. Comment bodies are truncated at " + strconv.Itoa(commentBodyLimit) +
				" characters and marked where they are cut; use " + string(MethodCommentGet) + " for the full text.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Comments",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Description: "The ID of the task to retrieve comments for. Provide this to scope comments to a task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"milestone_id": {
						Description: "The ID of the milestone to retrieve comments for. Provide this to scope comments to a milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"notebook_id": {
						Description: "The ID of the notebook to retrieve comments for. Provide this to scope comments to a notebook.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"link_id": {
						Description: "The ID of the link to retrieve comments for. Provide this to scope comments to a link.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"user_ids": {
						Description: "A list of user IDs to filter comments by",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"file_version_id": {
						Description: "The ID of the file version to retrieve comments for. Each file can have multiple versions, " +
							"and comments can be associated with specific versions.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"search_term": helpers.SearchTermSchema("comments", "name"),
					"updated_after": helpers.DateTimeFilterSchema(
						"Filter comments updated after. Defaults to the last 3 months.",
					),
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
					"fields":    helpers.FieldsSchema("comment"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(commentListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var commentListRequest projects.CommentListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&commentListRequest.Path.TaskID, "task_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.MilestoneID, "milestone_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.NotebookID, "notebook_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.LinkID, "link_id"),
				helpers.OptionalNumericParam(&commentListRequest.Path.FileVersionID, "file_version_id"),
				helpers.OptionalParam(&commentListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalTimeParam(&commentListRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&commentListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalFieldsParam[projects.Comment](&commentListRequest.Filters.Fields.Comments, "fields"),
				helpers.OptionalNumericListParam(&commentListRequest.Filters.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if commentListRequest.Filters.UpdatedAfter.IsZero() {
				// default to last 3 months to improve performance
				commentListRequest.Filters.UpdatedAfter = time.Now().AddDate(0, -3, 0)
			}

			switch {
			case len(commentListRequest.Filters.Fields.Comments) > 0:
				// an explicit selection wins over both defaults below
			case !verbose:
				commentListRequest.Filters.Fields.Comments = []projects.CommentField{
					projects.CommentFieldID,
				}
			default:
				commentListRequest.Filters.Fields.Comments = commentListFields
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, commentListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list comments")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list comments"), "failed to list comments")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, truncateCommentBodies(body), commentPathBuilder)
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}

// truncateCommentBodies caps the body of every comment in a raw list_comments
// response at commentBodyLimit characters, appending a marker naming how much
// was cut and how to get the rest.
//
// The marker is not decoration. Silent truncation is worse than none: a caller
// handed half a comment with no sign of it reasons over the half confidently.
// It goes inline at the point the text stops rather than into a sibling field
// so it is read as part of the content, survives any later reshaping of the
// record, and needs no change to the published schema — body is still a string.
//
// It applies whether or not the caller named `fields`: an explicit selection of
// body across a few hundred records is exactly the payload this exists to
// bound, and get_comment is the way to full text either way.
//
// Anything unexpected leaves the payload untouched, matching how WebLinker
// treats a body it cannot parse: returning the response whole is always
// preferable to failing a read the API already answered.
func truncateCommentBodies(data []byte) []byte {
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return data
	}
	comments, ok := decoded["comments"].([]any)
	if !ok {
		return data
	}

	var truncated bool
	for _, item := range comments {
		comment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		body, ok := comment["body"].(string)
		if !ok {
			continue
		}
		short, ok := truncateCommentBody(body, comment["id"])
		if !ok {
			continue
		}
		comment["body"] = short
		truncated = true
	}
	if !truncated {
		return data
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		return data
	}
	return encoded
}

// truncateCommentBody shortens a single body, reporting whether it had to. The
// marker carries all three things a caller needs at the point of the cut: that
// the text stops early, how much text there is in total, and the call that
// returns the rest. A bare "truncated" would not distinguish 520 characters
// missing from 128,927, which are very different decisions.
func truncateCommentBody(body string, id any) (string, bool) {
	// Byte length is never below rune count, so this rejects the common case
	// without allocating.
	if len(body) <= commentBodyLimit {
		return body, false
	}
	runes := []rune(body)
	if len(runes) <= commentBodyLimit {
		return body, false
	}

	var marker strings.Builder
	fmt.Fprintf(&marker, "...[truncated — %s chars total", formatThousands(len(runes)))
	if commentID := formatCommentID(id); commentID != "" {
		fmt.Fprintf(&marker, ", %s(id=%s) for full text", MethodCommentGet, commentID)
	}
	marker.WriteString("]")

	return string(runes[:commentBodyLimit]) + marker.String(), true
}

// formatCommentID renders the id of a decoded comment for the truncation
// marker, or an empty string when there is nothing addressable to point at. A
// caller can exclude id from `fields`, but the handler appends it to any
// selection, so in practice this only gives up on a malformed record.
func formatCommentID(id any) string {
	switch value := id.(type) {
	case float64:
		if math.Trunc(value) == value {
			return strconv.FormatInt(int64(value), 10)
		}
	case string:
		return value
	}
	return ""
}

// formatThousands renders a non-negative count with thousands separators, so
// the scale of what is missing reads at a glance rather than by counting
// digits.
func formatThousands(n int) string {
	digits := strconv.Itoa(n)
	if len(digits) <= 3 {
		return digits
	}
	var formatted strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			formatted.WriteRune(',')
		}
		formatted.WriteRune(digit)
	}
	return formatted.String()
}

func commentPathBuilder(object map[string]any) string {
	id := object["id"]
	var relatedObjectType, relatedObjectID any
	if relatedObject, ok := object["object"]; ok {
		if relatedMap, ok := relatedObject.(map[string]any); ok {
			relatedObjectType = relatedMap["type"]
			relatedObjectID = relatedMap["id"]
		}
	}
	if id == nil || relatedObjectType == nil {
		return ""
	}
	if id == reflect.Zero(reflect.TypeOf(id)).Interface() {
		return ""
	}
	if numeric, ok := id.(float64); ok && math.Trunc(numeric) == numeric {
		id = int64(numeric)
	}
	if relatedObjectType == reflect.Zero(reflect.TypeOf(relatedObjectType)).Interface() {
		return ""
	}
	if relatedObjectID == reflect.Zero(reflect.TypeOf(relatedObjectID)).Interface() {
		return ""
	}
	if numeric, ok := relatedObjectID.(float64); ok && math.Trunc(numeric) == numeric {
		relatedObjectID = int64(numeric)
	}
	return fmt.Sprintf("/#%v/%v?c=%v", relatedObjectType, relatedObjectID, id)
}
