package twdesk

import (
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/toolsets"
)

// DefaultToolsetGroup creates a default ToolsetGroup for Teamwork Projects.
func DefaultToolsetGroup(client *deskclient.Client) *toolsets.ToolsetGroup {
	readTools := []toolsets.ToolWrapper{
		CompanyGet(client),
		CompanyList(client),
		CustomerGet(client),
		CustomerList(client),
		InboxGet(client),
		InboxList(client),
		PriorityGet(client),
		PriorityList(client),
		StatusGet(client),
		StatusList(client),
		TagGet(client),
		TagList(client),
		TicketGet(client),
		TicketList(client),
		TicketSearch(client),
		TypeGet(client),
		TypeList(client),
		UserGet(client),
		UserList(client),
	}

	writeTools := []toolsets.ToolWrapper{
		CompanyCreate(client),
		CompanyUpdate(client),
		CustomerCreate(client),
		CustomerUpdate(client),
		FileCreate(client),
		MessageCreate(client),
		PriorityCreate(client),
		PriorityUpdate(client),
		StatusCreate(client),
		StatusUpdate(client),
		TagCreate(client),
		TagUpdate(client),
		TicketCreate(client),
		TicketUpdate(client),
		TypeCreate(client),
		TypeUpdate(client),
	}

	group := toolsets.NewToolsetGroup(false)
	group.AddToolset(toolsets.NewToolset("desk", projectDescription).
		AddWriteTools(writeTools...).
		AddReadTools(readTools...))
	return group
}
