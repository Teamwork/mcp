package twdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
	MethodTicketCreate toolsets.Method = "twdesk-create_ticket"
	MethodTicketUpdate toolsets.Method = "twdesk-update_ticket"
	MethodTicketGet    toolsets.Method = "twdesk-get_ticket"
	MethodTicketList   toolsets.Method = "twdesk-list_tickets"
)

func init() {
	toolsets.RegisterMethod(MethodTicketCreate)
	toolsets.RegisterMethod(MethodTicketUpdate)
	toolsets.RegisterMethod(MethodTicketGet)
	toolsets.RegisterMethod(MethodTicketList)
}

// TicketGet finds a ticket in Teamwork Desk.  This will find it by ID
func TicketGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTicketGet),
			mcp.WithDescription(
				"Retrieve detailed information about a specific ticket in Teamwork Desk by its ID. "+
					"Useful for auditing ticket records, troubleshooting support workflows, or "+
					"integrating Desk ticket data into automation and reporting systems."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the ticket to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticket, err := client.Tickets.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get ticket: %w", err)
			}

			encoded, err := json.Marshal(ticket)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/desk/tickets"),
			))), nil
		},
	}
}

// TicketList returns a list of tickets that apply to the filters in Teamwork Desk
func TicketList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithOutputSchema[deskmodels.TicketsResponse](),
		mcp.WithDescription(
			"List all tickets in Teamwork Desk, with extensive filters for inbox, customer, company, tag, status, " +
				"priority, SLA, user, and more. Enables users to audit, analyze, or synchronize ticket data for support " +
				"management, reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("inboxIDs", mcp.Description(`
			The IDs of the inboxes to filter by.  
			Inbox IDs can be found by using the 'twdesk-list_inboxes' tool.
		`)),
		mcp.WithArray("customerIDs", mcp.Description(`
			The IDs of the customers to filter by. 
			Customer IDs can be found by using the 'twdesk-list_customers' tool.
		`)),
		mcp.WithArray("companyIDs", mcp.Description(`
			The IDs of the companies to filter by. 
			Company IDs can be found by using the 'twdesk-list_companies' tool.
		`)),
		mcp.WithArray("tagIDs", mcp.Description(`
			The IDs of the tags to filter by. 
			Tag IDs can be found by using the 'twdesk-list_tags' tool.
		`)),
		mcp.WithArray("taskIDs", mcp.Description(`
			The IDs of the tasks to filter by. 
			Task IDs can be found by using the 'twprojects-list_tasks' tool.
		`)),
		mcp.WithArray("projectsIDs", mcp.Description(`
			The IDs of the projects to filter by. 
			Project IDs can be found by using the 'twprojects-list_projects' tool.
		`)),
		mcp.WithArray("statusIDs", mcp.Description(`
			The IDs of the statuses to filter by. 
			Status IDs can be found by using the 'twdesk-list_statuses' tool.
		`)),
		mcp.WithArray("priorityIDs", mcp.Description(`
			The IDs of the priorities to filter by. 
			Priority IDs can be found by using the 'twdesk-list_priorities' tool.
		`)),
		mcp.WithArray("slaIDs", mcp.Description(`
			The IDs of the SLAs to filter by. 
			SLA IDs can be found by using the 'twdesk-list_slas' tool.
		`)),
		mcp.WithArray("userIDs", mcp.Description(`
			The IDs of the users to filter by. 
			User IDs can be found by using the 'twdesk-list_users' tool.
		`)),
		mcp.WithBoolean("shared", mcp.Description(`
			Find tickets shared with me outside of inboxes I have access to
		`)),
		mcp.WithBoolean("slaBreached", mcp.Description(`
			Find tickets where the SLA has been breached
		`)),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTicketList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the ticket list
			inboxIDs := request.GetIntSlice("inboxIDs", []int{})
			customerIDs := request.GetIntSlice("customerIDs", []int{})
			companyIDs := request.GetIntSlice("companyIDs", []int{})
			tagIDs := request.GetIntSlice("tagIDs", []int{})
			taskIDs := request.GetIntSlice("taskIDs", []int{})
			projectsIDs := request.GetIntSlice("projectsIDs", []int{})
			statusIDs := request.GetIntSlice("statusIDs", []int{})
			priorityIDs := request.GetIntSlice("priorityIDs", []int{})
			slaIDs := request.GetIntSlice("slaIDs", []int{})
			userIDs := request.GetIntSlice("userIDs", []int{})
			shared := request.GetBool("shared", false)
			slaBreached := request.GetBool("slaBreached", false)

			filter := deskclient.NewFilter()

			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", helpers.SliceToAny(inboxIDs))
			}

			if len(customerIDs) > 0 {
				filter = filter.In("customers.id", helpers.SliceToAny(customerIDs))
			}

			if len(companyIDs) > 0 {
				filter = filter.In("companies.id", helpers.SliceToAny(companyIDs))
			}

			if len(tagIDs) > 0 {
				filter = filter.In("tags.id", helpers.SliceToAny(tagIDs))
			}

			if len(taskIDs) > 0 {
				filter = filter.In("tasks.id", helpers.SliceToAny(taskIDs))
			}

			if len(projectsIDs) > 0 {
				filter = filter.In("projects.id", helpers.SliceToAny(projectsIDs))
			}

			if len(statusIDs) > 0 {
				filter = filter.In("statuses.id", helpers.SliceToAny(statusIDs))
			}

			if len(priorityIDs) > 0 {
				filter = filter.In("priorities.id", helpers.SliceToAny(priorityIDs))
			}

			if len(slaIDs) > 0 {
				filter = filter.In("slas.id", helpers.SliceToAny(slaIDs))
			}

			if len(userIDs) > 0 {
				filter = filter.In("users.id", helpers.SliceToAny(userIDs))
			}

			if shared {
				filter = filter.Eq("shared", true)
			}

			if slaBreached {
				filter = filter.Eq("sla_breached", true)
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, request)

			tickets, err := client.Tickets.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list tickets: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tickets retrieved successfully: %v", tickets)), nil
		},
	}
}

// TicketCreate creates a ticket in Teamwork Desk
func TicketCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTicketCreate),
			mcp.WithDescription(
				"Create a new ticket in Teamwork Desk by specifying subject, description, priority, and status. "+
					"Useful for automating ticket creation, integrating external systems, or customizing support workflows."),
			mcp.WithString("subject", mcp.Required(), mcp.Description("The subject of the ticket.")),
			mcp.WithString("description", mcp.Description("The description of the ticket.")),
			mcp.WithString("priority", mcp.Description("The priority of the ticket.")),
			mcp.WithString("status", mcp.Description("The status of the ticket.")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticket, err := client.Tickets.Create(ctx, &deskmodels.TicketResponse{
				Ticket: deskmodels.Ticket{
					Subject: request.GetString("subject", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create ticket: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Ticket created successfully with ID %d", ticket.Ticket.ID)), nil
		},
	}
}

// TicketUpdate updates a ticket in Teamwork Desk
func TicketUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTicketUpdate),
			mcp.WithDescription(
				"Update an existing ticket in Teamwork Desk by ID, allowing changes to its attributes. "+
					"Supports evolving support processes, correcting ticket records, or integrating with automation "+
					"systems for improved ticket handling."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the ticket to update."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			_, err := client.Tickets.Update(ctx, request.GetInt("id", 0), &deskmodels.TicketResponse{
				Ticket: deskmodels.Ticket{},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create ticket: %w", err)
			}

			return mcp.NewToolResultText("Ticket updated successfully"), nil
		},
	}
}
