package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

	httpClient := resources.TeamworkHTTPClient()
	if *mcpToken != "" {
		httpClient.Transport = newAuthRoundTripper(*mcpToken, httpClient.Transport)
	}

	mcpTransport := &mcp.SSEClientTransport{
		Endpoint:   *mcpURL,
		HTTPClient: httpClient,
	}

	ctx := context.Background()
	_, mcpClientSession, err := config.NewMCPClient(ctx, resources, mcpTransport, &mcp.ClientOptions{})
	if err != nil {
		resources.Logger().Error("failed to create MCP client",
			slog.String("error", err.Error()),
		)
		exit(exitCodeSetupFailure)
	}
	defer func() {
		if err := mcpClientSession.Close(); err != nil {
			resources.Logger().Error("failed to close MCP client session",
				slog.String("error", err.Error()),
			)
		}
	}()

	initResult := mcpClientSession.InitializeResult()
	resources.Logger().Info("MCP client created successfully",
		slog.String("server_name", initResult.ServerInfo.Name),
		slog.String("server_version", initResult.ServerInfo.Version),
		slog.String("protocol_version", initResult.ProtocolVersion),
	)

	args := flag.CommandLine.Args()
	if len(args) < 1 {
		resources.Logger().Error("no command provided")
		exit(exitCodeSetupFailure)
	}

	switch args[0] {
	case "list-tools":
		toolsResult, err := mcpClientSession.ListTools(ctx, &mcp.ListToolsParams{})
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

		toolResult, err := mcpClientSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      toolName,
			Arguments: toolParams,
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

type authRoundTripper struct {
	token string
	next  http.RoundTripper
}

func newAuthRoundTripper(token string, next http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{
		token: token,
		next:  next,
	}
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+a.token)
	return a.next.RoundTrip(req)
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
