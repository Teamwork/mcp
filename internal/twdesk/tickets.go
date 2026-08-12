package twdesk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sonh/qs"
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
	MethodTicketSearch toolsets.Method = "twdesk-search_tickets"
)

// TicketGet finds a ticket in Teamwork Desk.  This will find it by ID
func TicketGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketGet),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Ticket",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Get ticket.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id", "fields"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			ticket, err := client.Tickets.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get ticket")
			}

			encoded, err := json.Marshal(ticket)
			if err != nil {
				return helpers.NewToolResultTextError("failed to encode ticket: %s", err.Error()), nil
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/desk/tickets"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, ticket,
					helpers.WebLinkerWithIDPathBuilder("/desk/tickets"),
				),
			}, nil
		},
	}
}

// ticketSearchService points a generic Desk service at the ticket search
// endpoint.
//
// client.Tickets.Search cannot be used: it builds its query string solely from
// the qs-encoded SearchTicketsFilter, which has no page, pageSize, ordering or
// sparse-fieldset field. Every call therefore came back as the endpoint's
// default first page of 50 tickets with the full attribute set, no matter what
// the caller asked for, and the response metadata reported those defaults so
// the caller could not tell. Routing through Service.List hits the same
// /search/tickets.json path while letting the handler own the query string.
func ticketSearchService(
	client *deskclient.Client,
) *deskclient.Service[deskmodels.TicketResponse, deskmodels.TicketsResponse] {
	return deskclient.NewService[deskmodels.TicketResponse, deskmodels.TicketsResponse](
		client, deskclient.NewDefaultPathHandler("search/tickets"),
	)
}

// TicketSearch uses the search API to find tickets in Teamwork Desk
func TicketSearch(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"search": {
			Description: "Search term matched against subject, body, and other ticket fields.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "null"},
			},
		},
		"inboxIDs": {
			Description: "Filter by inbox. Use twdesk-list_inboxes to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"customerIDs": {
			Description: "Filter by customer. Use twdesk-list_customers to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"companyIDs": {
			Description: "Filter by company. Use twdesk-list_companies to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"tagIDs": {
			Description: "Filter by tag. Use twdesk-list_tags to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"statusIDs": {
			Description: "Filter by status. Use twdesk-list_statuses to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"priorityIDs": {
			Description: "Filter by priority. Use twdesk-list_priorities to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"userIDs": {
			Description: "Filter by user. Use twdesk-list_users to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
		"createdAfter": helpers.DateTimeFilterSchema(
			"Filter by ticket creation date: only tickets created on or after this day. " +
				"This search filters by whole days, so a time of day is ignored.",
		),
		"createdBefore": helpers.DateTimeFilterSchema(
			"Filter by ticket creation date: only tickets created on or before this day. " +
				"This search filters by whole days, so a time of day is ignored.",
		),
	}
	properties = searchPaginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTicketSearch),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Search Tickets",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Search tickets. Filter by inbox, customer, company, tag, status, priority, user, " +
				"or creation date range.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required: append(paginationRequiredKeys(),
					"search", "inboxIDs", "customerIDs", "companyIDs",
					"tagIDs", "statusIDs", "priorityIDs", "userIDs",
					"createdAfter", "createdBefore",
				),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			filter := &deskmodels.SearchTicketsFilter{}

			filter.Search = arguments.GetString("search", "")

			if arguments.GetIntSlice("inboxIDs", nil) != nil {
				filter.Inboxes = helpers.IntSliceToInt64(arguments.GetIntSlice("inboxIDs", nil))
			}
			if arguments.GetIntSlice("customerIDs", nil) != nil {
				filter.Customers = helpers.IntSliceToInt64(arguments.GetIntSlice("customerIDs", nil))
			}
			if arguments.GetIntSlice("companyIDs", nil) != nil {
				filter.Companies = helpers.IntSliceToInt64(arguments.GetIntSlice("companyIDs", nil))
			}
			if arguments.GetIntSlice("tagIDs", nil) != nil {
				filter.Tags = helpers.IntSliceToInt64(arguments.GetIntSlice("tagIDs", nil))
			}
			if arguments.GetIntSlice("statusIDs", nil) != nil {
				filter.Statuses = helpers.IntSliceToInt64(arguments.GetIntSlice("statusIDs", nil))
			}
			if arguments.GetIntSlice("priorityIDs", nil) != nil {
				filter.Priorities = helpers.IntSliceToInt64(arguments.GetIntSlice("priorityIDs", nil))
			}
			if arguments.GetIntSlice("userIDs", nil) != nil {
				filter.Agents = helpers.IntSliceToInt64(arguments.GetIntSlice("userIDs", nil))
			}

			// The creation-date window is bound here rather than onto
			// filter.StartDate/EndDate so that the value the endpoint receives is
			// not the RFC 3339 one qs renders a time.Time as. See below.
			var createdAfter, createdBefore *time.Time
			err = helpers.ParamGroup(arguments,
				helpers.OptionalTimePointerParam(&createdAfter, "createdAfter"),
				helpers.OptionalTimePointerParam(&createdBefore, "createdBefore"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Encode the filter the same way the SDK's Search does, then add the
			// pagination, ordering and sparse fieldset the filter struct cannot
			// carry. See ticketSearchService.
			params, err := qs.NewEncoder().Values(filter)
			if err != nil {
				return helpers.NewToolResultTextError("failed to encode ticket search filter: %s", err.Error()), nil
			}
			setSearchPagination(&params, arguments)

			if createdAfter != nil {
				params.Set("startDate", createdAfter.Format(time.DateOnly))
			}
			if createdBefore != nil {
				params.Set("endDate", createdBefore.Format(time.DateOnly))
			}

			tickets, err := ticketSearchService(client).List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to search tickets")
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
				// A ticket created with notifyCustomer (or cc/bcc) emails the external
				// customer a copy that cannot be unsent, so it can reach an open world
				// of external recipients and is irreversible.
				DestructiveHint: new(true),
				OpenWorldHint:   new(true),
			},
			Description: "Create ticket.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"subject": {
						Type:        "string",
						Description: "The subject of the ticket.",
					},
					"body": {
						Type:        "string",
						Description: "The body of the ticket.",
					},
					"inboxId": {
						Type:        "integer",
						Description: "Inbox of the ticket. Use twdesk-list_inboxes to discover.",
					},
					"notifyCustomer": {
						Description: "Set to true if the customer should be sent a copy of the ticket.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"bcc": {
						Description: "Email addresses to BCC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
					"cc": {
						Description: "Email addresses to CC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
					"files": {
						Description: "File IDs to attach. Use twdesk-create_file to upload.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"tags": {
						Description: "Tags to associate with the ticket. Use twdesk-list_tags to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"priorityId": {
						Description: "Priority of the ticket. Use twdesk-list_priorities to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"statusId": {
						Description: "Status of the ticket. Use twdesk-list_statuses to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"customerId": {
						Description: "Customer of the ticket. Use twdesk-list_customers to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"customerEmail": {
						Description: "Customer email; required when customerId is not given. Existing customers are matched, " +
							"otherwise a new customer is created.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"typeId": {
						Description: "Ticket type. Use twdesk-list_ticket_types to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"agentId": {
						Description: "Agent the ticket is assigned to. Use twdesk-list_users to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{
					"subject", "body", "inboxId",
					"notifyCustomer", "bcc", "cc", "files", "tags",
					"priorityId", "statusId", "customerId", "customerEmail", "typeId", "agentId",
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			subject := arguments.GetString("subject", "")
			body := arguments.GetString("body", "")
			data := deskmodels.Ticket{
				Subject: &subject,
				Body:    &body,
				Inbox: &deskmodels.EntityRef{
					ID: arguments.GetInt("inboxId", 0),
				},
			}

			if arguments.GetInt("customerId", 0) != 0 {
				data.Customer = &deskmodels.EntityRef{
					ID: arguments.GetInt("customerId", 0),
				}
			}

			email := arguments.GetString("customerEmail", "")
			if email != "" {
				filter := deskclient.NewFilter()
				filter = filter.Eq("contacts.value", email)

				params := url.Values{}
				params.Set("filter", filter.Build())
				setPagination(&params, arguments)

				customers, err := client.Customers.List(ctx, params)
				if err != nil {
					return helpers.HandleAPIError(err, "failed to list customers")
				}

				if len(customers.Customers) > 0 {
					data.Customer = &deskmodels.EntityRef{
						ID: customers.Customers[0].ID,
					}
				} else {
					// Create the customer
					customer, err := client.Customers.Create(ctx, &deskmodels.CustomerResponse{
						Customer: deskmodels.Customer{
							Email: strPtr(email),
						},
					})
					if err != nil {
						return helpers.HandleAPIError(err, "failed to create customer")
					}
					data.Customer = &deskmodels.EntityRef{
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
				data.NotifyCustomer = new(true)
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
				return helpers.HandleAPIError(err, "failed to create ticket")
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
				Title:           "Update Ticket",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Update ticket.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the ticket to update.",
					},
					"subject": {
						Description: "The subject of the ticket.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"body": {
						Description: "The body of the ticket.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tags": {
						Description: "Tags to associate with the ticket. Use twdesk-list_tags to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"deleteTags": {
						Description: "Tags to remove from the ticket. Use twdesk-list_tags to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"bcc": {
						Description: "Email addresses to BCC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
					"cc": {
						Description: "Email addresses to CC.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
					"inboxId": {
						Description: "Inbox of the ticket. Use twdesk-list_inboxes to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"priorityId": {
						Description: "Priority of the ticket. Use twdesk-list_priorities to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"statusId": {
						Description: "Status of the ticket. Use twdesk-list_statuses to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"typeId": {
						Description: "Ticket type. Use twdesk-list_ticket_types to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"agentId": {
						Description: "Agent the ticket is assigned to. Use twdesk-list_users to discover.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{
					"id", "subject", "body", "tags", "deleteTags",
					"bcc", "cc", "inboxId", "priorityId", "statusId", "typeId", "agentId",
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			data := deskmodels.Ticket{}

			subject := arguments.GetString("subject", "")
			if subject != "" {
				data.Subject = &subject
			}

			inboxID := arguments.GetInt("inboxId", 0)
			if inboxID > 0 {
				data.Inbox = &deskmodels.EntityRef{
					ID: inboxID,
				}
			}

			body := arguments.GetString("body", "")
			if body != "" {
				data.Body = &body
			}

			data.Tags = []deskmodels.EntityRef{}
			if len(arguments.GetIntSlice("tags", []int{})) > 0 {
				for _, tagID := range arguments.GetIntSlice("tags", []int{}) {
					data.Tags = append(data.Tags, deskmodels.EntityRef{ID: tagID})
				}
			}

			if len(arguments.GetIntSlice("deleteTags", []int{})) > 0 {
				for _, tagID := range arguments.GetIntSlice("deleteTags", []int{}) {
					data.Tags = append(data.Tags, deskmodels.EntityRef{
						ID: tagID,
						Meta: map[string]any{
							"delete": true,
						},
					})
				}
			}

			if len(arguments.GetStringSlice("bcc", []string{})) > 0 {
				data.BCC = arguments.GetStringSlice("bcc", []string{})
			}

			if len(arguments.GetStringSlice("cc", []string{})) > 0 {
				data.CC = arguments.GetStringSlice("cc", []string{})
			}

			statusID := arguments.GetInt("statusId", 0)
			if statusID > 0 {
				data.Status = &deskmodels.EntityRef{ID: statusID}
			}

			typeID := arguments.GetInt("typeId", 0)
			if typeID > 0 {
				data.Type = &deskmodels.EntityRef{ID: typeID}
			}

			agentID := arguments.GetInt("agentId", 0)
			if agentID > 0 {
				data.Agent = &deskmodels.EntityRef{ID: agentID}
			}

			ticket, err := client.Tickets.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TicketResponse{
				Ticket: data,
			})
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update ticket")
			}
			return helpers.NewToolResultJSON(ticket)
		},
	}
}
