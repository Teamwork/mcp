# docs-gen

Generates the Teamwork MCP tool reference from the registered toolsets, in two
forms:

| Output | What it is |
|---|---|
| `docs/tool-reference.md` | Markdown CRUD matrix, read on GitHub |
| `docs/index.html` | Browsable page published to [GitHub Pages](https://teamwork.github.io/mcp/) |

Both read the same registry, so they cannot describe different tool surfaces.
Neither needs an API token or a live server: the groups are built with nil
dependencies, and only static tool metadata (name, description, annotations) is
reflected over.

```sh
go run ./cmd/docs-gen           # write both documents
go run ./cmd/docs-gen -o -      # Markdown to stdout
go run ./cmd/docs-gen -html -   # HTML page to stdout
go run ./cmd/docs-gen -check    # fail if either committed document is stale
```

`go test ./cmd/docs-gen` runs the same check, so a tool added without
regenerating fails CI.

## The HTML page

One self-contained file — CSS, script, and the Teamwork.com logo are inlined, so
Pages serves it with no build step. The only external request is the Work Sans
webfont from Google Fonts.

It carries what the Markdown cannot: a per-tool description, a read/write badge
taken from each tool's `ReadOnlyHint`, the profile endpoints resolved to their
toolsets, and a client-side filter over all of it.

Colours, typography and the logo follow the
[Teamwork.com brand guidelines](https://www.teamwork.com/brand/). The palette in
`assets/site.css` is the same token set the marketing site ships; pink is an
accent only, never on buttons or large blocks of text. `assets/teamwork-logo.svg`
and `assets/teamwork-mark.svg` are the official assets, embedded verbatim — the
guidelines forbid redrawing or restyling the lockup, so nothing in the generator
rewrites their fills or geometry.

Output must stay deterministic: no timestamps, no map iteration.
`TestGeneratedHTMLIsDeterministic` fails otherwise, and every regeneration would
otherwise produce a diff.

## Changing what is documented

Ordinary tool add/remove/rename is caught automatically — regenerate and commit.
Two changes need a manual edit to `main.go`:

- flipping `allowDelete` to `true` on a shipped server, which also invalidates
  the "deletes are not published" note in both documents;
- adding a new product package — register it in `products()`.

## Deployment

`.github/workflows/pages.yaml` publishes the committed `docs/index.html` on push
to `main`. It serves the file as-is rather than regenerating, because the golden
test already guarantees it is current.
