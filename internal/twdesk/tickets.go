package twdesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
	MethodTicketCreate toolsets.Method = "twdesk-create_ticket"
	MethodTicketUpdate toolsets.Method = "twdesk-update_ticket"
	MethodTicketGet    toolsets.Method = "twdesk-get_ticket"
	MethodTicketList   toolsets.Method = "twdesk-list_tickets"
	MethodTicketSearch toolsets.Method = "twdesk-search_tickets"
)

func init() {
	toolsets.RegisterMethod(MethodTicketCreate)
	toolsets.RegisterMethod(MethodTicketUpdate)
	toolsets.RegisterMethod(MethodTicketGet)
	toolsets.RegisterMethod(MethodTicketList)
	toolsets.RegisterMethod(MethodTicketSearch)
}

// TicketGet finds a ticket in Teamwork Desk.  This will find it by ID
func TicketGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Ticket",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific ticket in Teamwork Desk by its ID. " +
				"Useful for auditing ticket records, troubleshooting support workflows, or " +
				"integrating Desk ticket data into automation and reporting systems.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			ticket, err := client.Tickets.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get ticket: %w", err)
			}

			encoded, err := json.Marshal(ticket)
			if err != nil {
				return nil, err
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/desk/tickets"),
						)),
					},
				},
				StructuredContent: ticket,
			}, nil
		},
	}
}

// TicketList returns a list of tickets that apply to the filters in Teamwork Desk
func TicketList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"inboxIDs": {
			Type: "array",
			Description: `
				The IDs of the inboxes to filter by.
				Inbox IDs can be found by using the 'twdesk-list_inboxes' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"customerIDs": {
			Type: "array",
			Description: `
			The IDs of the customers to filter by. 
			Customer IDs can be found by using the 'twdesk-list_customers' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"companyIDs": {
			Type: "array",
			Description: `
			The IDs of the companies to filter by. 
			Company IDs can be found by using the 'twdesk-list_companies' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"tagIDs": {
			Type: "array",
			Description: `
			The IDs of the tags to filter by. 
			Tag IDs can be found by using the 'twdesk-list_tags' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"taskIDs": {
			Type: "array",
			Description: `
				The IDs of the tasks to filter by.
				Task IDs can be found by using the 'twprojects-list_tasks' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"projectsIDs": {
			Type: "array",
			Description: `
				The IDs of the projects to filter by.
				Project IDs can be found by using the 'twprojects-list_projects' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"statusIDs": {
			Type: "array",
			Description: `
				The IDs of the statuses to filter by.
				Status IDs can be found by using the 'twdesk-list_statuses' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"priorityIDs": {
			Type: "array",
			Description: `
				The IDs of the priorities to filter by.
				Priority IDs can be found by using the 'twdesk-list_priorities' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"slaIDs": {
			Type: "array",
			Description: `
				The IDs of the SLAs to filter by.
				SLA IDs can be found by using the 'twdesk-list_slas' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"userIDs": {
			Type: "array",
			Description: `
				The IDs of the users to filter by.
				User IDs can be found by using the 'twdesk-list_users' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"shared": {
			Type: "boolean",
			Description: `
			Find tickets shared with me outside of inboxes I have access to
		`,
		},
		"slaBreached": {
			Type: "boolean",
			Description: `
			Find tickets where the SLA has been breached
		`,
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tickets",
				ReadOnlyHint: true,
			},
			Description: "List all tickets in Teamwork Desk, with extensive filters for inbox, customer, company, " +
				"tag, status, priority, SLA, user, and more. Enables users to audit, analyze, or synchronize ticket data " +
				"for support management, reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			// Apply filters to the ticket list
			inboxIDs := arguments.GetIntSlice("inboxIDs", []int{})
			customerIDs := arguments.GetIntSlice("customerIDs", []int{})
			companyIDs := arguments.GetIntSlice("companyIDs", []int{})
			tagIDs := arguments.GetIntSlice("tagIDs", []int{})
			taskIDs := arguments.GetIntSlice("taskIDs", []int{})
			projectsIDs := arguments.GetIntSlice("projectsIDs", []int{})
			statusIDs := arguments.GetIntSlice("statusIDs", []int{})
			priorityIDs := arguments.GetIntSlice("priorityIDs", []int{})
			slaIDs := arguments.GetIntSlice("slaIDs", []int{})
			userIDs := arguments.GetIntSlice("userIDs", []int{})
			shared := arguments.GetBool("shared", false)
			slaBreached := arguments.GetBool("slaBreached", false)

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
			setPagination(&params, arguments)

			tickets, err := client.Tickets.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list tickets: %w", err)
			}
			return helpers.NewToolResultJSON(tickets)
		},
	}
}

// TicketSearch uses the search API to find tickets in Teamwork Desk
func TicketSearch(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"search": {
			Type: "string",
			Description: `
				The search term to use for finding tickets.
				This can be part of the subject, body, or other ticket fields.
			`,
		},
		"inboxIDs": {
			Type: "array",
			Description: `
				The IDs of the inboxes to filter by.
				Inbox IDs can be found by using the 'twdesk-list_inboxes' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"customerIDs": {
			Type: "array",
			Description: `
			The IDs of the customers to filter by. 
			Customer IDs can be found by using the 'twdesk-list_customers' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"companyIDs": {
			Type: "array",
			Description: `
			The IDs of the companies to filter by. 
			Company IDs can be found by using the 'twdesk-list_companies' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"tagIDs": {
			Type: "array",
			Description: `
			The IDs of the tags to filter by. 
			Tag IDs can be found by using the 'twdesk-list_tags' tool.
		`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"statusIDs": {
			Type: "array",
			Description: `
				The IDs of the statuses to filter by.
				Status IDs can be found by using the 'twdesk-list_statuses' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"priorityIDs": {
			Type: "array",
			Description: `
				The IDs of the priorities to filter by.
				Priority IDs can be found by using the 'twdesk-list_priorities' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"userIDs": {
			Type: "array",
			Description: `
				The IDs of the users to filter by.
				User IDs can be found by using the 'twdesk-list_users' tool.
			`,
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"shared": {
			Type: "boolean",
			Description: `
			Find tickets shared with me outside of inboxes I have access to
		`,
		},
		"slaBreached": {
			Type: "boolean",
			Description: `
			Find tickets where the SLA has been breached
		`,
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketSearch),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Search Tickets",
				ReadOnlyHint: true,
			},
			Description: "Search tickets in Teamwork Desk using various filters including inbox, customer, company, " +
				"tag, status, priority, SLA, user, and more. This tool enables users to perform targeted searches " +
				"for tickets, facilitating efficient support management, reporting, and integration with other systems.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
				Required:   []string{"search"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			params := &deskmodels.SearchTicketsFilter{}

			params.Search = arguments.GetString("search", "")

			if arguments.GetIntSlice("customerIDs", nil) != nil {
				params.Customers = helpers.IntSliceToInt64(arguments.GetIntSlice("customerIDs", nil))
			}
			if arguments.GetIntSlice("companyIDs", nil) != nil {
				params.Companies = helpers.IntSliceToInt64(arguments.GetIntSlice("companyIDs", nil))
			}
			if arguments.GetIntSlice("tagIDs", nil) != nil {
				params.Tags = helpers.IntSliceToInt64(arguments.GetIntSlice("tagIDs", nil))
			}
			if arguments.GetIntSlice("statusIDs", nil) != nil {
				params.Statuses = helpers.IntSliceToInt64(arguments.GetIntSlice("statusIDs", nil))
			}
			if arguments.GetIntSlice("priorityIDs", nil) != nil {
				params.Priorities = helpers.IntSliceToInt64(arguments.GetIntSlice("priorityIDs", nil))
			}
			if arguments.GetIntSlice("userIDs", nil) != nil {
				params.Agents = helpers.IntSliceToInt64(arguments.GetIntSlice("userIDs", nil))
			}

			tickets, err := client.Tickets.Search(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list tickets: %w", err)
			}
			return helpers.NewToolResultJSON(tickets)
		},
	}
}

// TicketCreate creates a ticket in Teamwork Desk
func TicketCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Ticket",
			},
			Description: `
				Create a new ticket in Teamwork Desk by specifying subject, description, priority, and status.
				"Useful for automating ticket creation, integrating external systems, or customizing support workflows.
			`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"subject": {
						Type:        "string",
						Description: "The subject of the ticket.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the ticket.",
					},
					"notifyCustomer": {
						Type:        "boolean",
						Description: "Set to true if the the customer should be sent a copy of the ticket.",
					},
					"bcc": {
						Type:        "array",
						Description: "An array of email addresses to BCC on ticket creation.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
					"cc": {
						Type:        "array",
						Description: "An array of email addresses to CC on ticket creation.",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
					"files": {
						Type: "array",
						Description: `
					An array of file IDs to attach to the ticket.  
					Use the 'twdesk-create_file' tool to upload files.
				`,
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"tags": {
						Type: "array",
						Description: `
					An array of tag IDs to associate with the ticket. 
					Tag IDs can be found by using the 'twdesk-list_tags' tool.
				`,
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"priorityId": {
						Type: "integer",
						Description: `
					The priority of the ticket. 
					Use the 'twdesk-list_priorities' tool to find valid IDs.
				`,
					},
					"statusId": {
						Type: "integer",
						Description: `
					The status of the ticket. 
					Use the 'twdesk-list_statuses' tool to find valid IDs.
				`,
					},
					"inboxId": {
						Type: "integer",
						Description: `
					The inbox ID of the ticket. 
					Use the 'twdesk-list_inboxes' tool to find valid IDs.
				`,
					},
					"customerId": {
						Type: "integer",
						Description: `
					The customer ID of the ticket. 
					Use the 'twdesk-list_customers' tool to find valid IDs.
				`,
					},
					"customerEmail": {
						Type: "string",
						Description: `
				The email address of the customer. 
				This is used to identify the customer in the system.
				Either the customerId or customerEmail is required to create a ticket.  
				If email is provided we will either find or create the customer.
			`,
					},
					"typeId": {
						Type: "integer",
						Description: `
					The type ID of the ticket. 
					Use the 'twdesk-list_types' tool to find valid IDs.
				`,
					},
					"agentId": {
						Type: "integer",
						Description: `
					The agent ID that the ticket should be assigned to. 
					Use the 'twdesk-list_agents' tool to find valid IDs.
				`,
					},
				},
				Required: []string{"subject", "body", "inboxId"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			data := deskmodels.Ticket{
				Subject: arguments.GetString("subject", ""),
				Body:    arguments.GetString("body", ""),
				Inbox: deskmodels.EntityRef{
					ID: arguments.GetInt("inboxId", 0),
				},
			}

			if arguments.GetInt("customerId", 0) != 0 {
				data.Customer = deskmodels.EntityRef{
					ID: arguments.GetInt("customerId", 0),
				}
			}

			if email := arguments.GetString("customerEmail", ""); email != "" {
				filter := deskclient.NewFilter()
				filter = filter.Eq("contacts.value", email)

				params := url.Values{}
				params.Set("filter", filter.Build())
				setPagination(&params, arguments)

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

			if arguments.GetInt("priorityId", 0) != 0 {
				data.Priority = &deskmodels.EntityRef{
					ID: arguments.GetInt("priorityId", 0),
				}
			}

			if arguments.GetInt("statusId", 0) != 0 {
				data.Status = &deskmodels.EntityRef{
					ID: arguments.GetInt("statusId", 0),
				}
			}

			if arguments.GetInt("typeId", 0) != 0 {
				data.Type = &deskmodels.EntityRef{
					ID: arguments.GetInt("typeId", 0),
				}
			}

			if arguments.GetInt("agentId", 0) != 0 {
				data.Agent = &deskmodels.EntityRef{
					ID: arguments.GetInt("agentId", 0),
				}
			}

			if arguments.GetBool("notifyCustomer", false) {
				data.NotifyCustomer = true
			}

			if len(arguments.GetIntSlice("files", []int{})) > 0 {
				data.Files = []deskmodels.EntityRef{}
				for _, fileID := range arguments.GetIntSlice("files", []int{}) {
					data.Files = append(data.Files, deskmodels.EntityRef{ID: fileID})
				}
			}

			if len(arguments.GetIntSlice("tags", []int{})) > 0 {
				data.Tags = []deskmodels.EntityRef{}
				for _, tagID := range arguments.GetIntSlice("tags", []int{}) {
					data.Tags = append(data.Tags, deskmodels.EntityRef{ID: tagID})
				}
			}

			if len(arguments.GetStringSlice("bcc", []string{})) > 0 {
				data.BCC = arguments.GetStringSlice("bcc", []string{})
			}

			if len(arguments.GetStringSlice("cc", []string{})) > 0 {
				data.CC = arguments.GetStringSlice("cc", []string{})
			}

			ticket, err := client.Tickets.Create(ctx, &deskmodels.TicketResponse{
				Ticket: data,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create ticket: %w", err)
			}
			return helpers.NewToolResultJSON(ticket)
		},
	}
}

// TicketUpdate updates a ticket in Teamwork Desk
func TicketUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Ticket",
			},
			Description: "Update an existing ticket in Teamwork Desk by ID, allowing changes to its attributes. " +
				"Supports evolving support processes, correcting ticket records, or integrating with automation " +
				"systems for improved ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket to update.",
					},
					"subject": {
						Type:        "string",
						Description: "The subject of the ticket.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the ticket.",
					},
					"priorityId": {
						Type: "integer",
						Description: `
					The priority of the ticket. 
					Use the 'twdesk-list_priorities' tool to find valid IDs.
				`,
					},
					"statusId": {
						Type: "integer",
						Description: `
					The status of the ticket. 
					Use the 'twdesk-list_statuses' tool to find valid IDs.
				`,
					},
					"typeId": {
						Type: "integer",
						Description: `
					The type ID of the ticket. 
					Use the 'twdesk-list_types' tool to find valid IDs.
				`,
					},
					"agentId": {
						Type: "integer",
						Description: `
					The agent ID that the ticket should be assigned to. 
					Use the 'twdesk-list_agents' tool to find valid IDs.
				`,
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			data := deskmodels.Ticket{}

			if subject := arguments.GetString("subject", ""); subject != "" {
				data.Subject = subject
			}

			if body := arguments.GetString("body", ""); body != "" {
				data.Body = body
			}

			if statusId := arguments.GetInt("statusId", 0); statusId > 0 {
				data.Status = &deskmodels.EntityRef{ID: statusId}
			}

			if typeId := arguments.GetInt("typeId", 0); typeId > 0 {
				data.Type = &deskmodels.EntityRef{ID: typeId}
			}

			if agentId := arguments.GetInt("agentId", 0); agentId > 0 {
				data.Agent = &deskmodels.EntityRef{ID: agentId}
			}

			ticket, err := client.Tickets.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketResponse{
				Ticket: data,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update ticket: %w", err)
			}
			return helpers.NewToolResultJSON(ticket)
		},
	}
}
