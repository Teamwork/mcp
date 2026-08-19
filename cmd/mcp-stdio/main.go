package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/cli"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/mcp/pkg/auth"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
	"github.com/teamwork/twapi-go-sdk/session"
)

var (
	methods   = cli.NewMethods(toolsets.MethodAll)
	readOnly  bool
	logToFile string
)

func main() {
	defer handleExit()

	flag.Var(methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.StringVar(&logToFile, "log-to-file", "", "Path to log file (if empty, logs to stderr)")
	flag.BoolVar(&readOnly, "read-only", false, "Restrict the server to read-only operations")
	flag.Parse()

	f := os.Stderr
	if logToFile != "" {
		var err error
		f, err = os.OpenFile(logToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %s\n", err)
			exit(exitCodeSetupFailure)
		}
	}

	defer f.Close() //nolint:errcheck
	resources, teardown := config.Load(f)
	defer teardown()

	ctx := context.Background()

	var authenticated bool
	if resources.Info.BearerToken != "" {
		// detect the installation from the bearer token
		validator := auth.NewValidator(resources.TeamworkHTTPClient(), resources.Info.APIURL, resources.Logger())
		if info, err := validator.GetBearerInfo(ctx, resources.Info.BearerToken); err != nil {
			resources.Logger().Error("failed to get bearer info",
				slog.String("error", err.Error()),
			)
		} else {
			authenticated = true
			// inject customer URL in the context
			ctx = twctx.WithCustomerURL(ctx, info.URL)
			// inject bearer token in the context (used by Desk SDK clients)
			ctx = twctx.WithBearerToken(ctx, resources.Info.BearerToken)
			// inject bearer token in the context
			ctx = session.WithBearerTokenContext(ctx, session.NewBearerToken(resources.Info.BearerToken, info.URL))
		}
	}

	mcpServer, err := newMCPServer(resources)
	if err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to create MCP server: %s", err), jsonRPCErrorCodeInternalError)
		exit(exitCodeSetupFailure)
	}
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			if !authenticated && !auth.BypassMethod(method) {
				return nil, errors.New("not authenticated")
			}
			return next(ctx, method, req)
		}
	})

	ss, err := mcpServer.Connect(ctx, &mcp.StdioTransport{}, nil)
	if err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to connect: %s", err), jsonRPCErrorCodeInternalError)
		exit(exitCodeSetupFailure)
	}

	// Keepalive pings are driven by the SDK itself, via
	// mcp.ServerOptions.KeepAlive in config.NewMCPServer. Do not add a manual
	// ping loop here: on failure it has no request to reply to, so mcpError
	// writes an id-less JSON-RPC error onto stdout, which is the protocol stream.

	if err := ss.Wait(); err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to serve: %s", err), jsonRPCErrorCodeInternalError)
		exit(exitCodeSetupFailure)
	}
}

func newMCPServer(resources config.Resources) (*mcp.Server, error) {
	projectsGroup := twprojects.DefaultToolsetGroup(readOnly, false, resources.TeamworkEngine())
	if err := projectsGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable projects toolsets: %w", err)
	}

	deskGroup := twdesk.DefaultToolsetGroup(readOnly, resources.TeamworkHTTPClient())
	if err := deskGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable desk toolsets: %w", err)
	}

	spacesGroup := twspaces.DefaultToolsetGroup(readOnly, false, resources.TeamworkHTTPClient())
	if err := spacesGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable spaces toolsets: %w", err)
	}

	chatGroup := twchat.DefaultToolsetGroup(readOnly, resources.TeamworkEngine())
	if err := chatGroup.EnableToolsets(methods.Toolsets()...); err != nil {
		return nil, fmt.Errorf("failed to enable chat toolsets: %w", err)
	}

	return config.NewMCPServer(resources, projectsGroup, deskGroup, spacesGroup, chatGroup), nil
}

func mcpError(logger *slog.Logger, err error, code jsonRPCErrorCode) {
	encoded, err := jsonrpc.EncodeMessage(&jsonrpc.Response{
		Error: &jsonrpc.Error{
			Code:    int64(code),
			Message: err.Error(),
		},
	})
	if err != nil {
		logger.Error("failed to encode error",
			slog.String("error", err.Error()),
		)
		return
	}
	fmt.Printf("%s\n", string(encoded))
}

type jsonRPCErrorCode int64

const (
	jsonRPCErrorCodeInternalError jsonRPCErrorCode = jsonrpc.CodeInternalError
)

type exitCode int

const (
	exitCodeOK exitCode = iota
	exitCodeSetupFailure
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
