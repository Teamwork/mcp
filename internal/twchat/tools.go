// Package twchat exposes a small set of Teamwork Chat API endpoints as MCP
// tools. The Chat API is served at /chat/v7/... on the same installation host
// as the Projects API and authenticates with the same bearer token, so these
// tools reuse the shared twapi.Engine instead of a dedicated SDK.
package twchat

import (
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
)

const chatDescription = "Read conversations, messages, and people, and send messages in Teamwork Chat."

// Sub-toolset key for twchat. This is the valid value for the -toolsets flag
// when selecting Teamwork Chat functionality.
const (
	// ToolsetChat covers reading conversations/messages/people and sending messages.
	ToolsetChat toolsets.Method = "twchat-chat"
)

// Tool method names as exposed to MCP clients.
const (
	// MethodCurrentUserGet retrieves the current authenticated chat user.
	MethodCurrentUserGet toolsets.Method = "twchat-get_current_user"
	// MethodConversationList lists conversations for the current user.
	MethodConversationList toolsets.Method = "twchat-list_conversations"
	// MethodConversationGet retrieves a single conversation.
	MethodConversationGet toolsets.Method = "twchat-get_conversation"
	// MethodMessageList lists messages within a conversation.
	MethodMessageList toolsets.Method = "twchat-list_messages"
	// MethodPeopleList lists people in the installation.
	MethodPeopleList toolsets.Method = "twchat-list_people"
	// MethodMessageSend posts a message to a conversation.
	MethodMessageSend toolsets.Method = "twchat-send_message"
)

func init() {
	toolsets.RegisterMethod(ToolsetChat)
}

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Chat. Write
// tools (send_message) are skipped automatically when readOnly is true.
func DefaultToolsetGroup(readOnly bool, engine *twapi.Engine) *toolsets.ToolsetGroup {
	group := toolsets.NewToolsetGroup(readOnly)

	group.AddToolset(toolsets.NewToolset(ToolsetChat, chatDescription).
		AddWriteTools(
			MessageSend(engine),
		).
		AddReadTools(
			CurrentUserGet(engine),
			ConversationList(engine),
			ConversationGet(engine),
			MessageList(engine),
			PeopleList(engine),
		))

	return group
}
