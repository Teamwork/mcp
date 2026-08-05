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
- Annotation hints are mandatory and must be explicit: every tool sets `ReadOnlyHint`, `DestructiveHint` and `OpenWorldHint`. The latter two are `*bool` in the SDK, so a nil value is omitted from `tools/list` and the spec then defaults it to `true`; OpenAI's app review rejects tools with missing hints. Use `new(false)` unless the tool truly destroys data (deletes, plus Desk ticket create/reply, which email the customer) or reaches outside the customer's Teamwork account (help-doc articles published to a public knowledge base, Desk ticket create/reply). `TestAnnotationHintsAreExplicit` in `cmd/docs-gen/main_test.go` guards this across all four products with `allowDelete=true`.
- JSON-Schema gotcha (OpenAI Responses API): every `Type: "array"` node — including inside `AnyOf`/`OneOf`/`AllOf` branches — must declare `Items`. OpenAI rejects bare arrays at tool-registration time even with `strict: false`; Anthropic does not, so Claude Desktop hides the bug. `TestToolInputSchemasArrayItems` in `internal/twprojects/tools_test.go` guards this — if it fires, pick the right item schema rather than weakening the test.
- Date and date-time parameters go through the binders in `internal/helpers/tool_parser.go`, which accept more than one layout (see `dateTimeLayouts` in `internal/helpers/datetime.go`): RFC 3339, an offset-less date-time, and a plain `YYYY-MM-DD`. Models emit the plain date by default when asked about a range, so a strict RFC 3339 parse costs a failed first call and a visible retry. Do not narrow this back. Use `helpers.DateTimeFilterSchema(...)` rather than an inline schema so every filter advertises both forms, and pass `helpers.EndOfDay()` to the binder for any *upper-bound* filter (`end_date`, `*_before`) — a date-only value there must resolve to the day's last second, or the range silently drops its closing day. Handlers that forward the value as a raw query-string parameter instead of binding it use `helpers.NormalizeDateTime(...)`.
- `list_*` tools follow a specific contract — see `TaskList` in `internal/twprojects/tasks.go` as the canonical pattern:
  - Expose a `verbose` parameter (default `true`) via `helpers.VerboseSchema()`.
  - Execute the request with `twapi.ExecuteRaw(ctx, engine, req)` and stream the body straight to the caller, instead of decoding into the typed `*XxxListResponse` (avoids re-marshalling and preserves any fields the SDK struct doesn't model).
  - Expose a `fields` parameter via `helpers.FieldsSchema("<entity>")` whenever the SDK request has a `Filters.Fields.<Entity>` slot, parsed with `helpers.OptionalFieldsParam[projects.<Entity>](&req.Filters.Fields.<Entity>, "fields")`. It lets the caller name the attributes it wants instead of choosing between everything and `id` plus a label. Accepted values are derived by reflection from the entity struct's JSON attributes — the same names the tool's output schema publishes — so the input schema deliberately carries no enum; `OptionalFieldsParam` rejects anything else and lists the valid names in the error, and always appends `id` (rows need it to be addressable by a follow-up `get_*`, and `WebLinker` needs it to attach a web link). Do not expose the sideload slots (`Fields.CustomFields`, `Fields.Projects`, …): they multiply the surface with entities the caller did not ask about. `summarize_timelogs` is deliberately excluded — its `Fields` slots feed an aggregate it computes itself, and its output schema is strict.
  - When `fields` is supplied it wins over `verbose`, and no sideloads are requested: sideloading would hand back the bulk the selection exists to avoid. Guard the `verbose` defaults with `len(req.Filters.Fields.<Entity>) == 0` (or a leading `switch` case) so neither branch overwrites an explicit selection. `TestListTools*` in `internal/twprojects/sparse_fields_test.go` covers every tool from one table and fails when a tool declares `fields` without being listed there.
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
