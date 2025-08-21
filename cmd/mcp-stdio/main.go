package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
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
	methods          = methodsInput([]toolsets.Method{toolsets.MethodAll})
	methodsWhitelist = []string{
		// allow some protocol methods to bypass authentication
		//
		// https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle
		// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#listing-tools
		// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#listing-resources
		// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#resource-templates
		// https://modelcontextprotocol.io/specification/2025-06-18/server/prompts#listing-prompts
		"initialize",
		"notifications/initialized",
		"logging/setLevel",
		"tools/list",
		"resources/list",
		"resources/templates/list",
		"prompts/list",
	}
	readOnly bool
)

func main() {
	defer handleExit()

	resources, teardown := config.Load()
	defer teardown()

	flag.Var(&methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.BoolVar(&readOnly, "read-only", false, "Restrict the server to read-only operations")
	flag.Parse()

	ctx := context.Background()

	if resources.Info.BearerToken != "" {
		// detect the installation from the bearer token
		info, err := auth.GetBearerInfo(ctx, resources, resources.Info.BearerToken)
		if err != nil {
			mcpError(resources.Logger(), fmt.Errorf("failed to authenticate: %s", err), mcp.INVALID_PARAMS)
			exit(exitCodeSetupFailure)
		}

		// inject customer URL in the context
		ctx = config.WithCustomerURL(ctx, info.URL)
		// inject bearer token in the context
		ctx = session.WithBearerTokenContext(ctx, session.NewBearerToken(resources.Info.BearerToken, info.URL))
	}

	mcpServer, err := newMCPServer(resources)
	if err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to create MCP server: %s", err), mcp.INTERNAL_ERROR)
		exit(exitCodeSetupFailure)
	}
	mcpSTDIOServer := server.NewStdioServer(mcpServer)
	stdinWrapper := newStdinWrapper(resources.Logger(), resources.Info.BearerToken != "", methodsWhitelist)
	if err := mcpSTDIOServer.Listen(ctx, stdinWrapper, os.Stdout); err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to serve: %s", err), mcp.INTERNAL_ERROR)
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

func mcpError(logger *slog.Logger, err error, code int) {
	mcpError := mcp.NewJSONRPCError(mcp.NewRequestId("startup"), code, err.Error(), nil)
	encoded, err := json.Marshal(mcpError)
	if err != nil {
		logger.Error("failed to encode error",
			slog.String("error", err.Error()),
		)
		return
	}
	fmt.Printf("%s\n", string(encoded))
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

type stdinWrapper struct {
	logger           *slog.Logger
	authenticated    bool
	methodsWhitelist []string
}

func newStdinWrapper(logger *slog.Logger, authenticated bool, methods []string) stdinWrapper {
	return stdinWrapper{
		logger:           logger,
		authenticated:    authenticated,
		methodsWhitelist: methods,
	}
}

func (s stdinWrapper) Read(p []byte) (n int, err error) {
	if s.authenticated {
		return os.Stdin.Read(p)
	}
	buffer := make([]byte, len(p))
	n, err = os.Stdin.Read(buffer)
	if err != nil {
		return n, err
	}
	content := buffer[:n]
	if len(content) == 0 {
		return n, err
	}
	var baseMessage struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(content, &baseMessage); err != nil {
		return 0, errors.New("parse error")
	}
	if !slices.Contains(s.methodsWhitelist, baseMessage.Method) {
		return 0, errors.New("not authenticated")
	}
	copy(p, buffer)
	return n, err
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
