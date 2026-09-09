# mcp-vertex-check

Validates the published tool schemas against the subset of JSON Schema that
Vertex AI can parse — and therefore against what Gemini Enterprise will accept
when it loads "custom actions". It answers "would Gemini load these tools?"
locally, instead of through a customer retrying a connector.

```sh
go run ./cmd/mcp-vertex-check              # what a Gemini client gets; must be clean
go run ./cmd/mcp-vertex-check -published   # the raw published surface; will not be clean
go run ./cmd/mcp-vertex-check -warn        # include warnings
go run ./cmd/mcp-vertex-check -input-only  # inputSchema only
go run ./cmd/mcp-vertex-check -json        # machine-readable
go run ./cmd/mcp-vertex-check -tools=captured.json   # a saved body, checked as it stands
```

**The default checks what a Gemini client is served**, not what the server
publishes — that is the surface which has to be clean, and the published one
cannot be (see below). Exit status is 1 when anything is reported and 2 on a
usage error, so the default mode works as a CI gate.

## Why this is not the same as "the schema is valid"

Vertex models a tool schema as a protobuf message, not as JSON Schema. Its
`Schema` message is a declared subset of **OpenAPI 3.0.3**
(`google/cloud/aiplatform/v1/openapi.proto`), so a document can be perfectly
valid JSON Schema and still be unparseable there. Two consequences carry most of
the weight:

- `type` is a **single** enum field with members `string`, `number`, `integer`,
  `boolean`, `array`, `object`. There is no `NULL`, and a *list* cannot be
  parsed at all — Gemini clients report
  `Proto field is not repeating, cannot start list`.
- Any keyword outside the message's fields is an unknown field. `examples`
  (JSON Schema 2020-12, plural) is not one of them; the message has singular
  `example`.

Note this is the opposite of OpenAPI 3.1+, which adopted JSON Schema 2020-12 and
made `["string","null"]` the sanctioned nullable form. Chasing the newest spec
moves *away* from what Vertex accepts.

## Severities

- **error** — contradicts the `Schema` message, so it cannot be parsed.
- **warn** — permitted by the message but reported rejected by Gemini clients,
  or outside the documented value set. `additionalProperties` as a schema rather
  than a boolean is the clearest case: the field is a `Value`, so the proto
  allows it, but Gemini answers `Expected boolean, received object`
  (gemini-cli #13694, closed as not planned). `many-tools` is another: at most
  128 function declarations reach the model per request, but Gemini Enterprise
  imports a connector's actions and enables them individually, so a longer list
  does not stop them loading.

## Output schemas are checked, and that is deliberate

Vertex's `FunctionDeclaration` carries `Schema response = 4` alongside
`parameters`, and Gemini clients do validate an MCP tool's `outputSchema`. Pass
`-input-only` to scope a run to what a `tools/list` caller sends, but a clean
`-input-only` run does **not** mean Gemini will load the tools.

## Two surfaces, and only one of them passes

The output schemas cannot be fixed the way the input ones were: their nulls are
real wire values (`twapi.Date` encodes as `null` when unset), so removing them
would publish a schema every response violates.

So a Gemini client is served a rewritten copy instead — `helpers.VertexTools`,
wired to `wantsVertexSchemas` in `pkg/config` — and that copy is what the
default mode checks:

```sh
go run ./cmd/mcp-vertex-check              # the copy a Gemini client gets: clean
go run ./cmd/mcp-vertex-check -published   # the raw surface: 602 errors, and must be
```

`TestVertexVariantIsClean` is the gate, and
`TestPublishedSurfaceStillCarriesOutputSchemaNulls` stops it passing vacuously —
if the published surface ever becomes Vertex-safe on its own, the variant is no
longer needed.

## Adding a rule

Rules live in `vertex.go`, driven by `vertexFields`, `vertexTypes` and
`documentedFormats`. Take a new rule from the proto or from a reproducible
client error, and put it at `SeverityError` only when the proto says it cannot
parse — a guess at `error` turns this tool into noise. `vertex_test.go` drives
each rule from one table.

`TestVertexVariantIsClean` gates the variant in CI. The published surface is
deliberately not gated: it cannot pass while its output schemas carry real
nulls.
