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
- Run `go test ./internal/twprojects` until green.

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
- `usage.md` — end-user connection guide for popular MCP clients
- `cmd/mcp-stdio/README.md` — STDIO server flags and envs
- `cmd/mcp-http/README.md` — HTTP server envs and endpoints
- `cmd/mcp-http-cli/README.md` — CLI usage
- `internal/twprojects/tools.go` — tool registration hub

---
Treat this file as living documentation. Update it alongside changes to commands, flags, and conventions so agents can reliably build, test, and ship edits.
