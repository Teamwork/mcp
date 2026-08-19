package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/teamwork/mcp/internal/twprojects"
)

// customItemsSuite walks the Custom Items tools: creating an item type with a
// text and a dropdown field, then driving records through it by field *name* and
// option *label*, to check field-name resolution, value coercion and the
// twId↔name translation in both directions.
type customItemsSuite struct {
	r *runner

	customItemID int64
	notesID      int64
	notesTwID    string
	statusID     int64
	recordID     int64
	recordIDs    []int64
}

func newCustomItemsSuite(r *runner) suite {
	return &customItemsSuite{r: r}
}

func (s *customItemsSuite) steps() []step {
	return []step{
		{"LIST existing custom items via MCP", s.stepListCustomItems},
		{"CREATE custom item via MCP", s.stepCreateCustomItem},
		{`CREATE field "Notes" (text-short) via MCP`, s.stepCreateNotesField},
		{`CREATE field "Status" (dropdown) via MCP`, s.stepCreateStatusField},
		{"LIST fields via MCP — verify twIds", s.stepListFields},
		{"CREATE record by FIELD NAME (Notes + Status)", s.stepCreateRecordByName},
		{"GET record via MCP — verify twId→name translation", s.stepGetRecord},
		{"UPDATE record by field name (clear section, change Status by label)", s.stepUpdateRecord},
		{"LIST records via MCP — verify translation across a page", s.stepListRecords},
		{"CREATE 2 extra records for bulk delete", s.stepCreateExtraRecords},
		{"NEGATIVE: unknown field name should error clearly", s.stepNegativeUnknownField},
	}
}

func (s *customItemsSuite) artefacts() []string {
	var artefacts []string
	if s.customItemID != 0 {
		artefacts = append(artefacts, fmt.Sprintf("customItemID  = %d", s.customItemID))
	}
	if s.notesID != 0 {
		artefacts = append(artefacts, fmt.Sprintf("notesFieldID  = %d", s.notesID))
	}
	if s.statusID != 0 {
		artefacts = append(artefacts, fmt.Sprintf("statusFieldID = %d", s.statusID))
	}
	if len(s.recordIDs) > 0 {
		artefacts = append(artefacts, fmt.Sprintf("recordIDs     = %v", s.recordIDs))
	}
	return artefacts
}

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

func (s *customItemsSuite) stepListCustomItems(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "list_custom_items",
		twprojects.CustomItemList(s.r.engine), map[string]any{
			"project_id": s.r.projectID,
		})
	return err
}

func (s *customItemsSuite) stepCreateCustomItem(ctx context.Context) error {
	name := fmt.Sprintf("mcp-test-%d", s.r.projectID)
	text, err := s.r.callToolExpectOK(ctx, "create_custom_item",
		twprojects.CustomItemCreate(s.r.engine), map[string]any{
			"project_id":     s.r.projectID,
			"display_name":   name,
			"label_singular": "MCP Record",
			"label_plural":   "MCP Records",
		})
	if err != nil {
		return err
	}
	s.customItemID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract custom item id: %w", err)
	}
	fmt.Printf("  → captured customItemID=%d\n", s.customItemID)
	return nil
}

func (s *customItemsSuite) stepCreateNotesField(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "create_custom_item_field",
		twprojects.CustomItemFieldCreate(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
			"display_name":   "Notes",
			"type":           "text-short",
		})
	if err != nil {
		return err
	}
	s.notesID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract field id: %w", err)
	}
	fmt.Printf("  → captured notesFieldID=%d\n", s.notesID)
	return nil
}

func (s *customItemsSuite) stepCreateStatusField(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "create_custom_item_field",
		twprojects.CustomItemFieldCreate(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
			"display_name":   "Status",
			"type":           "dropdown",
			"tw_type":        "status",
			"options": []map[string]any{
				{"label": "Active", "color": "#22c55e"},
				{"label": "Pending", "color": "#facc15"},
				{"label": "Closed", "color": "#94a3b8"},
			},
		})
	if err != nil {
		return err
	}
	s.statusID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract field id: %w", err)
	}
	fmt.Printf("  → captured statusFieldID=%d\n", s.statusID)
	return nil
}

func (s *customItemsSuite) stepListFields(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "list_custom_item_fields",
		twprojects.CustomItemFieldList(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
		})
	if err != nil {
		return err
	}
	// Pull the Notes field twId out so we can compare against the value
	// coming back in the record step.
	var listResp struct {
		CustomItemFields []struct {
			ID          int64  `json:"id"`
			TwID        string `json:"twId"`
			DisplayName string `json:"displayName"`
		} `json:"customItemFields"`
	}
	if err := json.Unmarshal([]byte(text), &listResp); err == nil {
		for _, field := range listResp.CustomItemFields {
			if field.DisplayName == "Notes" {
				s.notesTwID = field.TwID
				fmt.Printf("  → captured notesTwID=%q\n", s.notesTwID)
			}
		}
	}
	return nil
}

func (s *customItemsSuite) stepCreateRecordByName(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "create_custom_item_record",
		twprojects.CustomItemRecordCreate(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
			"name":           "Acme Inc",
			"field_values": []map[string]any{
				{"field_name": "Notes", "value": "initial contact"},
				// Pass the option by LABEL — the handler should resolve it
				// to the option twId before sending.
				{"field_name": "Status", "value": "Active"},
			},
		})
	if err != nil {
		return err
	}
	s.recordID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract record id: %w", err)
	}
	s.recordIDs = append(s.recordIDs, s.recordID)
	fmt.Printf("  → captured recordID=%d\n", s.recordID)
	return nil
}

func (s *customItemsSuite) stepGetRecord(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "get_custom_item_record",
		twprojects.CustomItemRecordGet(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
			"id":             s.recordID,
		})
	if err != nil {
		return err
	}
	// Sanity: confirm field values came back keyed by name, not twId.
	if strings.Contains(text, `"Notes"`) {
		fmt.Println("  ✓ field values keyed by name (Notes)")
	} else {
		fmt.Println("  ! WARNING: Notes not present in response by display name")
	}
	if strings.Contains(text, `"Active"`) {
		fmt.Println("  ✓ Status value translated back to label (Active)")
	} else {
		fmt.Println("  ! WARNING: Status label not present in response")
	}
	if s.notesTwID != "" && strings.Contains(text, s.notesTwID) {
		fmt.Printf("  ! WARNING: raw twId %q leaked into response\n", s.notesTwID)
	}
	return nil
}

func (s *customItemsSuite) stepUpdateRecord(ctx context.Context) error {
	_, err := s.r.callToolExpectOK(ctx, "update_custom_item_record",
		twprojects.CustomItemRecordUpdate(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
			"id":             s.recordID,
			"name":           "Acme Inc (updated via MCP)",
			"clear_section":  true,
			"field_values": []map[string]any{
				{"field_name": "Status", "value": "Pending"}, // by label
				{"field_name": "Notes", "value": "follow-up next week"},
			},
		})
	return err
}

func (s *customItemsSuite) stepListRecords(ctx context.Context) error {
	text, err := s.r.callToolExpectOK(ctx, "list_custom_item_records",
		twprojects.CustomItemRecordList(s.r.engine), map[string]any{
			"custom_item_id": s.customItemID,
		})
	if err != nil {
		return err
	}
	if strings.Contains(text, `"Notes"`) && strings.Contains(text, `"Status"`) {
		fmt.Println("  ✓ list response uses display names for field keys")
	}
	return nil
}

func (s *customItemsSuite) stepCreateExtraRecords(ctx context.Context) error {
	for i := range 2 {
		text, err := s.r.callToolExpectOK(ctx, "create_custom_item_record (extra)",
			twprojects.CustomItemRecordCreate(s.r.engine), map[string]any{
				"custom_item_id": s.customItemID,
				"name":           fmt.Sprintf("bulk-target-%d", i+1),
			})
		if err != nil {
			return err
		}
		id, err := extractTrailingID(text)
		if err != nil {
			return fmt.Errorf("extract extra record id: %w", err)
		}
		s.recordIDs = append(s.recordIDs, id)
	}
	return nil
}

func (s *customItemsSuite) stepNegativeUnknownField(ctx context.Context) error {
	text, isError, err := s.r.callTool(ctx, twprojects.CustomItemRecordCreate(s.r.engine), map[string]any{
		"custom_item_id": s.customItemID,
		"name":           "negative-test",
		"field_values": []map[string]any{
			{"field_name": "NoSuchField", "value": "x"},
		},
	})
	if err != nil {
		return fmt.Errorf("negative test: %w", err)
	}
	if !isError {
		return fmt.Errorf("expected an error result for unknown field, got success: %s", text)
	}
	fmt.Printf("  ✓ unknown-field error surfaced: %s\n", strings.TrimSpace(text))
	return nil
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

// cleanup removes the records, then the fields, then the item type itself:
// deleting the item type first would orphan the rest.
func (s *customItemsSuite) cleanup(ctx context.Context) {
	if s.customItemID == 0 {
		return
	}

	if len(s.recordIDs) > 0 {
		fmt.Printf("  bulk-delete %d records via MCP\n", len(s.recordIDs))
		s.r.callToolIgnoreError(ctx, "bulk delete records",
			twprojects.CustomItemRecordBulkDelete(s.r.engine), map[string]any{
				"custom_item_id": s.customItemID,
				"ids":            asAnyInts(s.recordIDs),
			})
	}

	for _, fieldID := range []int64{s.notesID, s.statusID} {
		if fieldID == 0 {
			continue
		}
		fmt.Printf("  delete field %d via MCP\n", fieldID)
		s.r.callToolIgnoreError(ctx, fmt.Sprintf("delete field %d", fieldID),
			twprojects.CustomItemFieldDelete(s.r.engine), map[string]any{
				"custom_item_id": s.customItemID,
				"id":             fieldID,
			})
	}

	fmt.Printf("  delete custom item %d via MCP\n", s.customItemID)
	s.r.callToolIgnoreError(ctx, "delete custom item",
		twprojects.CustomItemDelete(s.r.engine), map[string]any{
			"id": s.customItemID,
		})
}
