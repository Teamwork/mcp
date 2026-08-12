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
- A tool that advertises a parameter needs a test that asserts the parameter reaches the wire, not just that the call succeeds: the mocks reply with the same canned body either way, so a silently dropped filter or page argument looks identical to a working one. Use `testutil.ProjectsMCPServerMockWithRequestURL` / `testutil.DeskMCPServerMockWithRequestURL` (or the `...WithRequestBody` variants for writes) to assert the query string, and `testutil.DeskMCPServerMockWithRequest` when the HTTP method is what distinguishes two tools (`twdesk-link_task_to_ticket` and `twdesk-unlink_task_from_ticket` share a path and differ only in POST versus DELETE). `twdesk-search_tickets` shipped ignoring page, pageSize, orderBy, orderDirection and fields for exactly this reason.
- Desk SDK gotcha: `desksdkgo`'s `client.Tickets.Search` builds its query solely from the qs-encoded `SearchTicketsFilter`, which carries no pagination, ordering or sparse-fieldset field — anything you set outside the struct is discarded. `internal/twdesk/tickets.go` therefore reaches `/search/tickets.json` through `deskclient.NewService[...]` plus `NewDefaultPathHandler("search/tickets")`, which takes a `url.Values` the handler controls. Check any `<Action><Model>Filter` struct for the params you need before routing a tool through its bespoke SDK method.
- A tool handler never returns a raw Go error for something the API answered. Every API call goes through `helpers.HandleAPIError(err, "failed to …")`, which turns a status the caller can act on into an `IsError` tool result; a raw error becomes a protocol-level error instead, so the model sees a transport failure with no status to read and none of the handler's wrapped message. Local faults are tool results too, via `helpers.NewToolResultTextError(...)` — bad caller input (malformed base64 in `twdesk-create_file`) as much as an encode failure. `TestAPIFailuresAreToolResults` in `internal/twdesk/error_test.go` drives representative Desk tools against 403/404/500 mocks and fails on a Go error before its assertion runs.
- `HandleAPIError` reads the status out of the Desk SDK's *message text*, because `desksdkgo` declares no error type at all: every non-2xx response leaves it as a bare `fmt.Errorf("unexpected status code: %d", …)` (`client/resource.go`, `tickets.go`, `helpdocarticles.go`) or `"failed to upload file, status code: %d, …"` (`Files.Upload`). Without that parse the typed-`*twapi.HTTPError` branch misses and every Desk failure falls through to the raw-error return — so calling `HandleAPIError` in a Desk handler is only load-bearing while `deskStatusCodePattern` in `internal/helpers/error.go` still matches. If a Desk SDK bump introduces a real error type, switch to `errors.AsType` on it and drop the pattern. `TestHandleAPIError` in `internal/helpers/error_test.go` pins both SDKs' shapes, plus that an error carrying no status stays a Go error.
- Desk pages are smaller than v3's, so Desk tools use the local `pageSizeSchema()` / `searchPageSizeSchema()` in `internal/twdesk/meta.go`, never `helpers.PageSizeSchema()` (500, the v3 limit). Desk rewrites an oversized page instead of rejecting it: list endpoints clamp over 100 down to 100, and the search endpoints reset anything over 200 back to 50, so asking for 250 returns fewer rows than asking for 200 and the response says nothing about it. `TestPageSizeCeilingsMatchTheAPI` and `TestNoToolAdvertisesAnUnreachablePage` in `internal/twdesk/meta_test.go` guard both ceilings.
- Desk query-string dates are `YYYY-MM-DD` only. The Desk API binds them through `models.Time.UnmarshalParam` (deskapi `models/ticket_report_model.go`), which parses that one layout; its `UnmarshalJSON` sibling accepts RFC 3339, but that path only serves request bodies, so sending a timestamp on a GET fails the Echo bind and returns 400 before the handler runs. `desksdkgo` models `SearchTicketsFilter.StartDate`/`.EndDate` as `*time.Time`, and qs renders those as RFC 3339 — set them on the `url.Values` yourself instead. Do not add `helpers.EndOfDay()` to the upper bound: `SearchTickets` runs `ApplyDefaults` → `FixDateRanges`, which already widens `startDate` to the start of its day and `endDate` to the end of its, then filters `createdAt` on that range.
- Moving a task between tasklists carries its subtasks and detaches an invalidated parent, both server-side. `PUT /projects/api/v3/tasks/{id}.json` (the SDK's `TaskUpdate`, behind `twprojects-update_task`) moves the whole subtree, then drops the moved task's own parent link when that parent is not in the destination. Only a parent the payload *inherited* is dropped — one the caller sent is still validated, so passing `parent_task_id` alongside a `tasklist_id` change fails with "parent task is in another list". Do not send one when moving. This arrived in an API fix in 2026-08; before it, v3 moved only the named task and stranded the subtree, which is why the client used to move each descendant by hand. `twprojects-move_tasks` is now a loop of task updates, and exists to batch several subtrees into one tool call — it skips a task already carried by another, since a descendant listed before its ancestor would otherwise be detached.
- Clearing a task's parent needs the `0` sentinel, not `null`. `parentTaskId: 0` detaches a subtask; `null` is a no-op, because `null` means "not provided" for every optional parameter in this repo. Commit `be42c41` added `{Type: "null"}` to every optional param so OpenAI strict-mode clients, which must send every property, can fill the unset ones — so repurposing `null` as "clear" would detach subtasks on any unrelated update those clients make. Fields that need clearing get an explicit boolean instead: `clear_assignees`, `clear_parent_task`. `TestTaskUpdateNullParentTaskIDIsNotAClear` pins the null semantics.
- The MCP SDK validates a tool's `InputSchema` before the handler runs, so a parameter listed in `Required` can never be defaulted or aliased in Go. Widening a shipped tool's parameter while still accepting the old form (as `move_task_to_workflow_stage` does for the scalar `task_id` it advertised before it took `task_ids`) means leaving the new parameter out of `Required` and enforcing it in the handler.
- Annotation hints are mandatory and must be explicit: every tool sets `ReadOnlyHint`, `DestructiveHint` and `OpenWorldHint`. The latter two are `*bool` in the SDK, so a nil value is omitted from `tools/list` and the spec then defaults it to `true`; OpenAI's app review rejects tools with missing hints. Use `new(false)` unless the tool truly destroys data (deletes, plus Desk ticket create/reply, which email the customer) or reaches outside the customer's Teamwork account (help-doc articles published to a public knowledge base, Desk ticket create/reply). `TestAnnotationHintsAreExplicit` in `cmd/docs-gen/main_test.go` guards this across all four products with `allowDelete=true`.
- JSON-Schema gotcha (OpenAI Responses API): every `Type: "array"` node — including inside `AnyOf`/`OneOf`/`AllOf` branches — must declare `Items`. OpenAI rejects bare arrays at tool-registration time even with `strict: false`; Anthropic does not, so Claude Desktop hides the bug. `TestToolInputSchemasArrayItems` in `internal/twprojects/tools_test.go` guards this — if it fires, pick the right item schema rather than weakening the test.
- Date and date-time parameters go through the binders in `internal/helpers/tool_parser.go`, which accept more than one layout (see `dateTimeLayouts` in `internal/helpers/datetime.go`): RFC 3339, an offset-less date-time, and a plain `YYYY-MM-DD`. Models emit the plain date by default when asked about a range, so a strict RFC 3339 parse costs a failed first call and a visible retry. Do not narrow this back. Use `helpers.DateTimeFilterSchema(...)` rather than an inline schema so every filter advertises both forms, and pass `helpers.EndOfDay()` to the binder for any *upper-bound* filter (`end_date`, `*_before`) — a date-only value there must resolve to the day's last second, or the range silently drops its closing day. Handlers that forward the value as a raw query-string parameter instead of binding it use `helpers.NormalizeDateTime(...)`.
- The calendar events endpoint binds `fields[calendarsEvents]`, not `fields[events]` (SDK v1.20.8+ sends the right key via a `sparsefields:key` marker), and its server-side filtering was broken until an API fix in 2026-08. Verify against the live API before assuming an endpoint honours `fields[...]`.
- `list_*` tools follow a specific contract — see `TaskList` in `internal/twprojects/tasks.go` as the canonical pattern:
  - Expose a `verbose` parameter (default `true`) via `helpers.VerboseSchema()`.
  - Execute the request with `twapi.ExecuteRaw(ctx, engine, req)` and stream the body straight to the caller, instead of decoding into the typed `*XxxListResponse` (avoids re-marshalling and preserves any fields the SDK struct doesn't model).
  - Expose a `fields` parameter via `helpers.FieldsSchema[projects.<Entity>]("<entity>")` whenever the SDK request has a `Filters.Fields.<Entity>` slot, parsed with `helpers.OptionalFieldsParam[projects.<Entity>](&req.Filters.Fields.<Entity>, "fields")`. It lets the caller name the attributes it wants instead of choosing between everything and `id` plus a label. Pass the same entity type to both: accepted values are reflected from that struct's JSON attributes, and `FieldsSchema` publishes them as the item schema's enum. The enum is required — it is the only place the model can read the names (tool definitions carry no output schema), so without it callers guess and get rejected. Cost: ~1,500 o200k tokens over 43 tools, +3.1% of the tool-definitions block (`go run ./cmd/mcp-tokens -base=main`), mostly cache-absorbed. With the enum in place the SDK validates unknown names before the handler runs; `OptionalFieldsParam` still rejects them for clients that skip validation, and always appends `id` (rows need it to be addressable by a follow-up `get_*`, and `WebLinker` needs it to attach a web link). Do not expose the sideload slots (`Fields.CustomFields`, `Fields.Projects`, …): they multiply the surface with entities the caller did not ask about. `summarize_timelogs` is deliberately excluded — its `Fields` slots feed an aggregate it computes itself, and its output schema is strict.
  - Do not dedup the enums between a `get_*`/`list_*` pair with `$defs`/`$ref`: each tool publishes its own `inputSchema` document, so a `$ref` would point outside it and the protocol has no shared root. Within one tool the enum appears once, so `$defs` saves nothing.
  - When `fields` is supplied it wins over `verbose`, and no sideloads are requested: sideloading would hand back the bulk the selection exists to avoid. Guard the `verbose` defaults with `len(req.Filters.Fields.<Entity>) == 0` (or a leading `switch` case) so neither branch overwrites an explicit selection. `TestSparseFields*` in `internal/twprojects/sparse_fields_test.go` covers every list and get tool from one table and fails when a tool declares `fields` without being listed there. `TestSparseFieldsSchemaEnumMatchesValidator` is the one that catches a schema wired to a different entity than its validator.
  - When `verbose=false`: set sparse fields on `req.Filters.Fields.<Entity>` to a minimal set (typically `id` + name/title) and skip any hardcoded `Filters.Include` sideloads.
  - When `verbose=true`: include sideloads and the full field set.
  - Wrap the published output schema with `helpers.WithOptionalFields(...)` at the `OutputSchema:` line, not in the `init()` block: the schema vars are shared with `WithMetaWebLinkSchema` and the relaxation belongs to the tool that can return a sparse body. This clears every nested `required` array so sparse responses returned when `verbose=false` still validate. `StructuredContent` is always populated in both modes. It also nils every `additionalProperties`, so the value shape of an SDK `map[string]any` (a search hit's `meta`) has to go in the property's `description` — see `withSearchHighlightsSchema`.
  - OpenAI strict-mode caveat: because a sparse response returns a subset of fields, these output schemas cannot satisfy OpenAI's strict structured-output mode (which requires every property to be `required` and forbids `additionalProperties`). Clients must run with strict mode disabled.
- `get_*` tools whose SDK request has a `Fields <Entity>GetFields` slot expose the same `fields` parameter — see `TaskGet` in `internal/twprojects/tasks.go`:
  - The slot lives on the request root (`req.Fields.<Entity>`), not under `Filters`, so it is `helpers.OptionalFieldsParam[projects.<Entity>](&req.Fields.<Entity>, "fields")`. The schema side is the same `helpers.FieldsSchema[projects.<Entity>]("<entity>")` as the list tool.
  - A selection must be answered with `helpers.NewRawToolResult(...)` rather than the tool's usual `projects.<Entity>Get(...)` plus `json.Marshal`. The SDK entity structs carry no `omitempty`, so re-marshalling the typed response emits every attribute the caller excluded as a zero value — `"dueDate": null` is indistinguishable from a task with no due date. Reset `req.Filters` on that path too, to drop the sideloads the handler requests by default. `TestSparseFieldsGetOmitsUnselectedAttributes` and `TestSparseFieldsGetDropsSideloads` pin both halves.
  - The unselected path is left alone: it still returns the typed round-trip, so existing callers see no change.
  - Wrapping the output schema with `helpers.WithOptionalFields(...)` is required here too, which is why `get_*` schemas are no longer strict. Sparse fieldsets on single entities arrived in twapi-go-sdk v1.20.7; before that, `get_*` could not return a subset and stayed strict-compatible.
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
