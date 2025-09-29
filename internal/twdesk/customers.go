package twdesk

import (
	"context"
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
func CustomerGet(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCustomerGet),
			mcp.WithDescription(
				"Retrieve detailed information about a specific customer in Teamwork Desk by their ID. "+
					"Useful for auditing customer records, troubleshooting ticket associations, or "+
					"integrating Desk customer data into automation workflows."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the customer to retrieve."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			customer, err := client.Customers.Get(ctx, request.GetInt("id", 0))
			if err != nil {
				return nil, fmt.Errorf("failed to get customer: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Customer retrieved successfully: %s", customer.Customer.FirstName)), nil
		},
	}
}

// CustomerList returns a list of customers that apply to the filters in Teamwork Desk
func CustomerList(client *deskclient.Client) server.ServerTool {
	opts := []mcp.ToolOption{
		mcp.WithOutputSchema[deskmodels.CustomersResponse](),
		mcp.WithDescription(
			"List all customers in Teamwork Desk, with optional filters for company, email, and other attributes. " +
				"Enables users to audit, analyze, or synchronize customer configurations for ticket management, " +
				"reporting, or integration scenarios."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("companyIDs", mcp.Description("The IDs of the companies to filter by.")),
		mcp.WithArray("companyNames", mcp.Description("The names of the companies to filter by.")),
		mcp.WithArray("emails", mcp.Description("The emails of the customers to filter by.")),
	}

	opts = append(opts, paginationOptions()...)

	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCustomerList), opts...),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Apply filters to the customer list
			companyIDs := request.GetIntSlice("companyIDs", []int{})
			companyNames := request.GetStringSlice("companyNames", []string{})
			emails := request.GetStringSlice("emails", []string{})

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
			setPagination(&params, request)

			customers, err := client.Customers.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list customers: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Customers retrieved successfully: %v", customers)), nil
		},
	}
}

// CustomerCreate creates a customer in Teamwork Desk
func CustomerCreate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCustomerCreate),
			mcp.WithDescription(
				"Create a new customer in Teamwork Desk by specifying their name, contact details, and other attributes. "+
					"Useful for onboarding new clients, customizing Desk for business relationships, or "+
					"adapting support processes."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the customer to update."),
			),
			mcp.WithString("firstName",
				mcp.Description("The first name of the customer."),
			),
			mcp.WithString("lastName",
				mcp.Description("The last name of the customer."),
			),
			mcp.WithString("email",
				mcp.Description("The email of the customer."),
			),
			mcp.WithString("organization",
				mcp.Description("The organization of the customer."),
			),
			mcp.WithString("extraData",
				mcp.Description("The extra data of the customer."),
			),
			mcp.WithString("notes",
				mcp.Description("The notes of the customer."),
			),
			mcp.WithString("linkedinURL",
				mcp.Description("The LinkedIn URL of the customer."),
			),
			mcp.WithString("facebookURL",
				mcp.Description("The Facebook URL of the customer."),
			),
			mcp.WithString("twitterHandle",
				mcp.Description("The Twitter handle of the customer."),
			),
			mcp.WithString("jobTitle",
				mcp.Description("The job title of the customer."),
			),
			mcp.WithString("phone",
				mcp.Description("The phone number of the customer."),
			),
			mcp.WithString("mobile",
				mcp.Description("The mobile number of the customer."),
			),
			mcp.WithString("address",
				mcp.Description("The address of the customer."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			domains := request.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}

			customer, err := client.Customers.Create(ctx, &deskmodels.CustomerResponse{
				Customer: deskmodels.Customer{
					FirstName:     request.GetString("firstName", ""),
					LastName:      request.GetString("lastName", ""),
					Email:         request.GetString("email", ""),
					Organization:  request.GetString("organization", ""),
					ExtraData:     request.GetString("extraData", ""),
					Notes:         request.GetString("notes", ""),
					LinkedinURL:   request.GetString("linkedinURL", ""),
					FacebookURL:   request.GetString("facebookURL", ""),
					TwitterHandle: request.GetString("twitterHandle", ""),
					JobTitle:      request.GetString("jobTitle", ""),
					Phone:         request.GetString("phone", ""),
					Mobile:        request.GetString("mobile", ""),
					Address:       request.GetString("address", ""),
					Trusted:       request.GetBool("trusted", false),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Customer created successfully with ID %d", customer.Customer.ID)), nil
		},
	}
}

// CustomerUpdate updates a customer in Teamwork Desk
func CustomerUpdate(client *deskclient.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCustomerUpdate),
			mcp.WithDescription(
				"Update an existing customer in Teamwork Desk by ID, allowing changes to their name, "+
					"contact details, and other attributes. Supports evolving business relationships, "+
					"correcting customer records, or improving ticket handling."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The ID of the customer to update."),
			),
			mcp.WithString("firstName",
				mcp.Description("The new first name of the customer."),
			),
			mcp.WithString("lastName",
				mcp.Description("The new last name of the customer."),
			),
			mcp.WithString("email",
				mcp.Description("The new email of the customer."),
			),
			mcp.WithString("organization",
				mcp.Description("The new organization of the customer."),
			),
			mcp.WithString("extraData",
				mcp.Description("The new extra data of the customer."),
			),
			mcp.WithString("notes",
				mcp.Description("The new notes of the customer."),
			),
			mcp.WithString("linkedinURL",
				mcp.Description("The new LinkedIn URL of the customer."),
			),
			mcp.WithString("facebookURL",
				mcp.Description("The new Facebook URL of the customer."),
			),
			mcp.WithString("twitterHandle",
				mcp.Description("The new Twitter handle of the customer."),
			),
			mcp.WithString("jobTitle",
				mcp.Description("The new job title of the customer."),
			),
			mcp.WithString("phone",
				mcp.Description("The new phone number of the customer."),
			),
			mcp.WithString("mobile",
				mcp.Description("The new mobile number of the customer."),
			),
			mcp.WithString("address",
				mcp.Description("The new address of the customer."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			domains := request.GetStringSlice("domains", []string{})
			domainEntities := make([]deskmodels.Domain, len(domains))
			for i, domain := range domains {
				domainEntities[i] = deskmodels.Domain{
					Name: domain,
				}
			}
			_, err := client.Customers.Update(ctx, request.GetInt("id", 0), &deskmodels.CustomerResponse{
				Customer: deskmodels.Customer{
					FirstName:     request.GetString("firstName", ""),
					LastName:      request.GetString("lastName", ""),
					Email:         request.GetString("email", ""),
					Organization:  request.GetString("organization", ""),
					ExtraData:     request.GetString("extraData", ""),
					Notes:         request.GetString("notes", ""),
					LinkedinURL:   request.GetString("linkedinURL", ""),
					FacebookURL:   request.GetString("facebookURL", ""),
					TwitterHandle: request.GetString("twitterHandle", ""),
					JobTitle:      request.GetString("jobTitle", ""),
					Phone:         request.GetString("phone", ""),
					Mobile:        request.GetString("mobile", ""),
					Address:       request.GetString("address", ""),
					Trusted:       request.GetBool("trusted", false),
				},
				Included: deskmodels.IncludedData{
					Domains: domainEntities,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}

			return mcp.NewToolResultText("Customer updated successfully"), nil
		},
	}
}
