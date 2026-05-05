# mcp-tokens

Reports tiktoken-based token counts for every MCP tool exposed by the codebase
(twprojects + twdesk + twspaces), sorted by cost. Introspects the local Go
source — no HTTP server, no auth.

Useful for tracking tool-description token budgets across revisions and for
spotting the heaviest tools when trimming.

## Snapshot the current tree

```bash
go run ./cmd/mcp-tokens                   # token table for every tool
go run ./cmd/mcp-tokens -encoding=cl100k_base
go run ./cmd/mcp-tokens -json > tools.json   # full export-tools-shaped JSON
```

## Diff against a base ref

`-base=<ref>` materialises that ref in a throwaway `git worktree`, runs the
same binary against it, and prints the delta. Your working tree is never
touched, so uncommitted edits in `internal/` are safe — handy for the inner
dev loop while you trim descriptions.

```bash
go run ./cmd/mcp-tokens -base=main                      # text summary + per-tool movers
go run ./cmd/mcp-tokens -base=main -format=markdown     # GFM table (PR-comment friendly)
go run ./cmd/mcp-tokens -base=main -format=json         # structured output for scripts
go run ./cmd/mcp-tokens -base=origin/main               # any git ref works
```

The diff cleanly handles tools added or removed between the two trees:
missing-on-one-side rows show the absent count as `0`.

Sample text output:

```
  base (main)         44,950
  current             33,723
  delta              -11,227  (-24.98%)

  per-tool deltas (117):
    tool                                      before     after     delta
    --------------------------------------  --------  --------  --------
    twprojects-list_tasks                        832       702      -130
    twprojects-update_timelog                    608       485      -123
    ...
```

## Caveats

- Counts use OpenAI's `tiktoken` (`o200k_base` by default). Treat them as a
  *relative* signal across revisions — Claude's tokenizer differs, but the
  percentage delta tracks closely.
- Counts cover the tool name, description, and JSON-marshalled input schema.
  They exclude per-message framing the model adds at runtime, so the absolute
  total is a lower bound on what the LLM actually sees.
- Diff mode requires `git` and a working `go` toolchain (it shells out to
  `go run` inside the temporary worktree). First run of a new base ref pays
  one Go-compile cost; subsequent runs reuse Go's build cache.
