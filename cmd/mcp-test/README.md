# mcp-test

Walks MCP tool handlers through a **real Teamwork.com site**. It bypasses the
MCP server transport and calls each tool's `Handler` directly with the same JSON
payload an LLM would send, so you see end-to-end behaviour — field-name
resolution, value coercion, `twId`↔name translation, the schema cache — without
standing up a server or an LLM client.

> [!WARNING]
> This creates and deletes real data on the site you point it at. Use a scratch
> project, not a customer's.

## Configure

Environment variables, or the equivalent flag. Each flag defaults to its
variable, so a flag wins when both are given.

| Variable | Flag | Meaning |
| --- | --- | --- |
| `TWAPI_SERVER` | `-server` | Site base URL, e.g. `https://yoursite.teamwork.com` |
| `TWAPI_TOKEN` | `-token` | Bearer token or personal API key |
| `PROJECT_ID` | `-project` | Project the suites create their artefacts on |
| `TWAPI_AUTH` | — | `basic` or `bearer`; auto-detected from the token prefix (`twp_` → basic) |

These are the same variables `CONTRIBUTING.md` uses for `go test`, not the
servers' `TW_MCP_*` ones.

## Run

```bash
export TWAPI_SERVER=https://yoursite.teamwork.com TWAPI_TOKEN=... PROJECT_ID=...

go run ./cmd/mcp-test                        # default suite
go run ./cmd/mcp-test -suite=all             # every suite
go run ./cmd/mcp-test -suite=custom-items    # one named suite
go run ./cmd/mcp-test -step                  # pause for ENTER between steps
go run ./cmd/mcp-test -keep                  # skip cleanup, print the IDs to inspect
```

Every suite cleans up after itself when the run ends, including after a failed
step or a Ctrl-C — cleanup runs on a context detached from the signal handler so
an interrupted run doesn't strand artefacts on the site. `-keep` replaces
cleanup with a list of what was created.

## Suites

| `-suite` | Covers |
| --- | --- |
| `custom-items` | Custom item types, their fields, and records addressed by field *name* and option *label* |

## Adding a suite

Suites live one per file. Implement the `suite` interface from
[main.go](main.go) — `steps()`, `cleanup()`, `artefacts()` — and register the
constructor in `suiteRegistry`; that's what publishes the new `-suite` value.
See [custom_items.go](custom_items.go) as the template.

Use `runner.callToolExpectOK` for a step that must succeed, `runner.callTool`
when you want to assert on an `IsError` result (negative tests), and
`runner.callToolIgnoreError` inside `cleanup`, which has to keep going after a
single delete fails.

## Caveats

- Handlers are called directly, so the toolset framework is not in the loop:
  `-read-only` and `allowDelete` gating never applies, and a tool that
  `DefaultToolsetGroup` would not register is still reachable here.
- New IDs are parsed out of the handler's text result (`"... with ID 1234"`),
  because the create handlers return no structured content. A reworded success
  message will break the capture.
- The customer URL is injected into the context the way the servers do it, so
  `meta.webLink` is populated; the bearer token is not, so tools relying on
  `config.BearerTokenFromContext` (Desk) would need that added.
