package twprojects_test

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/twprojects"
)

func connectProjectsClientSession(t *testing.T, mcpServer *mcp.Server) *mcp.ClientSession {
	t.Helper()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	_, err := mcpServer.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect test server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)
	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect test client: %v", err)
	}
	return clientSession
}

func TestTimelogCreateHasMCPAppsMeta(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timelog":{"id":123}}`))
	clientSession := connectProjectsClientSession(t, mcpServer)
	defer clientSession.Close() //nolint:errcheck

	tools, err := clientSession.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	var createTimelogTool *mcp.Tool
	for _, tool := range tools.Tools {
		if tool.Name == twprojects.MethodTimelogCreate.String() {
			createTimelogTool = tool
			break
		}
	}
	if createTimelogTool == nil {
		t.Fatal("twprojects-create_timelog tool not found")
		return
	}

	uiMetaRaw, ok := createTimelogTool.Meta["ui"]
	if !ok {
		t.Fatalf("expected _meta.ui on %s", twprojects.MethodTimelogCreate)
	}
	uiMeta, ok := uiMetaRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected _meta.ui to be map[string]any, got %T", uiMetaRaw)
	}

	resourceURI, ok := uiMeta["resourceUri"].(string)
	if !ok || resourceURI == "" {
		t.Fatalf("expected _meta.ui.resourceUri to be non-empty string, got %#v", uiMeta["resourceUri"])
	}
	if resourceURI != "ui://teamwork/timelog-create" {
		t.Fatalf("unexpected resource URI %q", resourceURI)
	}

	openAIOutputTemplate, ok := createTimelogTool.Meta["openai/outputTemplate"].(string)
	if !ok || openAIOutputTemplate == "" {
		t.Fatalf("expected _meta.openai/outputTemplate to be non-empty string, got %#v",
			createTimelogTool.Meta["openai/outputTemplate"])
	}
	if openAIOutputTemplate != resourceURI {
		t.Fatalf("expected openai/outputTemplate to match ui.resourceUri, got %q and %q", openAIOutputTemplate, resourceURI)
	}
}

func TestTimelogCreateResourceRead(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timelog":{"id":123}}`))
	clientSession := connectProjectsClientSession(t, mcpServer)
	defer clientSession.Close() //nolint:errcheck

	result, err := clientSession.ReadResource(t.Context(), &mcp.ReadResourceParams{
		URI: "ui://teamwork/timelog-create",
	})
	if err != nil {
		t.Fatalf("failed to read timelog resource: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected exactly 1 resource content block, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.MIMEType != "text/html;profile=mcp-app" {
		t.Fatalf("unexpected mimeType: %q", content.MIMEType)
	}
	if !strings.Contains(content.Text, "Create Timelog") {
		t.Fatalf("expected embedded HTML to contain heading, got: %q", content.Text)
	}
	// no JS harness, so guard the required ui/initialize fields at source level.
	for _, fragment := range []string{"appInfo:", "appCapabilities:", `availableDisplayModes: ["inline"]`} {
		if !strings.Contains(content.Text, fragment) {
			t.Errorf("expected embedded HTML ui/initialize handshake to declare %s", fragment)
		}
	}
	// same for the payload shapes the spec pins on the host notifications.
	for _, fragment := range []string{
		"params.arguments",                    // ui/notifications/tool-input
		"(params && params.result) || params", // ui/notifications/tool-result
		"ui/notifications/size-changed",       // flexible-dimension hosts size from this
	} {
		if !strings.Contains(content.Text, fragment) {
			t.Errorf("expected embedded HTML to handle %s", fragment)
		}
	}

	uiMetaRaw, ok := content.Meta["ui"]
	if !ok {
		t.Fatal("expected resource content to include _meta.ui")
	}
	uiMeta, ok := uiMetaRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected _meta.ui to be map[string]any, got %T", uiMetaRaw)
	}
	csp, ok := uiMeta["csp"].(map[string]any)
	if !ok {
		t.Fatalf("expected _meta.ui.csp to be map[string]any, got %T", uiMeta["csp"])
	}
	// the standard MCP Apps surface uses camelCase field names.
	for _, field := range []string{"connectDomains", "resourceDomains", "frameDomains", "baseUriDomains"} {
		if _, ok := csp[field]; !ok {
			t.Errorf("expected _meta.ui.csp to declare %q, got %#v", field, csp)
		}
	}
	if _, ok := content.Meta["openai/widgetDescription"].(string); !ok {
		t.Fatalf("expected _meta.openai/widgetDescription to be present, got %#v", content.Meta["openai/widgetDescription"])
	}
	openAICSP, ok := content.Meta["openai/widgetCSP"].(map[string]any)
	if !ok {
		t.Fatalf("expected _meta.openai/widgetCSP to be map[string]any, got %T", content.Meta["openai/widgetCSP"])
	}
	// the legacy ChatGPT key only recognises snake_case names.
	for _, field := range []string{"connect_domains", "resource_domains", "frame_domains", "redirect_domains"} {
		if _, ok := openAICSP[field]; !ok {
			t.Errorf("expected _meta.openai/widgetCSP to declare %q, got %#v", field, openAICSP)
		}
	}
}

// TestTimelogCreateAppCallsRegisteredTools guards the tool names the app calls
// from the iframe. A name no tool answers to is refused by the host as outside
// the app's trusted tool scope, which surfaces as a dead control in the form.
func TestTimelogCreateAppCallsRegisteredTools(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"timelog":{"id":123}}`))
	clientSession := connectProjectsClientSession(t, mcpServer)
	defer clientSession.Close() //nolint:errcheck

	tools, err := clientSession.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	registered := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		registered[tool.Name] = true
	}

	result, err := clientSession.ReadResource(t.Context(), &mcp.ReadResourceParams{
		URI: "ui://teamwork/timelog-create",
	})
	if err != nil {
		t.Fatalf("failed to read timelog resource: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("expected exactly 1 resource content block, got %d", len(result.Contents))
	}

	called := regexp.MustCompile(`callTool\("([^"]+)"`).FindAllStringSubmatch(result.Contents[0].Text, -1)
	if len(called) == 0 {
		t.Fatal("expected the app to call at least one tool")
	}
	for _, match := range called {
		if !registered[match[1]] {
			t.Errorf("app calls %q, which no registered tool answers to", match[1])
		}
	}
}

// TestTimelogCreateAppNeverLogsTwice runs the app's script in node against a
// fake host. The host renders the form for the model's own create, so the form
// must not send that create again, and a retry it does send must match it.
func TestTimelogCreateAppNeverLogsTwice(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	output, err := exec.CommandContext(t.Context(), node,
		"testdata/timelog_app_harness.js", "apps/timelog_create.html").Output()
	if err != nil {
		t.Fatalf("failed to run the app harness: %v", err)
	}

	type observation struct {
		Creates        []map[string]any `json:"creates"`
		Project        string           `json:"project"`
		Task           string           `json:"task"`
		Date           string           `json:"date"`
		SubmitDisabled bool             `json:"submitDisabled"`
		Locked         bool             `json:"locked"`
		Status         string           `json:"status"`
	}
	var observed map[string]observation
	if err := json.Unmarshal(output, &observed); err != nil {
		t.Fatalf("failed to decode harness output %q: %v", output, err)
	}

	tests := []struct {
		scenario    string
		creates     int
		locked      bool
		submittable bool
		check       func(t *testing.T, o observation)
	}{
		{scenario: "host-success", creates: 0, locked: true, check: func(t *testing.T, o observation) {
			// a task_id alone must not fall back to the first project by name.
			if o.Project != "12345" || o.Task != "777" {
				t.Errorf("expected the task's own project and the task, got %q and %q", o.Project, o.Task)
			}
		}},
		{scenario: "submit-during-host-call", creates: 0},
		{scenario: "host-result-without-content", creates: 0, locked: true},
		{scenario: "host-error-then-retry", creates: 1, locked: true, check: func(t *testing.T, o observation) {
			create := o.Creates[0]
			for key, want := range map[string]any{
				"is_utc": true, "task_id": 777.0, "user_id": 999.0, "time": "13:00:00",
			} {
				if create[key] != want {
					t.Errorf("expected the retry to send %s=%v, got %v", key, want, create[key])
				}
			}
			if _, ok := create["project_id"]; ok {
				t.Errorf("expected the retry to send no project_id, got %v", create["project_id"])
			}
			if tags, _ := create["tag_ids"].([]any); len(tags) != 1 {
				t.Errorf("expected the retry to keep the model's tag_ids, got %v", create["tag_ids"])
			}
		}},
		{scenario: "host-result-only", creates: 0, locked: true},
		{scenario: "host-error-without-text", creates: 0, submittable: true, check: func(t *testing.T, o observation) {
			if o.Status != "The timelog was not created." {
				t.Errorf("expected an error without text to read as not created, got %q", o.Status)
			}
		}},
		{scenario: "host-cancelled", creates: 1, locked: true},
		{scenario: "host-cancelled-message", creates: 0, submittable: true, check: func(t *testing.T, o observation) {
			// a cancel can follow a create the server already ran.
			if !strings.Contains(o.Status, "may or may not have been created") {
				t.Errorf("expected a cancel to warn the timelog may exist, got %q", o.Status)
			}
		}},
		{scenario: "input-after-lock", creates: 0, locked: true, check: func(t *testing.T, o observation) {
			if o.Date != "2026-10-23" {
				t.Errorf("expected a locked form to keep the created date, got %q", o.Date)
			}
		}},
		{scenario: "own-create-times-out", creates: 1, locked: true, check: func(t *testing.T, o observation) {
			if !strings.Contains(o.Status, "may have been created") {
				t.Errorf("expected a timed-out create to warn it may exist, got %q", o.Status)
			}
		}},
		{scenario: "own-create-host-error", creates: 1, locked: true, check: func(t *testing.T, o observation) {
			if !strings.Contains(o.Status, "may have been created") {
				t.Errorf("expected a host error on the create to warn it may exist, got %q", o.Status)
			}
		}},
		{scenario: "own-create-times-out-then-echo", creates: 1, locked: true, check: func(t *testing.T, o observation) {
			if o.Status != "Timelog created successfully with ID 123" {
				t.Errorf("expected the host's late result to settle the status, got %q", o.Status)
			}
		}},
		{scenario: "late-load-error", creates: 0, locked: true, check: func(t *testing.T, o observation) {
			if o.Status != "Timelog created successfully with ID 123" {
				t.Errorf("expected a late load error to keep the created status, got %q", o.Status)
			}
		}},
		{scenario: "load-error-without-text", creates: 0, submittable: true, check: func(t *testing.T, o observation) {
			if strings.Contains(o.Status, "timelog") {
				t.Errorf("expected a failed project load not to talk about the timelog, got %q", o.Status)
			}
		}},
		{scenario: "utc-without-time", creates: 1, locked: true, check: func(t *testing.T, o observation) {
			// the time field kept its local default, so it is not UTC.
			if o.Creates[0]["is_utc"] != false {
				t.Errorf("expected is_utc=false without a time, got %v", o.Creates[0]["is_utc"])
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			o, ok := observed[tt.scenario]
			if !ok {
				t.Fatalf("harness reported no %q scenario", tt.scenario)
			}
			if len(o.Creates) != tt.creates {
				t.Fatalf("expected %d create_timelog calls from the app, got %d: %v", tt.creates, len(o.Creates), o.Creates)
			}
			if o.Locked != tt.locked {
				t.Errorf("expected locked=%t, got %t (status %q)", tt.locked, o.Locked, o.Status)
			}
			if o.SubmitDisabled == tt.submittable {
				t.Errorf("expected submittable=%t, got submit disabled=%t (status %q)", tt.submittable, o.SubmitDisabled, o.Status)
			}
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}
