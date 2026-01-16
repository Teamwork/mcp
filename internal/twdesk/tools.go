package twdesk

import (
	"net/http"

	"github.com/teamwork/mcp/internal/toolsets"
)

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Desk.
func DefaultToolsetGroup(httpClient *http.Client) *toolsets.ToolsetGroup {
	readTools := []toolsets.ToolWrapper{
		CompanyGet(httpClient),
		CompanyList(httpClient),
		CustomerGet(httpClient),
		CustomerList(httpClient),
		InboxGet(httpClient),
		InboxList(httpClient),
		PriorityGet(httpClient),
		PriorityList(httpClient),
		StatusGet(httpClient),
		StatusList(httpClient),
		TagGet(httpClient),
		TagList(httpClient),
		TicketGet(httpClient),
		TicketList(httpClient),
		TicketSearch(httpClient),
		TypeGet(httpClient),
		TypeList(httpClient),
		UserGet(httpClient),
		UserList(httpClient),
	}

	writeTools := []toolsets.ToolWrapper{
		CompanyCreate(httpClient),
		CompanyUpdate(httpClient),
		CustomerCreate(httpClient),
		CustomerUpdate(httpClient),
		FileCreate(httpClient),
		MessageCreate(httpClient),
		PriorityCreate(httpClient),
		PriorityUpdate(httpClient),
		StatusCreate(httpClient),
		StatusUpdate(httpClient),
		TagCreate(httpClient),
		TagUpdate(httpClient),
		TicketCreate(httpClient),
		TicketUpdate(httpClient),
		TypeCreate(httpClient),
		TypeUpdate(httpClient),
	}

	group := toolsets.NewToolsetGroup(false)
	group.AddToolset(toolsets.NewToolset("desk", projectDescription).
		AddWriteTools(writeTools...).
		AddReadTools(readTools...))
	return group
}
