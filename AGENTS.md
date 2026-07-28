# AGENTS.md

A concise guide for AI coding agents working on this repository. It complements `README.md` with machine-actionable build, test, run, and convention notes.

## Project overview
- Language/runtime: Go 1.26 or newer
- Purpose: Model Context Protocol (MCP) server for Teamwork.com with HTTP, STDIO transports and HTTP CLI.
- Main entry points:
  - STDIO server: `cmd/mcp-stdio/main.go`
  - HTTP server: `cmd/mcp-http/main.go`
  - HTTP CLI (tester): `cmd/mcp-http-cli/main.go`
- Core domain/tooling: `internal/twprojects` (tools for projects, tasks, users, tags, comments, milestones, timers, timelogs, etc.)

## Setup commands
- Install Go toolchain: Go 1.26 or newer (module declares `go 1.26.0`).
- Sync deps: `go mod download`
- Lint/format (optional but recommended):
  - Format: `gofmt -s -w .`
  - Vet: `go vet ./...`
  - Install `golangci-lint` and run `golangci-lint -c .golangci.yml run ./...` for more checks.

## Build and run
- STDIO server (local, safest for editor/desktop agents):
  - Env: `TW_MCP_BEARER_TOKEN=<token>`
  - Run: `go run cmd/mcp-stdio/main.go`
  - Flags: `-read-only` to restrict writes, `-toolsets=<comma-separated>` to limit exposed tools.
- HTTP server (for hosted/cloud or tools that only speak HTTP):
  - Env (examples): `TW_MCP_SERVER_ADDRESS=:8080`, optional `TW_MCP_LOG_LEVEL=debug`
  - Run: `go run cmd/mcp-http/main.go`
  - Health: GET `/health`
- HTTP CLI (for quick tests against HTTP server):
  - List tools: `go run cmd/mcp-http-cli/main.go -mcp-url=<url> -mcp-token=<token> list-tools`
  - Call tool: `go run cmd/mcp-http-cli/main.go -mcp-url=<url> -mcp-token=<token> call-tool <toolName> '{"k":"v"}'`
- Docker (optional, for image builds):
  - Requires Docker Buildx. Local load: `make build` or `make build-stdio`
  - Multi-arch push (maintainers): `make push` or `make push-stdio`

## Testing instructions
- Run all tests: `go test ./...`
- Focus a package: `go test ./internal/twprojects`
- Run a single test: `go test ./internal/twprojects -run TestName`
- Notes:
  - Tests in `internal/twprojects` mock the Teamwork API via an in-memory `twapi.Engine` stub (`mcpServerMock` in `internal/twprojects/main_test.go`). No external services are required.
  - Keep tests fast and hermetic. Add/update tests alongside any tool changes.

## Configuration and env vars
Common variables (subset; see command READMEs for complete lists):
- Auth: `TW_MCP_BEARER_TOKEN` (Teamwork API bearer token; required for both transports)
- API base: `TW_MCP_API_URL` (defaults to `https://teamwork.com`; set to your site domain like `https://<site>.teamwork.com` when needed)
- HTTP server: `TW_MCP_SERVER_ADDRESS` (bind address, default `:8080`), `TW_MCP_URL`, `TW_MCP_ENV`, logging and Datadog vars
- Logging: `TW_MCP_LOG_FORMAT` (`text`|`json`), `TW_MCP_LOG_LEVEL` (`info`|`debug`|...)
- Inspector note: when using OAuth with Let’s Encrypt staging, set `NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem` for the MCP Inspector.

## Code layout and conventions
- Tool surface lives in `internal/twprojects/*.go`. Files are organized by resource (e.g., `tasks.go`, `projects.go`) with matching `*_test.go`.
- Tools are registered in `internal/twprojects/tools.go` via `DefaultToolsetGroup(readOnly, allowDelete, engine)`.
  - Read-only enforcement is centralized; writes go in `AddWriteTools(...)`, reads in `AddReadTools(...)`.
  - Destructive operations (delete) are guarded by the `allowDelete` flag; keep this pattern intact.
- Prefer clear, explicit parameter schemas for tools and consistent naming:
  - Tool names use the `twprojects-<action>` convention as exposed to MCP clients (see the existing tools for examples).
- Follow standard Go style: keep functions small, pass `context.Context`, check and wrap errors, and ensure deterministic tests.

## Adding or modifying tools (quick checklist)
- Implement the tool function in the appropriate file under `internal/twprojects/` and return a `server.ServerTool` (see existing patterns, e.g., create/update/get/list functions in each file).
- Register it in `DefaultToolsetGroup(...)`:
  - Read-only tools → `AddReadTools(...)`
  - Write tools → `AddWriteTools(...)` (and behind `allowDelete` for deletes)
- Add tests in the matching `*_test.go` file using `mcpServerMock(...)` and `toolRequest` helpers found in `internal/twprojects/main_test.go`.
- JSON-Schema gotcha (OpenAI Responses API): every `Type: "array"` node — including inside `AnyOf`/`OneOf`/`AllOf` branches — must declare `Items`. OpenAI rejects bare arrays at tool-registration time even with `strict: false`; Anthropic does not, so Claude Desktop hides the bug. `TestToolInputSchemasArrayItems` in `internal/twprojects/tools_test.go` guards this — if it fires, pick the right item schema rather than weakening the test.
- `list_*` tools follow a specific contract — see `TaskList` in `internal/twprojects/tasks.go` as the canonical pattern:
  - Expose a `verbose` parameter (default `true`) via `helpers.VerboseSchema()`.
  - Execute the request with `twapi.ExecuteRaw(ctx, engine, req)` and stream the body straight to the caller, instead of decoding into the typed `*XxxListResponse` (avoids re-marshalling and preserves any fields the SDK struct doesn't model).
  - When `verbose=false`: set sparse fields on `req.Filters.Fields.<Entity>` to a minimal set (typically `id` + name/title) and skip any hardcoded `Filters.Include` sideloads.
  - When `verbose=true`: include sideloads and the full field set.
  - Wrap the published output schema with `helpers.WithOptionalFields(...)` at the `OutputSchema:` line (not in the `init()` block — keep `get_*` schemas strict). This clears every nested `required` array so sparse responses returned when `verbose=false` still validate. `StructuredContent` is always populated in both modes.
  - OpenAI strict-mode caveat: because `verbose=false` returns a sparse subset of fields, `list_*` output schemas cannot satisfy OpenAI's strict structured-output mode (which requires every property to be `required` and forbids `additionalProperties`). Clients targeting `list_*` tools must run with strict mode disabled; `get_*` tools remain strict-compatible.
- Run `go test ./internal/twprojects` until green.
- `docs/tool-reference.md` is generated by `cmd/docs-gen` and guarded by `go test`; ordinary tool add/remove/rename is caught automatically (regenerate with `go run ./cmd/docs-gen`), but two changes need a manual edit to `cmd/docs-gen/main.go`: flipping `allowDelete` to `true` on a shipped server, or adding a new product package (register it in `products()`).

## MCP SDK upgrades and protocol compatibility
- The SDK (`github.com/modelcontextprotocol/go-sdk`) keeps its Go API stable but does change wire behaviour between minor versions. Read `docs/mcpgodebug.md` in the module cache (`$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@<version>/docs/mcpgodebug.md`) before and after any bump: it lists every behaviour change and the `MCPGODEBUG=<param>=1` escape hatch that restores the old behaviour. Escape hatches are removed two minor versions later, so they are stopgaps, never permanent settings.
- Old clients must keep working. When the SDK adds a protocol method that replaces an older one, support both: `server/discover` (SEP-2575) and `initialize` are both in `auth.methodsWhitelist` for exactly this reason. Never remove a legacy entry from that whitelist.
- `internal/request.ResponseWriter` must keep its `Flush()` and `Unwrap()` methods. The SDK writes Server-Sent Events through `http.ResponseController`, which resolves the flusher by unwrapping — every streamable HTTP response on `/` is an SSE stream. Without them flushing silently degrades to a no-op and streaming clients hang. It also caps the body it captures for logging, because SSE streams live as long as the request.
- Cacheable results (SEP-2549: `tools/list`, `prompts/list`, `resources/list`, `resources/templates/list`, `resources/read`, `server/discover`) default to `cacheScope: "public"`. Anything whose content varies per token — `tools/list` is filtered by OAuth scope — must be set to `cacheScopePrivate` in the receiving middleware in `internal/config/config.go`, or a shared proxy can serve one tenant's response to another.
- Do not advertise capabilities the server does not honour. `logging` is deliberately omitted, and the `listChanged` flags are deliberately false: advertising them invites clients to open a long-lived `subscriptions/listen` stream that a stateless, load-balanced deployment would hold open with nothing to send. `TestCapabilitiesOmitLogging` and `TestCapabilitiesOmitListChanged` guard this.
- Do not add a manual keepalive ping loop. `mcp.ServerOptions.KeepAlive` in `config.NewMCPServer` already drives pings. In `cmd/mcp-stdio` a hand-rolled loop is actively harmful: on failure it has no request to reply to, so the error is written onto stdout, which is the protocol stream.
- Pin limits the SDK also enforces. `mcp.StreamableHTTPOptions.MaxRequestBodyBytes` is set explicitly in `cmd/mcp-http/main.go`; left at zero the SDK applies its own default (4 MiB), silently tightening the limit `limitBodyMiddleware` advertises.
- Client-visible protocol changes belong in the "Protocol Compatibility" section of `cmd/mcp-http/README.md`.

## Security considerations for agents
- Never print or commit bearer tokens. Use env vars only. Redact tokens in logs.
- Prefer running the STDIO server with `-read-only` by default during development.
- For HTTP deployments, ensure TLS, least-privilege tokens, and sensible log levels.
- Be careful with delete operations; keep them gated behind `allowDelete`.

## PR/commit guidance
- Before commit: `gofmt -s -w .` (or run your editor’s Go format), `go vet ./...`, `go test ./...`.
- Include or update tests for any tool behavior changes.
- Keep `README.md` end-user focused; put agent-oriented details here.

## Useful references (in-repo)
- `README.md` — overview and quick-starts (HTTP, STDIO, CLI)
- `docs/usage/README.md` — end-user connection guide for popular MCP clients
- `cmd/mcp-stdio/README.md` — STDIO server flags and envs
- `cmd/mcp-http/README.md` — HTTP server envs and endpoints
- `cmd/mcp-http-cli/README.md` — CLI usage
- `internal/twprojects/tools.go` — tool registration hub

---
Treat this file as living documentation. Update it alongside changes to commands, flags, and conventions so agents can reliably build, test, and ship edits.
