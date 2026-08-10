package twprojects_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk"
)

// taskSkillsAndRolesEngine serves the endpoints the task skills and roles prompt
// walks, recording every path it requests. The tasklist reports a different
// project than the task's tasklist relationship hints at, so the recorded
// project path tells which of the two the prompt trusted.
func taskSkillsAndRolesEngine(t *testing.T, tasklistMeta string) (*twapi.Engine, *[]string) {
	t.Helper()

	var mu sync.Mutex
	var paths []string

	bodies := map[string]string{
		"/v3/tasks/": fmt.Sprintf(`{"task":{"id":1,"name":"Ship it","tasklist":{"id":12,"type":"tasklists"%s}}}`,
			tasklistMeta),
		"/v3/tasklists/": `{"tasklist":{"id":12,"name":"Sprint 1","project":{"id":999,"type":"projects"}}}`,
		"/v3/projects/":  `{"project":{"id":5,"name":"Apollo"}}`,
		"/v3/skills":     `{"skills":[{"id":1,"name":"Go"}],"meta":{"page":{"hasMore":false}}}`,
		"/v3/jobroles":   `{"jobRoles":[{"id":2,"name":"Engineer"}],"meta":{"page":{"hasMore":false}}}`,
	}

	engine := twapi.NewEngine(testutil.ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
			mu.Lock()
			paths = append(paths, req.URL.Path)
			mu.Unlock()

			body := "{}"
			for match, candidate := range bodies {
				if strings.Contains(req.URL.Path, match) {
					body = candidate
					break
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(http.StatusOK),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		})
	}))

	return engine, &paths
}

func runTaskSkillsAndRolesPrompt(t *testing.T, engine *twapi.Engine) *mcp.GetPromptResult {
	t.Helper()

	prompt := twprojects.TaskSkillsAndRolesPrompt(engine)
	result, err := prompt.Handler(context.Background(), &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "twprojects_task_skills_and_roles",
			Arguments: map[string]string{"task_id": "1"},
		},
	})
	if err != nil {
		t.Fatalf("prompt handler failed: %v", err)
	}
	return result
}

func TestTaskSkillsAndRolesPromptUsesTasklistProjectHint(t *testing.T) {
	engine, paths := taskSkillsAndRolesEngine(t, `,"meta":{"projectId":5}`)
	result := runTaskSkillsAndRolesPrompt(t, engine)

	// The hint says project 5; the tasklist says 999. Trusting the hint means the
	// project fetch no longer waits on the tasklist response.
	if !containsPath(*paths, "/projects/api/v3/projects/5.json") {
		t.Errorf("expected the project to be loaded from the tasklist hint, got paths %v", *paths)
	}
	if containsPath(*paths, "/projects/api/v3/projects/999.json") {
		t.Errorf("expected the tasklist project not to be used, got paths %v", *paths)
	}
	assertPromptMentions(t, result, "Project Name: Apollo", "Tasklist Name: Sprint 1")
}

func TestTaskSkillsAndRolesPromptFallsBackWithoutHint(t *testing.T) {
	engine, paths := taskSkillsAndRolesEngine(t, "")
	result := runTaskSkillsAndRolesPrompt(t, engine)

	// Without the hint the prompt has to resolve the project through the tasklist.
	if !containsPath(*paths, "/projects/api/v3/projects/999.json") {
		t.Errorf("expected the project to be resolved through the tasklist, got paths %v", *paths)
	}
	assertPromptMentions(t, result, "Tasklist Name: Sprint 1")
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}

func assertPromptMentions(t *testing.T, result *mcp.GetPromptResult, wants ...string) {
	t.Helper()

	var text strings.Builder
	for _, message := range result.Messages {
		if content, ok := message.Content.(*mcp.TextContent); ok {
			text.WriteString(content.Text)
		}
	}
	for _, want := range wants {
		if !strings.Contains(text.String(), want) {
			t.Errorf("expected the prompt to contain %q", want)
		}
	}
}
