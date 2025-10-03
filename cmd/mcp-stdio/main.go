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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/auth"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk/session"
)

var (
	methods   = methodsInput([]toolsets.Method{toolsets.MethodAll})
	readOnly  bool
	logToFile string
)

func main() {
	defer handleExit()

	flag.Var(&methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.StringVar(&logToFile, "log-to-file", "", "Path to log file (if empty, logs to stderr)")
	flag.BoolVar(&readOnly, "read-only", false, "Restrict the server to read-only operations")
	flag.Parse()

	f := os.Stderr
	if logToFile != "" {
		var err error
		f, err = os.Open(logToFile)
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
		if info, err := auth.GetBearerInfo(ctx, resources, resources.Info.BearerToken); err != nil {
			resources.Logger().Error("failed to get bearer info",
				slog.String("error", err.Error()),
			)
		} else {
			authenticated = true
			// inject customer URL in the context
			ctx = config.WithCustomerURL(ctx, info.URL)
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

	if err := mcpServer.Run(ctx, &mcp.StdioTransport{}); err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to serve: %s", err), jsonRPCErrorCodeInternalError)
		exit(exitCodeSetupFailure)
	}
}

func newMCPServer(resources config.Resources) (*mcp.Server, error) {
	projectsGroup := twprojects.DefaultToolsetGroup(readOnly, false, resources.TeamworkEngine())
	if err := projectsGroup.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable projects toolsets: %w", err)
	}

	deskGroup := twdesk.DefaultToolsetGroup(resources.DeskClient())
	if err := deskGroup.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable desk toolsets: %w", err)
	}

	return config.NewMCPServer(resources, projectsGroup, deskGroup), nil
}

func mcpError(logger *slog.Logger, err error, code jsonRPCErrorCode) {
	encoded, err := json.Marshal(jsonRPCError{
		Code:    code,
		Message: err.Error(),
	})
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

type jsonRPCErrorCode int64

const (
	jsonRPCErrorCodeParse          jsonRPCErrorCode = -32700
	jsonRPCErrorCodeInvalidRequest jsonRPCErrorCode = -32600
	jsonRPCErrorCodeMethodNotFound jsonRPCErrorCode = -32601
	jsonRPCErrorCodeInvalidParams  jsonRPCErrorCode = -32602
	jsonRPCErrorCodeInternalError  jsonRPCErrorCode = -32603
)

// jsonRPCError represents a JSON-RPC level error.
//
// https://www.jsonrpc.org/specification#error_object
//
// The library does not export this type, so we need to redefine it here.
// https://github.com/modelcontextprotocol/go-sdk/blob/1dcbf62661fc9c54ae364e0af80433db347e2fc4/internal/jsonrpc2/wire.go#L66-L74
//
//nolint:lll
type jsonRPCError struct {
	// Code is an error code indicating the type of failure.
	Code jsonRPCErrorCode `json:"code"`
	// Message is a short description of the error.
	Message string `json:"message"`
	// Data is optional structured data containing additional information about the error.
	Data json.RawMessage `json:"data,omitempty"`
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
