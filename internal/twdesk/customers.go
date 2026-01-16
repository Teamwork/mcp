package twdesk

import (
	"context"
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
	MethodCustomerCreate toolsets.Method = "twdesk-create_customer"
	MethodCustomerUpdate toolsets.Method = "twdesk-update_customer"
	MethodCustomerGet    toolsets.Method = "twdesk-get_customer"
	MethodCustomerList   toolsets.Method = "twdesk-list_customers"
)

func init() {
	toolsets.RegisterMethod(MethodCustomerCreate)
	toolsets.RegisterMethod(MethodCustomerUpdate)
	toolsets.RegisterMethod(MethodCustomerGet)
	toolsets.RegisterMethod(MethodCustomerList)
}

// CustomerGet finds a customer in Teamwork Desk.  This will find it by ID
func CustomerGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomerGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Customer",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific customer in Teamwork Desk by their ID. " +
				"Useful for auditing customer records, troubleshooting ticket associations, or " +
				"integrating Desk customer data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the customer to retrieve.",
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

			customer, err := client.Customers.Get(ctx, arguments.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get customer: %w", err)
			}

			firstName := customer.Customer.FirstName
			return helpers.NewToolResultText("Customer retrieved successfully: %s", firstName), nil
		},
	}
}

// CustomerList returns a list of customers that apply to the filters in Teamwork Desk
func CustomerList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"companyIDs": {
			Type:        "array",
			Description: "The IDs of the companies to filter by.",
			Items: &jsonschema.Schema{
				Type: "integer",
			},
		},
		"companyNames": {
			Type:        "array",
			Description: "The names of the companies to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
		"emails": {
			Type:        "array",
			Description: "The emails of the customers to filter by.",
			Items: &jsonschema.Schema{
				Type: "string",
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomerList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Customers",
				ReadOnlyHint: true,
			},
			Description: "List all customers in Teamwork Desk, with optional filters for company, email, and other " +
				"attributes. Enables users to audit, analyze, or synchronize customer configurations for ticket management, " +
				"reporting, or integration scenarios.",
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

			// Apply filters to the customer list
			companyIDs := arguments.GetIntSlice("companyIDs", []int{})
			companyNames := arguments.GetStringSlice("companyNames", []string{})
			emails := arguments.GetStringSlice("emails", []string{})

			filter := deskclient.NewFilter()
			if len(companyIDs) > 0 {
				filter = filter.In("companies.id", helpers.SliceToAny(companyIDs))
			}

			if len(companyNames) > 0 {
				filter = filter.In("companies.name", helpers.SliceToAny(companyNames))
			}

			if len(emails) > 0 {
				filter = filter.In("contacts.value", helpers.SliceToAny(emails))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			customers, err := client.Customers.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list customers: %w", err)
			}

			return helpers.NewToolResultJSON(customers)
		},
	}
}

// CustomerCreate creates a customer in Teamwork Desk
func CustomerCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomerCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Customer",
			},
			Description: "Create a new customer in Teamwork Desk by specifying their name, contact details, and other " +
				"attributes. Useful for onboarding new clients, customizing Desk for business relationships, or " +
				"adapting support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"firstName": {
						Type:        "string",
						Description: "The first name of the customer.",
					},
					"lastName": {
						Type:        "string",
						Description: "The last name of the customer.",
					},
					"email": {
						Type:        "string",
						Description: "The email of the customer.",
					},
					"organization": {
						Type:        "string",
						Description: "The organization of the customer.",
					},
					"extraData": {
						Type:        "string",
						Description: "The extra data of the customer.",
					},
					"notes": {
						Type:        "string",
						Description: "The notes of the customer.",
					},
					"linkedinURL": {
						Type:        "string",
						Description: "The LinkedIn URL of the customer.",
					},
					"facebookURL": {
						Type:        "string",
						Description: "The Facebook URL of the customer.",
					},
					"twitterHandle": {
						Type:        "string",
						Description: "The Twitter handle of the customer.",
					},
					"jobTitle": {
						Type:        "string",
						Description: "The job title of the customer.",
					},
					"phone": {
						Type:        "string",
						Description: "The phone number of the customer.",
					},
					"mobile": {
						Type:        "string",
						Description: "The mobile number of the customer.",
					},
					"address": {
						Type:        "string",
						Description: "The address of the customer.",
					},
				},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}

			customer, err := client.Customers.Create(ctx, &deskmodels.CustomerResponse{
				Customer: deskmodels.Customer{
					FirstName:     arguments.GetString("firstName", ""),
					LastName:      arguments.GetString("lastName", ""),
					Email:         arguments.GetString("email", ""),
					Organization:  arguments.GetString("organization", ""),
					ExtraData:     arguments.GetString("extraData", ""),
					Notes:         arguments.GetString("notes", ""),
					LinkedinURL:   arguments.GetString("linkedinURL", ""),
					FacebookURL:   arguments.GetString("facebookURL", ""),
					TwitterHandle: arguments.GetString("twitterHandle", ""),
					JobTitle:      arguments.GetString("jobTitle", ""),
					Phone:         arguments.GetString("phone", ""),
					Mobile:        arguments.GetString("mobile", ""),
					Address:       arguments.GetString("address", ""),
					Trusted:       arguments.GetBool("trusted", false),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}
			return helpers.NewToolResultText("Customer created successfully with ID %d", customer.Customer.ID), nil
		},
	}
}

// CustomerUpdate updates a customer in Teamwork Desk
func CustomerUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomerUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Customer",
			},
			Description: "Update an existing customer in Teamwork Desk by ID, allowing changes to their name, " +
				"contact details, and other attributes. Supports evolving business relationships, " +
				"correcting customer records, or improving ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the customer to update.",
					},
					"firstName": {
						Type:        "string",
						Description: "The new first name of the customer.",
					},
					"lastName": {
						Type:        "string",
						Description: "The new last name of the customer.",
					},
					"email": {
						Type:        "string",
						Description: "The new email of the customer.",
					},
					"organization": {
						Type:        "string",
						Description: "The new organization of the customer.",
					},
					"extraData": {
						Type:        "string",
						Description: "The new extra data of the customer.",
					},
					"notes": {
						Type:        "string",
						Description: "The new notes of the customer.",
					},
					"linkedinURL": {
						Type:        "string",
						Description: "The new LinkedIn URL of the customer.",
					},
					"facebookURL": {
						Type:        "string",
						Description: "The new Facebook URL of the customer.",
					},
					"twitterHandle": {
						Type:        "string",
						Description: "The new Twitter handle of the customer.",
					},
					"jobTitle": {
						Type:        "string",
						Description: "The new job title of the customer.",
					},
					"phone": {
						Type:        "string",
						Description: "The new phone number of the customer.",
					},
					"mobile": {
						Type:        "string",
						Description: "The new mobile number of the customer.",
					},
					"address": {
						Type:        "string",
						Description: "The new address of the customer.",
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

			domains := arguments.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}
			_, err = client.Customers.Update(ctx, arguments.GetInt("id", 0), &deskmodels.CustomerResponse{
				Customer: deskmodels.Customer{
					FirstName:     arguments.GetString("firstName", ""),
					LastName:      arguments.GetString("lastName", ""),
					Email:         arguments.GetString("email", ""),
					Organization:  arguments.GetString("organization", ""),
					ExtraData:     arguments.GetString("extraData", ""),
					Notes:         arguments.GetString("notes", ""),
					LinkedinURL:   arguments.GetString("linkedinURL", ""),
					FacebookURL:   arguments.GetString("facebookURL", ""),
					TwitterHandle: arguments.GetString("twitterHandle", ""),
					JobTitle:      arguments.GetString("jobTitle", ""),
					Phone:         arguments.GetString("phone", ""),
					Mobile:        arguments.GetString("mobile", ""),
					Address:       arguments.GetString("address", ""),
					Trusted:       arguments.GetBool("trusted", false),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}

			return helpers.NewToolResultText("Customer updated successfully"), nil
		},
	}
}
