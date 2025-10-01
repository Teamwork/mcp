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
			mcp.WithTitleAnnotation("Get Ticket"),
			mcp.WithOutputSchema[deskmodels.TicketResponse](),
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
		mcp.WithTitleAnnotation("List Tickets"),
		mcp.WithOutputSchema[deskmodels.TicketsResponse](),
		mcp.WithDescription(
			"List all tickets in Teamwork Desk, with extensive filters for inbox, customer, company, tag, status, " +
				"priority, SLA, user, and more. Enables users to audit, analyze, or synchronize ticket data for support " +
				"management, reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("inboxIDs",
			mcp.Description(`
				The IDs of the inboxes to filter by.
				Inbox IDs can be found by using the 'twdesk-list_inboxes' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("customerIDs", mcp.Description(`
			The IDs of the customers to filter by. 
			Customer IDs can be found by using the 'twdesk-list_customers' tool.
		`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("companyIDs", mcp.Description(`
			The IDs of the companies to filter by. 
			Company IDs can be found by using the 'twdesk-list_companies' tool.
		`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("tagIDs", mcp.Description(`
			The IDs of the tags to filter by. 
			Tag IDs can be found by using the 'twdesk-list_tags' tool.
		`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("taskIDs",
			mcp.Description(`
				The IDs of the tasks to filter by.
				Task IDs can be found by using the 'twprojects-list_tasks' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("projectsIDs",
			mcp.Description(`
				The IDs of the projects to filter by.
				Project IDs can be found by using the 'twprojects-list_projects' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("statusIDs",
			mcp.Description(`
				The IDs of the statuses to filter by.
				Status IDs can be found by using the 'twdesk-list_statuses' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("priorityIDs",
			mcp.Description(`
				The IDs of the priorities to filter by.
				Priority IDs can be found by using the 'twdesk-list_priorities' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("slaIDs",
			mcp.Description(`
				The IDs of the SLAs to filter by.
				SLA IDs can be found by using the 'twdesk-list_slas' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
		mcp.WithArray("userIDs",
			mcp.Description(`
				The IDs of the users to filter by.
				User IDs can be found by using the 'twdesk-list_users' tool.
			`),
			mcp.Items(map[string]any{
				"type": "integer",
			}),
		),
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
			mcp.WithTitleAnnotation("Create Ticket"),
			mcp.WithDescription(
				"Create a new ticket in Teamwork Desk by specifying subject, description, priority, and status. "+
					"Useful for automating ticket creation, integrating external systems, or customizing support workflows."),
			mcp.WithString("subject",
				mcp.Required(),
				mcp.Description("The subject of the ticket."),
			),
			mcp.WithString("body",
				mcp.Required(),
				mcp.Description("The body of the ticket."),
			),
			mcp.WithBoolean("notifyCustomer",
				mcp.Description("Set to true if the the customer should be sent a copy of the ticket."),
			),
			mcp.WithArray("bcc",
				mcp.Description("An array of email addresses to BCC on ticket creation."),
				mcp.Items(map[string]any{
					"type": "string",
				}),
			),
			mcp.WithArray("cc",
				mcp.Description("An array of email addresses to CC on ticket creation."),
				mcp.Items(map[string]any{
					"type": "string",
				}),
			),
			mcp.WithArray("files",
				mcp.Description(`
					An array of file IDs to attach to the ticket.  
					Use the 'twdesk-create_file' tool to upload files.
				`),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("tags",
				mcp.Description(`
					An array of tag IDs to associate with the ticket. 
					Tag IDs can be found by using the 'twdesk-list_tags' tool.
				`),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithNumber("priorityId",
				mcp.Description(`
					The priority of the ticket. 
					Use the 'twdesk-list_priorities' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("statusId",
				mcp.Description(`
					The status of the ticket. 
					Use the 'twdesk-list_statuses' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("inboxId",
				mcp.Required(),
				mcp.Description(`
					The inbox ID of the ticket. 
					Use the 'twdesk-list_inboxes' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("customerId",
				mcp.Description(`
					The customer ID of the ticket. 
					Use the 'twdesk-list_customers' tool to find valid IDs.
				`),
			),
			mcp.WithString("customerEmail",
				mcp.Description(`
				The email address of the customer. 
				This is used to identify the customer in the system.
				Either the customerId or customerEmail is required to create a ticket.  
				If email is provided we will either find or create the customer.
			`),
			),
			mcp.WithNumber("typeId",
				mcp.Description(`
					The type ID of the ticket. 
					Use the 'twdesk-list_types' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("agentId",
				mcp.Description(`
					The agent ID that the ticket should be assigned to. 
					Use the 'twdesk-list_agents' tool to find valid IDs.
				`),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := deskmodels.Ticket{
				Subject: request.GetString("subject", ""),
				Body:    request.GetString("body", ""),
				Inbox: deskmodels.EntityRef{
					ID: request.GetInt("inboxId", 0),
				},
			}

			if request.GetInt("customerId", 0) != 0 {
				data.Customer = deskmodels.EntityRef{
					ID: request.GetInt("customerId", 0),
				}
			}

			if email := request.GetString("customerEmail", ""); email != "" {
				filter := deskclient.NewFilter()
				filter = filter.Eq("contacts.value", email)

				params := url.Values{}
				params.Set("filter", filter.Build())
				setPagination(&params, request)

				customers, err := client.Customers.List(ctx, params)
				if err != nil {
					return nil, fmt.Errorf("failed to list customers: %w", err)
				}

				if len(customers.Customers) > 0 {
					data.Customer = deskmodels.EntityRef{
						ID: customers.Customers[0].ID,
					}
				} else {
					// Create the customer
					customer, err := client.Customers.Create(ctx, &deskmodels.CustomerResponse{
						Customer: deskmodels.Customer{
							Email: email,
						},
					})
					if err != nil {
						return nil, fmt.Errorf("failed to create customer: %w", err)
					}
					data.Customer = deskmodels.EntityRef{
						ID: customer.Customer.ID,
					}
				}
			}

			if request.GetInt("priorityId", 0) != 0 {
				data.Priority = &deskmodels.EntityRef{
					ID: request.GetInt("priorityId", 0),
				}
			}

			if request.GetInt("statusId", 0) != 0 {
				data.Status = &deskmodels.EntityRef{
					ID: request.GetInt("statusId", 0),
				}
			}

			if request.GetInt("typeId", 0) != 0 {
				data.Type = &deskmodels.EntityRef{
					ID: request.GetInt("typeId", 0),
				}
			}

			if request.GetInt("agentId", 0) != 0 {
				data.Agent = &deskmodels.EntityRef{
					ID: request.GetInt("agentId", 0),
				}
			}

			if request.GetBool("notifyCustomer", false) {
				data.NotifyCustomer = true
			}

			if len(request.GetIntSlice("files", []int{})) > 0 {
				data.Files = []deskmodels.EntityRef{}
				for _, fileID := range request.GetIntSlice("files", []int{}) {
					data.Files = append(data.Files, deskmodels.EntityRef{ID: fileID})
				}
			}

			if len(request.GetIntSlice("tags", []int{})) > 0 {
				data.Tags = []deskmodels.EntityRef{}
				for _, tagID := range request.GetIntSlice("tags", []int{}) {
					data.Tags = append(data.Tags, deskmodels.EntityRef{ID: tagID})
				}
			}

			if len(request.GetStringSlice("bcc", []string{})) > 0 {
				data.BCC = request.GetStringSlice("bcc", []string{})
			}

			if len(request.GetStringSlice("cc", []string{})) > 0 {
				data.CC = request.GetStringSlice("cc", []string{})
			}

			ticket, err := client.Tickets.Create(ctx, &deskmodels.TicketResponse{
				Ticket: data,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create ticket: %w", err)
			}

			return mcp.NewToolResultJSON(ticket)
		},
	}
}

// TicketUpdate updates a ticket in Teamwork Desk
func TicketUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTicketUpdate),
			mcp.WithTitleAnnotation("Update Ticket"),
			mcp.WithDescription(
				"Update an existing ticket in Teamwork Desk by ID, allowing changes to its attributes. "+
					"Supports evolving support processes, correcting ticket records, or integrating with automation "+
					"systems for improved ticket handling."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the ticket to update."),
			),
			mcp.WithString("subject",
				mcp.Description("The subject of the ticket."),
			),
			mcp.WithString("body",
				mcp.Description("The body of the ticket."),
			),
			mcp.WithNumber("priorityId",
				mcp.Description(`
					The priority of the ticket. 
					Use the 'twdesk-list_priorities' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("statusId",
				mcp.Description(`
					The status of the ticket. 
					Use the 'twdesk-list_statuses' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("typeId",
				mcp.Description(`
					The type ID of the ticket. 
					Use the 'twdesk-list_types' tool to find valid IDs.
				`),
			),
			mcp.WithNumber("agentId",
				mcp.Description(`
					The agent ID that the ticket should be assigned to. 
					Use the 'twdesk-list_agents' tool to find valid IDs.
				`),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := deskmodels.Ticket{}

			if subject := request.GetString("subject", ""); subject != "" {
				data.Subject = subject
			}

			if body := request.GetString("body", ""); body != "" {
				data.Body = body
			}

			if statusId := request.GetInt("statusId", 0); statusId > 0 {
				data.Status = &deskmodels.EntityRef{ID: statusId}
			}

			if typeId := request.GetInt("typeId", 0); typeId > 0 {
				data.Type = &deskmodels.EntityRef{ID: typeId}
			}

			if agentId := request.GetInt("agentId", 0); agentId > 0 {
				data.Agent = &deskmodels.EntityRef{ID: agentId}
			}

			ticket, err := client.Tickets.Update(ctx, request.GetInt("id", 0), &deskmodels.TicketResponse{
				Ticket: data,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update ticket: %w", err)
			}

			return mcp.NewToolResultJSON(ticket)
		},
	}
}
