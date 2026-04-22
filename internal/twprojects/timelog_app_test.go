package twprojects_test

import (
	"net/http"
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

	uiMetaRaw, ok := content.Meta["ui"]
	if !ok {
		t.Fatal("expected resource content to include _meta.ui")
	}
	uiMeta, ok := uiMetaRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected _meta.ui to be map[string]any, got %T", uiMetaRaw)
	}
	if _, ok := uiMeta["csp"].(map[string]any); !ok {
		t.Fatalf("expected _meta.ui.csp to be map[string]any, got %T", uiMeta["csp"])
	}
	if _, ok := content.Meta["openai/widgetDescription"].(string); !ok {
		t.Fatalf("expected _meta.openai/widgetDescription to be present, got %#v", content.Meta["openai/widgetDescription"])
	}
	if _, ok := content.Meta["openai/widgetCSP"].(map[string]any); !ok {
		t.Fatalf("expected _meta.openai/widgetCSP to be map[string]any, got %T", content.Meta["openai/widgetCSP"])
	}
}
