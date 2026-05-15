// Package testutil provides schema validation helpers for testing MCP tools
//
//nolint:lll
package testutil

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
)

// SchemaValidationTestSuite provides a comprehensive test suite for validating MCP tool JSON schemas
type SchemaValidationTestSuite struct {
	tools       map[string]toolsets.ToolWrapper
	validData   map[string]map[string]map[string]any // [toolName][testCase] -> data
	invalidData map[string]map[string]map[string]any // [toolName][testCase] -> data
}

// NewSchemaValidationTestSuite creates a new test suite with all twdesk tools
func NewSchemaValidationTestSuite() *SchemaValidationTestSuite {
	httpClient := &http.Client{}

	tools := map[string]toolsets.ToolWrapper{
		// Company tools
		"CompanyCreate": twdesk.CompanyCreate(httpClient),
		"CompanyUpdate": twdesk.CompanyUpdate(httpClient),
		"CompanyGet":    twdesk.CompanyGet(httpClient),
		"CompanyList":   twdesk.CompanyList(httpClient),

		// Customer tools
		"CustomerCreate": twdesk.CustomerCreate(httpClient),
		"CustomerUpdate": twdesk.CustomerUpdate(httpClient),
		"CustomerGet":    twdesk.CustomerGet(httpClient),
		"CustomerList":   twdesk.CustomerList(httpClient),

		// Ticket tools
		"TicketCreate": twdesk.TicketCreate(httpClient),
		"TicketUpdate": twdesk.TicketUpdate(httpClient),
		"TicketGet":    twdesk.TicketGet(httpClient),
		"TicketSearch": twdesk.TicketSearch(httpClient),

		// Priority tools
		"PriorityCreate": twdesk.PriorityCreate(httpClient),
		"PriorityUpdate": twdesk.PriorityUpdate(httpClient),
		"PriorityGet":    twdesk.PriorityGet(httpClient),
		"PriorityList":   twdesk.PriorityList(httpClient),

		// Status tools
		"StatusCreate": twdesk.StatusCreate(httpClient),
		"StatusUpdate": twdesk.StatusUpdate(httpClient),
		"StatusGet":    twdesk.StatusGet(httpClient),
		"StatusList":   twdesk.StatusList(httpClient),

		// Tag tools
		"TagCreate": twdesk.TagCreate(httpClient),
		"TagUpdate": twdesk.TagUpdate(httpClient),
		"TagGet":    twdesk.TagGet(httpClient),
		"TagList":   twdesk.TagList(httpClient),

		// Type tools
		"TypeCreate": twdesk.TypeCreate(httpClient),
		"TypeUpdate": twdesk.TypeUpdate(httpClient),
		"TypeGet":    twdesk.TypeGet(httpClient),
		"TypeList":   twdesk.TypeList(httpClient),

		// User tools
		"UserGet":  twdesk.UserGet(httpClient),
		"UserList": twdesk.UserList(httpClient),

		// Message tools
		"MessageCreate": twdesk.MessageCreate(httpClient),

		// File tools
		"FileCreate": twdesk.FileCreate(httpClient),
	}

	return &SchemaValidationTestSuite{
		tools:       tools,
		validData:   GetValidTestData(),
		invalidData: GetInvalidTestData(),
	}
}

// RunAllSchemaValidationTests runs comprehensive schema validation tests for all tools
func (s *SchemaValidationTestSuite) RunAllSchemaValidationTests(t *testing.T) {
	for toolName, tool := range s.tools {
		t.Run(toolName, func(t *testing.T) {
			s.runToolSchemaValidation(t, toolName, tool)
		})
	}
}

// GetTool returns a tool by name if it exists
func (s *SchemaValidationTestSuite) GetTool(toolName string) (toolsets.ToolWrapper, bool) {
	tool, exists := s.tools[toolName]
	return tool, exists
}

// RunToolSchemaValidation runs schema validation tests for a single tool (exported version)
func (s *SchemaValidationTestSuite) RunToolSchemaValidation(t *testing.T, toolName string, tool toolsets.ToolWrapper) {
	s.runToolSchemaValidation(t, toolName, tool)
}

// runToolSchemaValidation runs schema validation tests for a single tool
func (s *SchemaValidationTestSuite) runToolSchemaValidation(t *testing.T, toolName string, tool toolsets.ToolWrapper) {
	inputSchema := tool.Tool.InputSchema

	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		t.Fatalf("Failed to marshal input schema to JSON: %v", err)
	}

	var schema jsonschema.Schema
	err = json.Unmarshal(schemaBytes, &schema)
	if err != nil {
		t.Fatalf("Invalid JSON schema for %s tool: %v\nSchema: %s", toolName, err, string(schemaBytes))
	}

	resolvedSchema, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("Failed to resolve schema for %s tool: %v", toolName, err)
	}

	t.Run("ValidateValidData", func(t *testing.T) {
		s.testValidDataAgainstSchema(t, toolName, resolvedSchema)
	})

	t.Run("ValidateInvalidData", func(t *testing.T) {
		s.testInvalidDataAgainstSchema(t, toolName, resolvedSchema)
	})

	t.Run("ValidateArrayItemTypes", func(t *testing.T) {
		s.validateArrayItemTypes(t, toolName, inputSchema)
	})
}

// testValidDataAgainstSchema tests the schema with valid input data
func (s *SchemaValidationTestSuite) testValidDataAgainstSchema(t *testing.T, toolName string, resolvedSchema *jsonschema.Resolved) {
	validTestData, exists := s.validData[toolName]
	if !exists {
		t.Logf("No valid test data defined for %s tool, skipping", toolName)
		return
	}

	for testName, testData := range validTestData {
		t.Run(testName, func(t *testing.T) {
			err := resolvedSchema.Validate(testData)
			if err != nil {
				t.Errorf("Valid data should pass schema validation for %s tool.\nError: %v\nData: %+v",
					toolName, err, testData)
			}
		})
	}
}

// testInvalidDataAgainstSchema tests the schema with invalid input data
func (s *SchemaValidationTestSuite) testInvalidDataAgainstSchema(t *testing.T, toolName string, resolvedSchema *jsonschema.Resolved) {
	invalidTestData, exists := s.invalidData[toolName]
	if !exists {
		t.Logf("No invalid test data defined for %s tool, skipping", toolName)
		return
	}

	for testName, testData := range invalidTestData {
		t.Run(testName, func(t *testing.T) {
			err := resolvedSchema.Validate(testData)
			if err == nil {
				t.Errorf("Invalid data should fail schema validation for %s tool.\nData: %+v",
					toolName, testData)
			}
		})
	}
}

// validateArrayItemTypes specifically checks that array properties have proper string type constraints
func (s *SchemaValidationTestSuite) validateArrayItemTypes(t *testing.T, toolName string, inputSchema any) {
	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		t.Fatalf("Failed to marshal schema for %s tool: %v", toolName, err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema for %s tool: %v", toolName, err)
	}

	properties, ok := schemaMap["properties"].(map[string]any)
	if !ok {
		return
	}

	for propName, property := range properties {
		propertyMap, ok := property.(map[string]any)
		if !ok {
			continue
		}

		if propertyType, exists := propertyMap["type"]; exists && propertyType == "array" {
			if items, exists := propertyMap["items"]; exists {
				itemsMap, ok := items.(map[string]any)
				if !ok {
					t.Errorf("%s property items should be a map for %s tool", propName, toolName)
					continue
				}

				if itemType, exists := itemsMap["type"]; exists {
					if itemType == "" {
						t.Errorf("%s array items should have a non-empty type for %s tool", propName, toolName)
					}
				} else {
					t.Errorf("%s array items should have a 'type' property for %s tool", propName, toolName)
				}
			} else {
				t.Errorf("%s array should have an 'items' property for %s tool", propName, toolName)
			}
		}
	}
}

// GetValidTestData returns valid test data for all tools.
// All required fields (including nullable optional ones) must be provided —
// pass nil to satisfy a required nullable field without setting a value.
func GetValidTestData() map[string]map[string]map[string]any {
	return map[string]map[string]map[string]any{
		"CompanyCreate": {
			"minimal": {
				"name": "Test Company", "description": nil, "details": nil,
				"industry": nil, "website": nil, "permission": nil,
				"kind": nil, "note": nil, "domains": nil,
			},
			"complete": {
				"name": "Test Company", "description": "A test company",
				"details": "Company details", "industry": "Technology",
				"website": "https://example.com", "permission": "own",
				"kind": "company", "note": "Test note",
				"domains": []string{"example.com", "test.com"},
			},
		},
		"CompanyUpdate": {
			"minimal": {
				"id": 123, "name": nil, "description": nil, "details": nil,
				"industry": nil, "website": nil, "permission": nil,
				"kind": nil, "note": nil, "domains": nil,
			},
			"complete": {
				"id": 123, "name": "Updated Company",
				"description": "Updated description", "details": nil,
				"industry": nil, "website": nil, "permission": nil,
				"kind": nil, "note": nil, "domains": []string{"updated.com"},
			},
		},
		"CompanyGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"CompanyList": {
			"empty": {
				"name": nil, "domains": nil, "kind": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
			"with_filters": {
				"name": "Test Company", "domains": []string{"example.com"},
				"kind": "company", "page": 1, "pageSize": 10,
				"orderBy": nil, "orderDirection": nil, "fields": nil,
			},
		},
		"CustomerCreate": {
			"minimal": {
				"firstName": nil, "lastName": nil, "email": "john.doe@example.com",
				"organization": nil, "extraData": nil, "notes": nil,
				"linkedinURL": nil, "facebookURL": nil, "twitterHandle": nil,
				"jobTitle": nil, "phone": nil, "mobile": nil, "address": nil,
			},
		},
		"CustomerUpdate": {
			"minimal": {
				"id": 123, "firstName": nil, "lastName": nil, "email": nil,
				"organization": nil, "extraData": nil, "notes": nil,
				"linkedinURL": nil, "facebookURL": nil, "twitterHandle": nil,
				"jobTitle": nil, "phone": nil, "mobile": nil, "address": nil,
			},
		},
		"CustomerGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"CustomerList": {
			"empty": {
				"companyIDs": nil, "companyNames": nil, "emails": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
		},
		"TicketCreate": {
			"minimal": {
				"subject": "Test Ticket", "body": "Test message", "inboxId": 1,
				"notifyCustomer": nil, "bcc": nil, "cc": nil, "files": nil,
				"tags": nil, "priorityId": 1, "statusId": 1,
				"customerId": 1, "customerEmail": nil, "typeId": 1, "agentId": 1,
			},
		},
		"TicketUpdate": {
			"minimal": {
				"id": 123, "subject": "Updated Ticket", "body": nil,
				"tags": nil, "deleteTags": nil, "bcc": nil, "cc": nil,
				"inboxId": nil, "priorityId": nil, "statusId": nil,
				"typeId": nil, "agentId": nil,
			},
		},
		"TicketGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"TicketSearch": {
			"empty": {
				"search": nil, "inboxIDs": nil, "customerIDs": nil,
				"companyIDs": nil, "tagIDs": nil, "statusIDs": nil,
				"priorityIDs": nil, "userIDs": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
		},
		"PriorityCreate": {
			"minimal": {"name": "High Priority", "color": nil},
		},
		"PriorityUpdate": {
			"minimal": {"id": 123, "name": nil, "color": nil},
		},
		"PriorityGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"PriorityList": {
			"empty": {
				"name": nil, "color": nil, "page": nil, "pageSize": nil,
				"orderBy": nil, "orderDirection": nil, "fields": nil,
			},
		},
		"StatusCreate": {
			"minimal": {"name": "Open", "color": nil, "displayOrder": nil},
		},
		"StatusUpdate": {
			"minimal": {"id": 123, "name": nil, "color": nil, "displayOrder": nil},
		},
		"StatusGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"StatusList": {
			"empty": {
				"name": nil, "color": nil, "code": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
		},
		"TagCreate": {
			"minimal": {"name": "Important", "color": nil},
		},
		"TagUpdate": {
			"minimal": {"id": 123, "name": nil, "color": nil},
		},
		"TagGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"TagList": {
			"empty": {
				"name": nil, "color": nil, "inboxIDs": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
		},
		"TypeCreate": {
			"minimal": {"name": "Bug Report", "displayOrder": nil, "enabledForFutureInboxes": nil},
		},
		"TypeUpdate": {
			"minimal": {"id": 123, "name": nil, "displayOrder": nil, "enabledForFutureInboxes": nil},
		},
		"TypeGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"TypeList": {
			"empty": {
				"name": nil, "inboxIDs": nil, "page": nil, "pageSize": nil,
				"orderBy": nil, "orderDirection": nil, "fields": nil,
			},
		},
		"UserGet": {
			"valid": {"id": 123, "fields": nil},
		},
		"UserList": {
			"empty": {
				"firstName": nil, "lastName": nil, "email": nil,
				"inboxIDs": nil, "isPartTime": nil,
				"page": nil, "pageSize": nil, "orderBy": nil,
				"orderDirection": nil, "fields": nil,
			},
		},
		"MessageCreate": {
			"minimal": {
				"ticketID": 123, "body": "Test message",
				"threadType": nil, "bcc": nil, "cc": nil,
			},
		},
		"FileCreate": {
			"minimal": {
				"name": "test.txt", "mimeType": "text/plain",
				"data": "VGVzdCBjb250ZW50", "disposition": nil,
			},
		},
	}
}

// GetInvalidTestData returns invalid test data for all tools
func GetInvalidTestData() map[string]map[string]map[string]any {
	return map[string]map[string]map[string]any{
		"CompanyCreate": {
			"missing_required_name": {
				"description": "A test company",
			},
			"invalid_permission": {
				"name":       "Test Company",
				"permission": "invalid_permission",
			},
			"invalid_kind": {
				"name": "Test Company",
				"kind": "invalid_kind",
			},
			"invalid_domains_type": {
				"name":    "Test Company",
				"domains": "should_be_array",
			},
			"invalid_domain_item_type": {
				"name":    "Test Company",
				"domains": []any{123, 456}, // should be strings
			},
		},
		"CompanyUpdate": {
			"missing_required_id": {
				"name": "Updated Company",
			},
			"invalid_domains_type": {
				"id":      123,
				"domains": "should_be_array",
			},
		},
		"CompanyGet": {
			"missing_required_id": {},
		},
		"CompanyList": {
			"invalid_kind": {
				"kind": "invalid_kind",
			},
			"invalid_domains_type": {
				"domains": "should_be_array",
			},
		},
		"CustomerCreate": {
			"invalid_property_type": {
				"firstName": 123, // should be string, not number
				"lastName":  "Doe",
				"email":     "john@example.com",
			},
		},
		"CustomerGet": {
			"missing_required_id": {},
		},
		"TicketCreate": {
			"missing_required_subject": {
				"body": "Test message",
			},
		},
		"TicketUpdate": {
			"missing_required_id": {
				"subject": "Updated Ticket",
			},
		},
		"TicketGet": {
			"missing_required_id": {},
		},
		"PriorityCreate": {
			"missing_required_name": {},
		},
		"PriorityGet": {
			"missing_required_id": {},
		},
		"StatusCreate": {
			"missing_required_name": {},
		},
		"StatusGet": {
			"missing_required_id": {},
		},
		"TagCreate": {
			"missing_required_name": {},
		},
		"TagGet": {
			"missing_required_id": {},
		},
		"TypeCreate": {
			"missing_required_name": {},
		},
		"TypeGet": {
			"missing_required_id": {},
		},
		"UserGet": {
			"missing_required_id": {},
		},
		"MessageCreate": {
			"missing_required_ticketID": {
				"body": "Test message",
			},
		},
		"FileCreate": {
			"missing_required_name": {
				"mimeType": "text/plain",
				"data":     "VGVzdCBjb250ZW50",
			},
			"missing_required_data": {
				"name":     "test.txt",
				"mimeType": "text/plain",
			},
		},
	}
}
