// Command mcp-vertex-check validates the published tool schemas against the
// subset of JSON Schema that Vertex AI — and therefore Gemini Enterprise —
// can parse. It answers "would Gemini load these tools?" locally, instead of
// through a customer retrying a connector.
//
// The rules come from the Vertex Schema message
// (google/cloud/aiplatform/v1/openapi.proto): see vertex.go. Errors contradict
// that definition and cannot be parsed; warnings are permitted by it but have
// been reported rejected by Gemini clients.
//
// By default it checks what a Gemini client is actually served — the rewritten
// copy from helpers.VertexTools — because that is the surface that has to be
// clean. The published surface cannot be: its output schemas declare real nulls.
//
// Usage:
//
//	go run ./cmd/mcp-vertex-check                     # what a Gemini client gets; must be clean
//	go run ./cmd/mcp-vertex-check -published          # the raw published surface; will not be clean
//	go run ./cmd/mcp-vertex-check -warn               # include warnings
//	go run ./cmd/mcp-vertex-check -input-only         # ignore output schemas
//	go run ./cmd/mcp-vertex-check -json               # machine-readable
//	go run ./cmd/mcp-vertex-check -tools=live.json    # a captured tools/list result, as-is
//
// It exits non-zero when anything is reported, so the default mode works as a
// CI gate.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
)

func main() {
	var (
		toolsFile = flag.String("tools", "",
			"a captured tools/list result to check instead of the local surface ('-' for stdin)")
		withWarn  = flag.Bool("warn", false, "report warnings as well as errors")
		inputOnly = flag.Bool("input-only", false, "check inputSchema only, leaving output schemas out")
		asJSON    = flag.Bool("json", false, "write findings as JSON")
		published = flag.Bool("published", false,
			"check the raw published surface instead of what a Gemini client is served")
	)
	flag.Parse()

	tools, err := loadTools(*toolsFile, !*published)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-vertex-check: %v\n", err)
		os.Exit(2)
	}

	findings := CheckSurface(len(tools))
	for _, tool := range tools {
		output := tool.outputSchema
		if *inputOnly {
			output = nil
		}
		findings = append(findings, CheckTool(tool.name, tool.inputSchema, output)...)
	}
	if !*withWarn {
		findings = onlyErrors(findings)
	}

	if *asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
			fmt.Fprintf(os.Stderr, "mcp-vertex-check: %v\n", err)
			os.Exit(2)
		}
	} else {
		fmt.Print(report(len(tools), findings, *withWarn, *inputOnly, *toolsFile == "" && !*published))
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// tool is a tool's name and the decoded JSON of its published schemas.
type tool struct {
	name         string
	inputSchema  any
	outputSchema any
}

// loadTools reads the tools to check: the locally registered surface when path
// is empty, otherwise a captured tools/list result, which is checked as it
// stands since a saved body cannot be re-derived either way.
func loadTools(path string, variant bool) ([]tool, error) {
	if path == "" {
		return registeredTools(variant)
	}
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path) //nolint:gosec // a developer-supplied path
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return capturedTools(raw)
}

// registeredTools builds every product's toolset group and returns the schemas
// as they go out on the wire. Dependencies are nil: only static metadata is
// read, never the engine, as in cmd/docs-gen.
func registeredTools(variant bool) ([]tool, error) {
	groups := []*toolsets.ToolsetGroup{
		twprojects.DefaultToolsetGroup(false, false, nil),
		twdesk.DefaultToolsetGroup(false, nil),
		twspaces.DefaultToolsetGroup(false, false, nil),
		twchat.DefaultToolsetGroup(false, nil),
	}
	var registered []*mcp.Tool
	for _, group := range groups {
		for _, toolset := range group.Toolsets {
			for _, wrapper := range toolset.GetAvailableTools() {
				registered = append(registered, wrapper.Tool)
			}
		}
	}
	if variant {
		registered = helpers.VertexTools(registered)
	}

	tools := make([]tool, 0, len(registered))
	for _, t := range registered {
		decoded, err := decodeTool(t)
		if err != nil {
			return nil, err
		}
		tools = append(tools, decoded)
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].name < tools[j].name })
	return tools, nil
}

// decodeTool round-trips a registered tool's schemas through JSON, because the
// wire form is what Vertex parses and it is not the Go struct — Type/Types and
// Items/ItemsArray are rendered by a custom marshaller.
func decodeTool(t *mcp.Tool) (tool, error) {
	decoded := tool{name: t.Name}
	for _, part := range []struct {
		source any
		target *any
	}{{t.InputSchema, &decoded.inputSchema}, {t.OutputSchema, &decoded.outputSchema}} {
		if part.source == nil {
			continue
		}
		raw, err := json.Marshal(part.source)
		if err != nil {
			return tool{}, fmt.Errorf("marshal %s schema: %w", t.Name, err)
		}
		if err := json.Unmarshal(raw, part.target); err != nil {
			return tool{}, fmt.Errorf("decode %s schema: %w", t.Name, err)
		}
	}
	return decoded, nil
}

// capturedTools accepts a tools/list JSON-RPC response, a bare result, or the
// tools array on its own, so a body saved from any client works.
func capturedTools(raw []byte) ([]tool, error) {
	var envelope struct {
		Result *struct {
			Tools []json.RawMessage `json:"tools"`
		} `json:"result"`
		Tools []json.RawMessage `json:"tools"`
	}
	entries := []json.RawMessage(nil)
	switch err := json.Unmarshal(raw, &envelope); {
	case err == nil && envelope.Result != nil:
		entries = envelope.Result.Tools
	case err == nil && envelope.Tools != nil:
		entries = envelope.Tools
	default:
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, fmt.Errorf("not a tools/list result, a result body, or a tools array: %w", err)
		}
	}

	tools := make([]tool, 0, len(entries))
	for _, entry := range entries {
		var published struct {
			Name         string `json:"name"`
			InputSchema  any    `json:"inputSchema"`
			OutputSchema any    `json:"outputSchema"`
		}
		if err := json.Unmarshal(entry, &published); err != nil {
			return nil, fmt.Errorf("decode tool: %w", err)
		}
		tools = append(tools, tool{published.Name, published.InputSchema, published.OutputSchema})
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].name < tools[j].name })
	return tools, nil
}

func onlyErrors(findings []Finding) []Finding {
	kept := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if f.Severity == SeverityError {
			kept = append(kept, f)
		}
	}
	return kept
}

// report writes the findings grouped by rule, with the affected tools, since a
// single rule usually fires across a whole family of tools.
func report(toolCount int, findings []Finding, withWarn, inputOnly, variant bool) string {
	var out bytes.Buffer
	surface := "in the published surface"
	if variant {
		surface = "served to a Gemini client"
	}
	scope := "input and output schemas"
	if inputOnly {
		scope = "input schemas only"
	}
	if !withWarn {
		scope += ", errors only"
	}
	fmt.Fprintf(&out, "checked %d tools %s (%s)\n\n", toolCount, surface, scope)
	if len(findings) == 0 {
		fmt.Fprintf(&out, "No findings: Vertex AI can parse every schema %s.\n", surface)
		return out.String()
	}

	byRule := map[string][]Finding{}
	for _, f := range findings {
		byRule[f.Rule] = append(byRule[f.Rule], f)
	}
	rules := make([]string, 0, len(byRule))
	for rule := range byRule {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool { return len(byRule[rules[i]]) > len(byRule[rules[j]]) })

	for _, rule := range rules {
		group := byRule[rule]
		tools := map[string]struct{}{}
		for _, f := range group {
			tools[f.Tool] = struct{}{}
		}
		names := make([]string, 0, len(tools))
		for name := range tools {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Fprintf(&out, "%s  %s\n", group[0].Severity, rule)
		fmt.Fprintf(&out, "  %d occurrences across %d tools\n", len(group), len(names))
		fmt.Fprintf(&out, "  %s\n", group[0].Detail)
		fmt.Fprintf(&out, "  e.g. %s  %s%s\n", group[0].Tool, group[0].Schema, group[0].Path)
		if len(names) > 1 {
			shown := names
			if len(shown) > 6 {
				shown = shown[:6]
			}
			more := ""
			if len(names) > len(shown) {
				more = fmt.Sprintf(" (+%d more)", len(names)-len(shown))
			}
			fmt.Fprintf(&out, "  tools: %s%s\n", strings.Join(shown, ", "), more)
		}
		fmt.Fprintln(&out)
	}
	fmt.Fprintf(&out, "%d findings\n", len(findings))
	return out.String()
}
