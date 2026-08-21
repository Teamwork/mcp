package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"github.com/teamwork/mcp/internal/cli"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// assets holds everything the generated page inlines. The page is a single
// self-contained file so GitHub Pages serves it with no build step and no asset
// paths to keep in sync; the only external request it makes is the Work Sans
// webfont, which is the brand typeface.
//
//go:embed assets
var assets embed.FS

const (
	// endpointURL is the public hosted server, as documented in
	// docs/usage/README.md. Profile paths hang off it.
	endpointURL = "https://mcp.ai.teamwork.com"
	repoURL     = "https://github.com/teamwork/mcp"
	repoLabel   = "github.com/teamwork/mcp"
)

// htmlSite is the whole template context. Every field is derived from the
// toolset registry or from an embedded asset — nothing is timestamped or
// map-ordered, so repeated runs are byte-identical and the golden test holds.
type htmlSite struct {
	CSS       template.CSS
	JS        template.JS
	Logo      template.HTML
	Favicon   template.URL
	Endpoint  string
	RepoURL   string
	RepoLabel string
	Columns   []string
	Stats     htmlStats
	Profiles  []htmlProfile
	Products  []htmlProduct
}

type htmlStats struct {
	Products, Toolsets, Tools, Read, Write int
}

type htmlProfile struct {
	Name     string
	Toolsets string
}

type htmlProduct struct {
	Label     string
	Slug      string
	Prefix    string
	Scope     string
	ToolCount int
	Toolsets  []htmlToolset
}

type htmlToolset struct {
	Method      string
	Title       string
	Description string
	Resources   []htmlResource
	Tools       []htmlTool
}

type htmlResource struct {
	Name  string
	Cells []bool // one per matrixColumns entry, same order
}

type htmlTool struct {
	Name        string
	Description string
	Access      string // "read" or "write", from the tool's ReadOnlyHint
	Haystack    string // lowercased name + description, for client-side filtering
}

// generateHTML renders the browsable tool reference published to GitHub Pages.
// It is the HTML counterpart of generate() and reads the same registry, so the
// two documents cannot describe different tool surfaces.
func generateHTML() (string, error) {
	// Profiles are registered by internal/cli rather than by an init(), so the
	// profile table is empty unless this runs first.
	cli.NewMethods()

	site := htmlSite{
		Endpoint:  endpointURL,
		RepoURL:   repoURL,
		RepoLabel: repoLabel,
		Columns:   matrixColumns,
	}

	css, err := assets.ReadFile("assets/site.css")
	if err != nil {
		return "", fmt.Errorf("reading stylesheet: %w", err)
	}
	site.CSS = template.CSS(css)

	js, err := assets.ReadFile("assets/site.js")
	if err != nil {
		return "", fmt.Errorf("reading script: %w", err)
	}
	site.JS = template.JS(js)

	// The logo and mark are the official Teamwork.com assets, embedded verbatim:
	// the brand guidelines forbid redrawing or restyling the lockup, so nothing
	// here rewrites their fills or geometry.
	logo, err := assets.ReadFile("assets/teamwork-logo.svg")
	if err != nil {
		return "", fmt.Errorf("reading logo: %w", err)
	}
	site.Logo = template.HTML(logo) //nolint:gosec // embedded, hand-audited brand asset

	mark, err := assets.ReadFile("assets/teamwork-mark.svg")
	if err != nil {
		return "", fmt.Errorf("reading mark: %w", err)
	}
	site.Favicon = template.URL("image/svg+xml," + url.PathEscape(strings.TrimSpace(string(mark))))

	site.Products, site.Stats = collectProducts()
	site.Stats.Products = len(site.Products)
	site.Profiles = collectProfiles()

	tmpl, err := template.ParseFS(assets, "assets/page.html.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, site); err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}
	return b.String(), nil
}

// collectProducts walks the registry in the same order as the Markdown doc and
// returns the rendered products alongside the totals shown in the hero.
func collectProducts() ([]htmlProduct, htmlStats) {
	var stats htmlStats
	var out []htmlProduct

	for _, p := range products() {
		hp := htmlProduct{
			Label:  p.label,
			Slug:   strings.ToLower(p.label),
			Prefix: p.group.ToolPrefix(),
			Scope:  p.group.Scope(),
		}
		for _, method := range sortedMethods(p.group) {
			ts, err := p.group.GetToolset(method)
			if err != nil {
				continue
			}
			hts := htmlToolset{
				Method:      string(ts.Method),
				Title:       toolsetTitle(ts.Method),
				Description: ts.Description,
			}

			resources, _ := classifyToolset(ts)
			for _, ri := range resources {
				cells := make([]bool, 0, len(matrixColumns))
				for _, col := range matrixColumns {
					cells = append(cells, ri.columns[col])
				}
				hts.Resources = append(hts.Resources, htmlResource{Name: ri.name, Cells: cells})
			}

			for _, tw := range ts.GetAvailableTools() {
				access := "write"
				if tw.Tool.Annotations != nil && tw.Tool.Annotations.ReadOnlyHint {
					access = "read"
				}
				desc := firstSentence(tw.Tool.Description)
				hts.Tools = append(hts.Tools, htmlTool{
					Name:        tw.Tool.Name,
					Description: desc,
					Access:      access,
					Haystack:    strings.ToLower(tw.Tool.Name + " " + desc),
				})
				if access == "read" {
					stats.Read++
				} else {
					stats.Write++
				}
				stats.Tools++
				hp.ToolCount++
			}
			// GetAvailableTools returns read tools then write tools in
			// registration order; sorting by name gives a scannable list and a
			// stable one.
			sort.Slice(hts.Tools, func(i, j int) bool { return hts.Tools[i].Name < hts.Tools[j].Name })

			hp.Toolsets = append(hp.Toolsets, hts)
			stats.Toolsets++
		}
		out = append(out, hp)
	}
	return out, stats
}

// collectProfiles renders the named toolset collections the HTTP server serves
// on their own paths, resolving each method to its product and toolset title.
func collectProfiles() []htmlProfile {
	names := toolsets.ListProfiles()
	sort.Strings(names)

	// Map a tool prefix back to its product label so a profile spanning two
	// products reads as "Projects — …; Desk — …" rather than a flat list in
	// which twprojects-content and twspaces-content are indistinguishable.
	labelByPrefix := map[string]string{}
	var prefixOrder []string
	for _, p := range products() {
		labelByPrefix[p.group.ToolPrefix()] = p.label
		prefixOrder = append(prefixOrder, p.group.ToolPrefix())
	}

	out := make([]htmlProfile, 0, len(names))
	for _, name := range names {
		methods, ok := toolsets.LookupProfile(name)
		if !ok {
			continue
		}
		if len(methods) == 1 && methods[0] == toolsets.MethodAll {
			out = append(out, htmlProfile{Name: name, Toolsets: "Every toolset"})
			continue
		}

		titles := map[string][]string{}
		for _, m := range methods {
			prefix, _, _ := strings.Cut(string(m), "-")
			titles[prefix] = append(titles[prefix], toolsetTitle(m))
		}
		var groups []string
		for _, prefix := range prefixOrder {
			if len(titles[prefix]) == 0 {
				continue
			}
			groups = append(groups, labelByPrefix[prefix]+" — "+strings.Join(titles[prefix], ", "))
		}
		out = append(out, htmlProfile{Name: name, Toolsets: strings.Join(groups, "; ")})
	}
	return out
}

// abbreviations are the sentence-internal full stops firstSentence must not cut
// on. Tool descriptions use "e.g." and "i.e." freely, and splitting there would
// publish a fragment.
var abbreviations = []string{"e.g.", "i.e.", "etc.", "vs.", "no."}

// firstSentence returns the leading sentence of a tool description — enough to
// say what a tool does without reproducing its full parameter prose, which can
// run to several hundred characters. A description with no terminal full stop
// is returned whole.
func firstSentence(desc string) string {
	desc = strings.Join(strings.Fields(desc), " ")
	for i, r := range desc {
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		// A terminator only ends a sentence at the end of the string or before
		// whitespace; "v3.json" and "1.5" stay intact.
		rest := desc[i+1:]
		if rest == "" {
			return desc
		}
		if !unicode.IsSpace(rune(rest[0])) {
			continue
		}
		head := desc[:i+1]
		if isAbbreviation(head) {
			continue
		}
		return head
	}
	return desc
}

// isAbbreviation reports whether head ends in one of the known non-terminal
// abbreviations, or in a single letter followed by a full stop (an initial).
func isAbbreviation(head string) bool {
	lower := strings.ToLower(head)
	for _, abbr := range abbreviations {
		if strings.HasSuffix(lower, abbr) {
			return true
		}
	}
	fields := strings.Fields(lower)
	return len(fields) > 0 && len(fields[len(fields)-1]) == 2
}
