// mcptest walks through the Custom Items MCP tool handlers against a real
// Teamwork.com site. It bypasses the MCP server transport and invokes each
// tool's Handler directly with the same JSON payload an LLM would send, so
// you can see end-to-end behaviour — including field-name resolution, value
// coercion, twId↔name translation and the schema cache — without standing
// up an MCP server or LLM client.
//
// Configuration is read from .env (or the path given by -config). The auth
// mode (basic vs bearer) is auto-detected from the token prefix; override
// with TWAPI_AUTH. Same .env shape as sdktest.
//
// Usage:
//
//	cd c:\programming\mcptest
//	copy ..\sdktest\.env .env   # if you don't already have one here
//	go run .
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/session"
)

func main() {
	configPath := preScanConfig(os.Args[1:], ".env")
	if err := loadEnvFile(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", configPath, err)
		os.Exit(2)
	}

	flag.StringVar(&configPath, "config", configPath, "Path to .env file (ignored if missing)")
	server := flag.String("server", os.Getenv("TWAPI_SERVER"), "Teamwork base URL (env TWAPI_SERVER)")
	token := flag.String("token", os.Getenv("TWAPI_TOKEN"), "Bearer token / API key (env TWAPI_TOKEN)")
	projectID := flag.Int64("project", envInt64("PROJECT_ID"), "Project ID to test on (env PROJECT_ID)")
	keep := flag.Bool("keep", false, "Don't delete created artefacts so you can inspect them in the UI")
	step := flag.Bool("step", false, "Pause for ENTER between steps")
	flag.Parse()

	if *server == "" || *token == "" || *projectID == 0 {
		fmt.Fprintln(os.Stderr, "server, token and project are all required (set in .env, env vars, or flags)")
		flag.Usage()
		os.Exit(2)
	}
	*server = strings.TrimSuffix(*server, "/")

	authMode := strings.ToLower(strings.TrimSpace(os.Getenv("TWAPI_AUTH")))
	if authMode == "" {
		if strings.HasPrefix(*token, "twp_") {
			authMode = "basic"
		} else {
			authMode = "bearer"
		}
	}
	var sess twapi.Session
	switch authMode {
	case "basic":
		fmt.Println("auth: HTTP Basic (personal API key)")
		sess = session.NewBasicAuth(*token, "x", *server)
	case "bearer":
		fmt.Println("auth: Bearer token")
		sess = session.NewBearerToken(*token, *server)
	default:
		fmt.Fprintf(os.Stderr, "unknown TWAPI_AUTH %q (want basic|bearer)\n", authMode)
		os.Exit(2)
	}
	engine := twapi.NewEngine(sess)

	r := &runner{
		engine:    engine,
		projectID: *projectID,
		keep:      *keep,
		step:      *step,
		stdin:     bufio.NewReader(os.Stdin),
		ctx:       context.Background(),
	}
	if err := r.run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("DONE.")
}

type runner struct {
	engine    *twapi.Engine
	projectID int64
	keep      bool
	step      bool
	stdin     *bufio.Reader
	ctx       context.Context

	customItemID int64
	statusID     int64
	notesID      int64
	notesTwID    string
	recordID     int64
	recordIDs    []int64
}

func (r *runner) run() error {
	defer r.cleanup()

	steps := []struct {
		label string
		fn    func() error
	}{
		{"LIST existing custom items via MCP", r.stepListCustomItems},
		{"CREATE custom item via MCP", r.stepCreateCustomItem},
		{`CREATE field "Notes" (text-short) via MCP`, r.stepCreateNotesField},
		{`CREATE field "Status" (dropdown) via MCP`, r.stepCreateStatusField},
		{"LIST fields via MCP — verify twIds", r.stepListFields},
		{"CREATE record by FIELD NAME (Notes + Status)", r.stepCreateRecordByName},
		{"GET record via MCP — verify twId→name translation", r.stepGetRecord},
		{"UPDATE record by field name (clear section, change Status by label)", r.stepUpdateRecord},
		{"LIST records via MCP — verify translation across a page", r.stepListRecords},
		{"CREATE 2 extra records for bulk delete", r.stepCreateExtraRecords},
		{"NEGATIVE: unknown field name should error clearly", r.stepNegativeUnknownField},
	}
	for _, s := range steps {
		fmt.Printf("\n=== %s ===\n", s.label)
		if r.step {
			fmt.Print("press ENTER to continue: ")
			_, _ = r.stdin.ReadString('\n')
		}
		if err := s.fn(); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MCP invocation helpers
// ---------------------------------------------------------------------------

// callTool marshals args to JSON and invokes the tool's handler directly.
// Returns the structured text result and any error from the handler, plus a
// bool indicating whether the result was an error result (IsError).
func (r *runner) callTool(tool toolsets.ToolWrapper, args map[string]any) (text string, isError bool, err error) {
	raw, err := json.Marshal(args)
	if err != nil {
		return "", false, fmt.Errorf("marshal args: %w", err)
	}
	request := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      tool.Tool.Name,
			Arguments: json.RawMessage(raw),
		},
	}
	result, err := tool.Handler(r.ctx, request)
	if err != nil {
		return "", false, err
	}
	if result == nil {
		return "", false, errors.New("nil result")
	}
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			return text.Text, result.IsError, nil
		}
	}
	return "", result.IsError, errors.New("no text content in result")
}

// callToolExpectOK calls a tool, prints the response, and returns an error
// if the handler returned err or an IsError result.
func (r *runner) callToolExpectOK(label string, tool toolsets.ToolWrapper, args map[string]any) (string, error) {
	text, isError, err := r.callTool(tool, args)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	if isError {
		return "", fmt.Errorf("%s returned error result: %s", label, text)
	}
	prettyPrint(text)
	return text, nil
}

func prettyPrint(text string) {
	// If the body is JSON, pretty-print it; otherwise emit raw.
	var anyVal any
	if err := json.Unmarshal([]byte(text), &anyVal); err == nil {
		formatted, _ := json.MarshalIndent(anyVal, "  ", "  ")
		fmt.Printf("  %s\n", string(formatted))
		return
	}
	fmt.Printf("  %s\n", text)
}

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

func (r *runner) stepListCustomItems() error {
	_, err := r.callToolExpectOK("list_custom_items", twprojects.CustomItemList(r.engine), map[string]any{
		"project_id": r.projectID,
	})
	return err
}

func (r *runner) stepCreateCustomItem() error {
	name := fmt.Sprintf("MCPTest-%d", r.projectID)
	text, err := r.callToolExpectOK("create_custom_item", twprojects.CustomItemCreate(r.engine), map[string]any{
		"project_id":     r.projectID,
		"display_name":   name,
		"label_singular": "MCP Record",
		"label_plural":   "MCP Records",
	})
	if err != nil {
		return err
	}
	r.customItemID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract custom item id: %w", err)
	}
	fmt.Printf("  → captured customItemID=%d\n", r.customItemID)
	return nil
}

func (r *runner) stepCreateNotesField() error {
	text, err := r.callToolExpectOK("create_custom_item_field", twprojects.CustomItemFieldCreate(r.engine), map[string]any{
		"custom_item_id": r.customItemID,
		"display_name":   "Notes",
		"type":           "text-short",
	})
	if err != nil {
		return err
	}
	r.notesID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract field id: %w", err)
	}
	fmt.Printf("  → captured notesFieldID=%d\n", r.notesID)
	return nil
}

func (r *runner) stepCreateStatusField() error {
	text, err := r.callToolExpectOK("create_custom_item_field", twprojects.CustomItemFieldCreate(r.engine), map[string]any{
		"custom_item_id": r.customItemID,
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
	r.statusID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract field id: %w", err)
	}
	fmt.Printf("  → captured statusFieldID=%d\n", r.statusID)
	return nil
}

func (r *runner) stepListFields() error {
	text, err := r.callToolExpectOK("list_custom_item_fields",
		twprojects.CustomItemFieldList(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
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
				r.notesTwID = field.TwID
				fmt.Printf("  → captured notesTwID=%q\n", r.notesTwID)
			}
		}
	}
	return nil
}

func (r *runner) stepCreateRecordByName() error {
	text, err := r.callToolExpectOK("create_custom_item_record",
		twprojects.CustomItemRecordCreate(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
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
	r.recordID, err = extractTrailingID(text)
	if err != nil {
		return fmt.Errorf("extract record id: %w", err)
	}
	r.recordIDs = append(r.recordIDs, r.recordID)
	fmt.Printf("  → captured recordID=%d\n", r.recordID)
	return nil
}

func (r *runner) stepGetRecord() error {
	text, err := r.callToolExpectOK("get_custom_item_record",
		twprojects.CustomItemRecordGet(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
			"id":             r.recordID,
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
	if r.notesTwID != "" && strings.Contains(text, r.notesTwID) {
		fmt.Printf("  ! WARNING: raw twId %q leaked into response\n", r.notesTwID)
	}
	return nil
}

func (r *runner) stepUpdateRecord() error {
	_, err := r.callToolExpectOK("update_custom_item_record",
		twprojects.CustomItemRecordUpdate(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
			"id":             r.recordID,
			"name":           "Acme Inc (updated via MCP)",
			"clear_section":  true,
			"field_values": []map[string]any{
				{"field_name": "Status", "value": "Pending"}, // by label
				{"field_name": "Notes", "value": "follow-up next week"},
			},
		})
	return err
}

func (r *runner) stepListRecords() error {
	text, err := r.callToolExpectOK("list_custom_item_records",
		twprojects.CustomItemRecordList(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
		})
	if err != nil {
		return err
	}
	if strings.Contains(text, `"Notes"`) && strings.Contains(text, `"Status"`) {
		fmt.Println("  ✓ list response uses display names for field keys")
	}
	return nil
}

func (r *runner) stepCreateExtraRecords() error {
	for i := 0; i < 2; i++ {
		text, err := r.callToolExpectOK("create_custom_item_record (extra)",
			twprojects.CustomItemRecordCreate(r.engine), map[string]any{
				"custom_item_id": r.customItemID,
				"name":           fmt.Sprintf("bulk-target-%d", i+1),
			})
		if err != nil {
			return err
		}
		id, err := extractTrailingID(text)
		if err != nil {
			return fmt.Errorf("extract extra record id: %w", err)
		}
		r.recordIDs = append(r.recordIDs, id)
	}
	return nil
}

func (r *runner) stepNegativeUnknownField() error {
	tool := twprojects.CustomItemRecordCreate(r.engine)
	text, isError, err := r.callTool(tool, map[string]any{
		"custom_item_id": r.customItemID,
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

func (r *runner) cleanup() {
	if r.keep {
		fmt.Println("\n--- KEEP set, leaving artefacts in place ---")
		if r.customItemID != 0 {
			fmt.Printf("  customItemID = %d\n", r.customItemID)
		}
		if r.notesID != 0 {
			fmt.Printf("  notesFieldID = %d\n", r.notesID)
		}
		if r.statusID != 0 {
			fmt.Printf("  statusFieldID = %d\n", r.statusID)
		}
		if len(r.recordIDs) > 0 {
			fmt.Printf("  recordIDs    = %v\n", r.recordIDs)
		}
		return
	}

	fmt.Println("\n--- cleanup ---")

	if len(r.recordIDs) > 0 && r.customItemID != 0 {
		fmt.Printf("  bulk-delete %d records via MCP\n", len(r.recordIDs))
		_, isError, err := r.callTool(twprojects.CustomItemRecordBulkDelete(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
			"ids":            asAnyInts(r.recordIDs),
		})
		if err != nil || isError {
			fmt.Fprintf(os.Stderr, "  ! bulk delete records failed: err=%v\n", err)
		}
	}

	for _, fieldID := range []int64{r.notesID, r.statusID} {
		if fieldID == 0 || r.customItemID == 0 {
			continue
		}
		fmt.Printf("  delete field %d via MCP\n", fieldID)
		_, isError, err := r.callTool(twprojects.CustomItemFieldDelete(r.engine), map[string]any{
			"custom_item_id": r.customItemID,
			"id":             fieldID,
		})
		if err != nil || isError {
			fmt.Fprintf(os.Stderr, "  ! delete field %d failed: err=%v\n", fieldID, err)
		}
	}

	if r.customItemID != 0 {
		fmt.Printf("  delete custom item %d via MCP\n", r.customItemID)
		_, isError, err := r.callTool(twprojects.CustomItemDelete(r.engine), map[string]any{
			"id": r.customItemID,
		})
		if err != nil || isError {
			fmt.Fprintf(os.Stderr, "  ! delete custom item failed: err=%v\n", err)
		}
	}
}

// asAnyInts converts []int64 to []any (what JSON-encoded numeric arrays look
// like after a typical map[string]any unmarshal).
func asAnyInts(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// extractTrailingID pulls the last whitespace-separated token from the text,
// assuming a result like "Custom item created successfully with ID 1234".
func extractTrailingID(text string) (int64, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, errors.New("empty text")
	}
	parts := strings.Fields(trimmed)
	last := parts[len(parts)-1]
	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("trailing token %q is not an integer: %w", last, err)
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// .env loader (same shape as sdktest)
// ---------------------------------------------------------------------------

func preScanConfig(args []string, fallback string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-config" || arg == "--config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-config=") || strings.HasPrefix(arg, "--config="):
			return arg[strings.IndexByte(arg, '=')+1:]
		case arg == "--":
			return fallback
		}
	}
	return fallback
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return fmt.Errorf("%s line %d: expected KEY=VALUE", path, lineNo)
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if n := len(value); n >= 2 {
			first, last := value[0], value[n-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				value = value[1 : n-1]
			}
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s line %d: setenv %s: %w", path, lineNo, key, err)
		}
	}
	return scanner.Err()
}

func envInt64(key string) int64 {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
