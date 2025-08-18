package twdesk_test

import (
	"github.com/teamwork/mcp/internal/testutil"
)

// Re-export the shared utilities for convenience
var (
	mcpServerMock = testutil.DeskMCPServerMock
	checkMessage  = testutil.CheckMessage
)

type toolRequest = testutil.ToolRequest
