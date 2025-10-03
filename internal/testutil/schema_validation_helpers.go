// Package testutil provides schema validation helpers for testing MCP tools
//
//nolint:lll
package testutil

import (
	"encoding/json"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	deskclient "github.com/teamwork/desksdkgo/client"
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
	client := &deskclient.Client{}

	tools := map[string]toolsets.ToolWrapper{
		// Company tools
		"CompanyCreate": twdesk.CompanyCreate(client),
		"CompanyUpdate": twdesk.CompanyUpdate(client),
		"CompanyGet":    twdesk.CompanyGet(client),
		"CompanyList":   twdesk.CompanyList(client),

		// Customer tools
		"CustomerCreate": twdesk.CustomerCreate(client),
		"CustomerUpdate": twdesk.CustomerUpdate(client),
		"CustomerGet":    twdesk.CustomerGet(client),
		"CustomerList":   twdesk.CustomerList(client),

		// Ticket tools
		"TicketCreate": twdesk.TicketCreate(client),
		"TicketUpdate": twdesk.TicketUpdate(client),
		"TicketGet":    twdesk.TicketGet(client),
		"TicketList":   twdesk.TicketList(client),

		// Priority tools
		"PriorityCreate": twdesk.PriorityCreate(client),
		"PriorityUpdate": twdesk.PriorityUpdate(client),
		"PriorityGet":    twdesk.PriorityGet(client),
		"PriorityList":   twdesk.PriorityList(client),

		// Status tools
		"StatusCreate": twdesk.StatusCreate(client),
		"StatusUpdate": twdesk.StatusUpdate(client),
		"StatusGet":    twdesk.StatusGet(client),
		"StatusList":   twdesk.StatusList(client),

		// Tag tools
		"TagCreate": twdesk.TagCreate(client),
		"TagUpdate": twdesk.TagUpdate(client),
		"TagGet":    twdesk.TagGet(client),
		"TagList":   twdesk.TagList(client),

		// Type tools
		"TypeCreate": twdesk.TypeCreate(client),
		"TypeUpdate": twdesk.TypeUpdate(client),
		"TypeGet":    twdesk.TypeGet(client),
		"TypeList":   twdesk.TypeList(client),

		// User tools
		"UserGet":  twdesk.UserGet(client),
		"UserList": twdesk.UserList(client),

		// Message tools
		"MessageCreate": twdesk.MessageCreate(client),

		// File tools
		"FileCreate": twdesk.FileCreate(client),
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

// GetValidTestData returns valid test data for all tools
func GetValidTestData() map[string]map[string]map[string]any {
	return map[string]map[string]map[string]any{
		"CompanyCreate": {
			"minimal": {
				"name": "Test Company",
			},
			"complete": {
				"name":        "Test Company",
				"description": "A test company",
				"details":     "Company details",
				"industry":    "Technology",
				"website":     "https://example.com",
				"permission":  "own",
				"kind":        "company",
				"note":        "Test note",
				"domains":     []string{"example.com", "test.com"},
			},
		},
		"CompanyUpdate": {
			"minimal": {
				"id": 123,
			},
			"complete": {
				"id":          123,
				"name":        "Updated Company",
				"description": "Updated description",
				"domains":     []string{"updated.com"},
			},
		},
		"CompanyGet": {
			"valid": {
				"id": 123,
			},
		},
		"CompanyList": {
			"empty": {},
			"with_filters": {
				"name":      "Test Company",
				"domains":   []string{"example.com"},
				"kind":      "company",
				"page":      1,
				"page_size": 10,
			},
		},
		"CustomerCreate": {
			"minimal": {
				"firstName": "John",
				"lastName":  "Doe",
				"email":     "john.doe@example.com",
			},
		},
		"CustomerGet": {
			"valid": {
				"id": 123,
			},
		},
		"CustomerList": {
			"empty": {},
		},
		"TicketCreate": {
			"minimal": {
				"subject":    "Test Ticket",
				"body":       "Test message",
				"priorityId": 1,
				"statusId":   1,
				"inboxId":    1,
				"customerId": 1,
				"typeId":     1,
				"agentId":    1,
			},
		},
		"TicketUpdate": {
			"minimal": {
				"id":      123,
				"subject": "Updated Ticket",
			},
		},
		"TicketGet": {
			"valid": {
				"id": 123,
			},
		},
		"TicketList": {
			"empty": {},
		},
		"PriorityCreate": {
			"minimal": {
				"name": "High Priority",
			},
		},
		"PriorityGet": {
			"valid": {
				"id": 123,
			},
		},
		"PriorityList": {
			"empty": {},
		},
		"StatusCreate": {
			"minimal": {
				"name": "Open",
			},
		},
		"StatusGet": {
			"valid": {
				"id": 123,
			},
		},
		"StatusList": {
			"empty": {},
		},
		"TagCreate": {
			"minimal": {
				"name": "Important",
			},
		},
		"TagGet": {
			"valid": {
				"id": 123,
			},
		},
		"TagList": {
			"empty": {},
		},
		"TypeCreate": {
			"minimal": {
				"name": "Bug Report",
			},
		},
		"TypeGet": {
			"valid": {
				"id": 123,
			},
		},
		"TypeList": {
			"empty": {},
		},
		"UserGet": {
			"valid": {
				"id": 123,
			},
		},
		"UserList": {
			"empty": {},
		},
		"MessageCreate": {
			"minimal": {
				"ticketID": 123,
				"body":     "Test message",
			},
		},
		"FileCreate": {
			"minimal": {
				"name":     "test.txt",
				"mimeType": "text/plain",
				"data":     "VGVzdCBjb250ZW50", // base64 encoded "Test content"
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
