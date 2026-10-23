# Teamwork.com plugin

A portable Agent Plugins 1.0 package for the hosted Teamwork.com MCP server at `https://mcp.ai.teamwork.com`, with workflow guidance for Projects, Desk, and available Spaces and Chat tools.

## Connect

Import this package into a client that supports Agent Plugins 1.0 remote MCP connections. Complete the client's Teamwork.com OAuth sign-in and consent flow. The MCP connection uses streamable HTTP with OAuth discovery and dynamic client registration; the default requested product scopes are `projects`, `desk`, `spaces`, and `chat`. Available tools depend on the granted scopes and account permissions.

A Teamwork.com administrator may need to enable MCP under Settings → AI. See the [Teamwork.com MCP usage guide](https://github.com/teamwork/mcp/blob/main/docs/usage/README.md). This package contains no tokens, client secrets, or local server binaries.

## Contents

- `plugin.json`: portable identity, listing, and light/dark appearance.
- `mcp.json`: hosted server transport and OAuth connection configuration.
- `skills/teamwork-workflows/SKILL.md`: guidance for resolving records, making authorized changes, and reporting results.
- `assets/`: square variants of the mark supplied by the Teamwork.com MCP repository.
- `.codex-plugin/plugin.json`: compatibility metadata for clients supporting the legacy manifest layout, referencing the same `mcp.json`. Remote transport and OAuth support still depend on the client.

The original mark's paths and colors are preserved with square padding. The dark-mode asset adds a white backing for contrast against dark surfaces. Light-mode accent uses the supplied navy `#1D1C39`; dark-mode accent uses the mark's pink `#FF22B1`.

This is a source package. Creating or exporting it does not install the plugin, publish a listing, or prove an authenticated connection. Connect it in a supported client to verify account-specific tool discovery and behavior.

## Submission metadata (0.1.1)

The portable manifest includes Teamwork support, privacy notice, and terms of service URLs, package release notes, and five positive plus three negative review cases. MCP server history is available in the [server releases](https://github.com/Teamwork/mcp/releases). All eight cases were executed against a dedicated demo account; see the review notes for outcomes and caveats. The [hosted walkthrough](https://youtu.be/b6myj3Mpkfw) was verified playable without YouTube sign-in and is included in the review metadata. Secure reviewer OAuth sign-in details remain required in the submission portal. The publisher confirmed availability in all supported countries and no purchases or payments.
