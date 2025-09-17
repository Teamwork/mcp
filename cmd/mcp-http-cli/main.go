package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/teamwork/mcp/internal/config"
)

var (
	mcpURL = flag.String("mcp-url", "https://mcp.ai.teamwork.com",
		"The URL of the MCP server to connect to")
	mcpToken = flag.String("mcp-token", os.Getenv("TW_MCP_BEARER_TOKEN"),
		"The token to use for authentication with the MCP server")
)

func main() {
	defer handleExit()

	resources, teardown := config.Load(os.Stdout)
	defer teardown()

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		resources.Logger().Error("failed to parse global flags",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}

	if *mcpURL == "" {
		resources.Logger().Error("MCP URL is required")
		exit(exitCodeSetupFailure)
	}

	var options []transport.StreamableHTTPCOption
	if *mcpToken != "" {
		options = append(options, transport.WithHTTPHeaders(map[string]string{
			"Authorization": "Bearer " + *mcpToken,
		}))
	}

	mcpTransport, err := transport.NewStreamableHTTP(*mcpURL, options...)
	if err != nil {
		resources.Logger().Error("failed to create MCP transport",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}

	ctx := context.Background()
	mcpClient, mcpServerInfo, err := config.NewMCPClient(ctx, resources, mcpTransport)
	if err != nil {
		resources.Logger().Error("failed to create MCP client",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}

	resources.Logger().Info("MCP client created successfully",
		slog.String("server_name", mcpServerInfo.ServerInfo.Name),
		slog.String("server_version", mcpServerInfo.ServerInfo.Version),
		slog.String("protocol_version", mcpServerInfo.ProtocolVersion),
	)

	args := flag.CommandLine.Args()
	if len(args) < 1 {
		resources.Logger().Error("no command provided")
		exit(exitCodeSetupFailure)
	}

	switch args[0] {
	case "list-tools":
		toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			resources.Logger().Error("failed to list tools",
				slog.String("error", err.Error()),
			)
			exit(exitCodeRunFailure)
		}

		for _, tool := range toolsResult.Tools {
			resources.Logger().Info("tool",
				slog.String("name", tool.Name),
				slog.String("description", tool.Description),
			)
		}
	case "call-tool":
		if len(args) < 2 {
			resources.Logger().Error("no tool name provided")
			exit(exitCodeSetupFailure)
		}
		toolName := args[1]

		var toolParams map[string]any
		if len(args) > 2 {
			if err := json.Unmarshal([]byte(args[2]), &toolParams); err != nil {
				resources.Logger().Error("failed to parse tool arguments",
					slog.String("error", err.Error()),
				)
				exit(exitCodeSetupFailure)
			}
		}

		toolResult, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Request: mcp.Request{
				Method: toolName,
			},
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: toolParams,
			},
		})
		if err != nil {
			resources.Logger().Error("failed to run tool",
				slog.String("tool_name", toolName),
				slog.String("error", err.Error()),
			)
			exit(exitCodeRunFailure)
		}

		if toolResult.IsError {
			resources.Logger().Error("tool execution failed",
				slog.String("tool_name", toolName),
				slog.Any("error", toolResult.Content),
			)
			exit(exitCodeRunFailure)
		}

		resources.Logger().Info("tool executed successfully",
			slog.String("tool_name", toolName),
			slog.Any("result", toolResult.Content),
		)

	default:
		resources.Logger().Error("unknown command",
			slog.String("command", args[0]),
			slog.String("available_commands", "list-tools, call-tool"),
		)
		exit(exitCodeSetupFailure)
	}
}

type exitCode int

const (
	exitCodeOK exitCode = iota
	exitCodeSetupFailure
	exitCodeRunFailure
)

type exitData struct {
	code exitCode
}

// exit allows to abort the program while still executing all defer statements.
func exit(code exitCode) {
	panic(exitData{code: code})
}

// handleExit exit code handler.
func handleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(exitData); ok {
			os.Exit(int(exit.code))
		}
		panic(e)
	}
}
