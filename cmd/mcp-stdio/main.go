package main

import (
	"bytes"
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

	resources, teardown := config.Load(os.Stderr)
	defer teardown()

	flag.Var(&methods, "toolsets", "Comma-separated list of toolsets to enable")
	flag.BoolVar(&readOnly, "read-only", false, "Restrict the server to read-only operations")
	flag.Parse()

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
		mcpError(resources.Logger(), fmt.Errorf("failed to create MCP server: %s", err), mcp.INTERNAL_ERROR)
		exit(exitCodeSetupFailure)
	}
	mcpSTDIOServer := server.NewStdioServer(mcpServer)
	stdinWrapper := newStdinWrapper(resources.Logger(), authenticated)
	if err := mcpSTDIOServer.Listen(ctx, &stdinWrapper, os.Stdout); err != nil {
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
	logger        *slog.Logger
	buffer        []byte
	authenticated bool
}

func newStdinWrapper(logger *slog.Logger, authenticated bool) stdinWrapper {
	return stdinWrapper{
		logger:        logger,
		authenticated: authenticated,
	}
}

func (s *stdinWrapper) Read(p []byte) (n int, err error) {
	if s.authenticated {
		return os.Stdin.Read(p)
	}

	buffer := make([]byte, len(p))
	n, err = os.Stdin.Read(buffer)
	if err != nil {
		return n, err
	}
	content := buffer[:n]
	s.buffer = append(s.buffer, content...)

	for {
		lineBreakPos := bytes.Index(s.buffer, []byte("\n"))
		if lineBreakPos == -1 {
			break
		}
		var remaining []byte
		if lineBreakPos+1 > len(s.buffer) {
			remaining = s.buffer[lineBreakPos+1:]
		}
		content = s.buffer[:lineBreakPos]
		s.buffer = remaining

		if len(content) > 0 {
			if bypass, err := auth.Bypass(content); err != nil {
				return 0, err
			} else if !bypass {
				return 0, errors.New("not authenticated")
			}
		}
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
