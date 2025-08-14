package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/auth"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/session"
)

var (
	methods  = methodsInput([]toolsets.Method{toolsets.MethodAll})
	readOnly bool
)

func main() {
	defer handleExit()

	resources, teardown := config.Load()
	defer teardown()

	flag.Var(&methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.BoolVar(&readOnly, "read-only", false, "Restrict the server to read-only operations")
	flag.Parse()

	if resources.Info.BearerToken == "" {
		mcpError(resources, errors.New("TEAMWORK_BEARER_TOKEN environment variable is not set"), mcp.INVALID_PARAMS)
		exit(exitCodeSetupFailure)
	}

	ctx := context.Background()

	// detect the installation from the bearer token
	info, err := auth.GetBearerInfo(ctx, resources, resources.Info.BearerToken)
	if err != nil {
		mcpError(resources, fmt.Errorf("failed to authenticate: %s", err), mcp.INVALID_PARAMS)
		exit(exitCodeSetupFailure)
	}

	// inject customer URL in the context
	ctx = config.WithCustomerURL(ctx, info.URL)
	// inject bearer token in the context
	ctx = session.WithBearerTokenContext(ctx, session.NewBearerToken(resources.Info.BearerToken, info.URL))

	mcpServer, err := newMCPServer(resources)
	if err != nil {
		mcpError(resources, fmt.Errorf("failed to create MCP server: %s", err), mcp.INTERNAL_ERROR)
		exit(exitCodeSetupFailure)
	}
	mcpSTDIOServer := server.NewStdioServer(mcpServer)
	if err := mcpSTDIOServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		mcpError(resources, fmt.Errorf("failed to serve: %s", err), mcp.INTERNAL_ERROR)
		exit(exitCodeSetupFailure)
	}
}

func newMCPServer(resources config.Resources) (*server.MCPServer, error) {
	group := twprojects.DefaultToolsetGroup(readOnly, false, resources.TeamworkEngine())
	if err := group.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}
	return config.NewMCPServer(resources, group), nil
}

func mcpError(resources config.Resources, err error, code int) {
	mcpError := mcp.NewJSONRPCError(mcp.NewRequestId("startup"), code, err.Error(), nil)
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(mcpError); err != nil {
		resources.Logger().Error("failed to encode error",
			slog.String("error", err.Error()),
		)
	}
}

type methodsInput []toolsets.Method

func (t methodsInput) String() string {
	methods := make([]string, len(t))
	for i, m := range t {
		methods[i] = m.String()
	}
	return strings.Join(methods, ", ")
}

func (t *methodsInput) Set(value string) error {
	if value == "" {
		return nil
	}
	*t = (*t)[:0] // reset slice

	var errs error
	for methodString := range strings.SplitSeq(value, ",") {
		if method := toolsets.Method(strings.TrimSpace(methodString)); method.IsRegistered() {
			*t = append(*t, method)
		} else {
			errs = errors.Join(errs, fmt.Errorf("invalid toolset method: %q", methodString))
		}
	}
	return errs
}

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
