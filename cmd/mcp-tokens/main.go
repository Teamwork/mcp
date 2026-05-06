// mcp-tokens reports tiktoken-based token counts for every MCP tool exposed
// by the codebase, sorted by cost. It introspects the Go source directly
// (no HTTP server, no auth), so it always reflects the local working tree.
//
// Usage:
//
//	go run ./cmd/mcp-tokens                       # token counts (default)
//	go run ./cmd/mcp-tokens -json                 # full export-tools-shaped JSON
//	go run ./cmd/mcp-tokens -base=main            # diff vs main, text output
//	go run ./cmd/mcp-tokens -base=main -format=markdown
//	go run ./cmd/mcp-tokens -encoding=cl100k_base
//
// Diff mode spins up a temporary `git worktree` at the base ref and runs
// the same binary there, so your working tree is never touched — uncommitted
// or staged changes under internal/ are safe.
//
// Token counts use OpenAI's tiktoken; treat the numbers as a *relative*
// signal across revisions, not as absolute Claude figures.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/localit-io/tiktoken-go"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
)

type toolCount struct {
	Name   string `json:"name"`
	Tokens int    `json:"tokens"`
}

type diffRow struct {
	Name   string `json:"name"`
	Before int    `json:"before"`
	After  int    `json:"after"`
}

func (d diffRow) Delta() int { return d.After - d.Before }

func main() {
	asJSON := flag.Bool("json", false, "emit full export-tools JSON instead of token counts")
	asCounts := flag.Bool("counts", false, "emit {tool: tokens} map as JSON (used internally by -base)")
	encoding := flag.String("encoding", "o200k_base", "tiktoken encoding name (e.g. o200k_base, cl100k_base)")
	baseRef := flag.String("base", "", "compare against this git ref (enables diff mode)")
	format := flag.String("format", "text", "diff output format: text, markdown, json")
	flag.Parse()

	if *format != "text" && *baseRef == "" {
		fail("-format=%s requires -base; -format only applies to diff output", *format)
	}

	groups := allGroups()

	if *asJSON {
		emitJSON(groups)
		return
	}

	enc, err := tiktoken.GetEncoding(*encoding)
	if err != nil {
		fail("get encoding %q: %v", *encoding, err)
	}

	if *asCounts {
		emitCounts(countTools(groups, enc))
		return
	}

	if *baseRef != "" {
		if err := runDiff(*baseRef, *format, *encoding, enc, groups); err != nil {
			fail("%v", err)
		}
		return
	}

	printSnapshot(countTools(groups, enc))
}

func emitCounts(rows []toolCount) {
	m := make(map[string]int, len(rows))
	for _, r := range rows {
		m[r.Name] = r.Tokens
	}
	if err := json.NewEncoder(os.Stdout).Encode(m); err != nil {
		fail("encode counts: %v", err)
	}
}

// allGroups builds every toolset group registered by the servers. The engine
// and HTTP client passed to factories are only dereferenced inside tool
// handlers — schemas are static at registration time, so we can pass
// nil/empty values safely.
func allGroups() []*toolsets.ToolsetGroup {
	httpClient := &http.Client{}
	return []*toolsets.ToolsetGroup{
		twprojects.DefaultToolsetGroup(false, true, nil),
		twdesk.DefaultToolsetGroup(false, httpClient),
		twspaces.DefaultToolsetGroup(false, httpClient),
	}
}

func countTools(groups []*toolsets.ToolsetGroup, enc *tiktoken.Tiktoken) []toolCount {
	var rows []toolCount
	for _, g := range groups {
		for _, ts := range g.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				t := tw.Tool
				payload := map[string]any{
					"name":         t.Name,
					"description":  t.Description,
					"input_schema": t.InputSchema,
				}
				b, err := json.Marshal(payload)
				if err != nil {
					fail("marshal %s: %v", t.Name, err)
				}
				rows = append(rows, toolCount{t.Name, len(enc.Encode(string(b), nil, nil))})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Tokens != rows[j].Tokens {
			return rows[i].Tokens > rows[j].Tokens
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func printSnapshot(rows []toolCount) {
	width := 0
	for _, r := range rows {
		if len(r.Name) > width {
			width = len(r.Name)
		}
	}
	dash := strings.Repeat("-", width)
	total := 0
	fmt.Printf("%-*s  tokens\n", width, "tool")
	fmt.Printf("%s  ------\n", dash)
	for _, r := range rows {
		fmt.Printf("%-*s  %6d\n", width, r.Name, r.Tokens)
		total += r.Tokens
	}
	fmt.Printf("%s  ------\n", dash)
	fmt.Printf("%-*s  %6d  (%d tools)\n", width, "TOTAL", total, len(rows))
}

func emitJSON(groups []*toolsets.ToolsetGroup) {
	out := map[string]any{}
	for _, g := range groups {
		for _, ts := range g.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				t := tw.Tool
				entry := map[string]any{
					"description": t.Description,
					"title":       t.Title,
					"inputSchema": map[string]any{"jsonSchema": t.InputSchema},
					"type":        "dynamic",
				}
				if t.OutputSchema != nil {
					entry["outputSchema"] = map[string]any{"jsonSchema": t.OutputSchema}
				}
				if t.Annotations != nil {
					entry["annotations"] = t.Annotations
				}
				if t.Meta != nil {
					entry["_meta"] = t.Meta
				}
				out[t.Name] = entry
			}
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail("encode: %v", err)
	}
}

// runDiff captures token counts for the current tree and the tree at baseRef,
// then emits the diff. The base ref is materialised in a throwaway git worktree
// so the caller's working tree is never modified — uncommitted edits in
// internal/ are safe to leave in place.
func runDiff(baseRef, format, encName string, enc *tiktoken.Tiktoken, groups []*toolsets.ToolsetGroup) error {
	if err := exec.Command("git", "rev-parse", "--verify", "--quiet", baseRef+"^{commit}").Run(); err != nil {
		return fmt.Errorf("base ref %q not found", baseRef)
	}

	current := countTools(groups, enc)

	wt, err := os.MkdirTemp("", "mcp-tokens-base-*")
	if err != nil {
		return fmt.Errorf("create tmpdir: %w", err)
	}
	defer func() {
		_ = exec.Command("git", "worktree", "remove", "--force", wt).Run()
		_ = os.RemoveAll(wt)
	}()

	if out, err := exec.Command("git", "worktree", "add", "--detach", wt, baseRef).CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %v: %s", err, out)
	}

	// Inject our own cmd/mcp-tokens into the base worktree (it may not exist
	// there yet) so we can `go run ./cmd/mcp-tokens -json` against base's
	// internal/ packages.
	cmdDest := filepath.Join(wt, "cmd", "mcp-tokens")
	_ = os.RemoveAll(cmdDest)
	if err := os.MkdirAll(filepath.Dir(cmdDest), 0o755); err != nil {
		return fmt.Errorf("mkdir cmd dir in worktree: %w", err)
	}
	if out, err := exec.Command("cp", "-R", "cmd/mcp-tokens", cmdDest).CombinedOutput(); err != nil {
		return fmt.Errorf("copy cmd/mcp-tokens into worktree: %v: %s", err, out)
	}

	// Make sure tiktoken-go is in the base worktree's go.mod (idempotent).
	if out, err := runIn(wt, "go", "get", "github.com/localit-io/tiktoken-go").CombinedOutput(); err != nil {
		return fmt.Errorf("go get tiktoken-go in base worktree: %v: %s", err, out)
	}

	var stdout bytes.Buffer
	dump := runIn(wt, "go", "run", "./cmd/mcp-tokens", "-counts", "-encoding", encName)
	dump.Stdout = &stdout
	dump.Stderr = os.Stderr
	if err := dump.Run(); err != nil {
		return fmt.Errorf("count tools in base worktree: %w", err)
	}

	var baseMap map[string]int
	if err := json.Unmarshal(stdout.Bytes(), &baseMap); err != nil {
		return fmt.Errorf("parse base counts: %w", err)
	}

	base := make([]toolCount, 0, len(baseMap))
	for name, tokens := range baseMap {
		base = append(base, toolCount{name, tokens})
	}

	rows := buildDiff(base, current)
	switch format {
	case "json":
		return emitDiffJSON(baseRef, rows)
	case "markdown":
		emitDiffMarkdown(baseRef, rows)
	case "text", "":
		emitDiffText(baseRef, rows)
	default:
		return fmt.Errorf("unknown format %q (want text|markdown|json)", format)
	}
	return nil
}

func runIn(dir, name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.Dir = dir
	return c
}

func buildDiff(before, after []toolCount) []diffRow {
	bMap := make(map[string]int, len(before))
	for _, r := range before {
		bMap[r.Name] = r.Tokens
	}
	aMap := make(map[string]int, len(after))
	for _, r := range after {
		aMap[r.Name] = r.Tokens
	}
	seen := make(map[string]bool)
	for n := range bMap {
		seen[n] = true
	}
	for n := range aMap {
		seen[n] = true
	}
	rows := make([]diffRow, 0, len(seen))
	for n := range seen {
		rows = append(rows, diffRow{n, bMap[n], aMap[n]})
	}
	// Most-saved (largest negative delta) first; tiebreak by name.
	sort.Slice(rows, func(i, j int) bool {
		di, dj := rows[i].Delta(), rows[j].Delta()
		if di != dj {
			return di < dj
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func diffTotals(rows []diffRow) (before, after, delta int, pct float64) {
	for _, r := range rows {
		before += r.Before
		after += r.After
	}
	delta = after - before
	if before > 0 {
		pct = float64(delta) / float64(before) * 100
	}
	return
}

func emitDiffText(baseRef string, rows []diffRow) {
	before, after, delta, pct := diffTotals(rows)
	baseLabel := fmt.Sprintf("base (%s)", baseRef)
	labelWidth := max(len(baseLabel), len("current"), len("delta"))
	fmt.Println()
	fmt.Printf("  %-*s  %12s\n", labelWidth, baseLabel, commafy(before))
	fmt.Printf("  %-*s  %12s\n", labelWidth, "current", commafy(after))
	fmt.Printf("  %-*s  %12s  (%+.2f%%)\n\n", labelWidth, "delta", signed(delta), pct)

	movers := filterMovers(rows)
	if len(movers) == 0 {
		fmt.Println("  no per-tool changes")
		return
	}

	width := 0
	for _, r := range movers {
		if len(r.Name) > width {
			width = len(r.Name)
		}
	}
	fmt.Printf("  per-tool deltas (%d):\n", len(movers))
	fmt.Printf("    %-*s  %8s  %8s  %8s\n", width, "tool", "before", "after", "delta")
	fmt.Printf("    %s  %s  %s  %s\n",
		strings.Repeat("-", width), strings.Repeat("-", 8), strings.Repeat("-", 8), strings.Repeat("-", 8))
	for _, r := range movers {
		fmt.Printf("    %-*s  %8d  %8d  %+8d\n", width, r.Name, r.Before, r.After, r.Delta())
	}
}

func emitDiffMarkdown(baseRef string, rows []diffRow) {
	before, after, delta, pct := diffTotals(rows)
	fmt.Printf("### MCP tool token deltas vs `%s`\n\n", baseRef)
	fmt.Println("| metric | tokens |")
	fmt.Println("|---|---:|")
	fmt.Printf("| base | %s |\n", commafy(before))
	fmt.Printf("| current | %s |\n", commafy(after))
	fmt.Printf("| **delta** | **%s (%+.2f%%)** |\n\n", signed(delta), pct)

	movers := filterMovers(rows)
	if len(movers) == 0 {
		fmt.Println("_No per-tool changes._")
		return
	}
	fmt.Printf("#### Per-tool deltas (%d)\n\n", len(movers))
	fmt.Println("| tool | before | after | delta |")
	fmt.Println("|---|---:|---:|---:|")
	for _, r := range movers {
		fmt.Printf("| `%s` | %s | %s | **%s** |\n",
			r.Name, commafy(r.Before), commafy(r.After), signed(r.Delta()))
	}
}

func emitDiffJSON(baseRef string, rows []diffRow) error {
	before, after, delta, pct := diffTotals(rows)
	out := map[string]any{
		"base":         baseRef,
		"before_total": before,
		"after_total":  after,
		"delta":        delta,
		"delta_pct":    pct,
		"tools":        rows,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func filterMovers(rows []diffRow) []diffRow {
	out := make([]diffRow, 0, len(rows))
	for _, r := range rows {
		if r.Delta() != 0 {
			out = append(out, r)
		}
	}
	return out
}

func commafy(n int) string {
	negative := n < 0
	if negative {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func signed(n int) string {
	if n > 0 {
		return "+" + commafy(n)
	}
	return commafy(n)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
