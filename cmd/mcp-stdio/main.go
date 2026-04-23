package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/auth"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/twapi-go-sdk/session"
)

var (
	methods   = methodsInput([]toolsets.Method{toolsets.MethodAll})
	readOnly  bool
	logToFile string
)

func init() {
	toolsets.RegisterProfile("project-manager", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetContent,
	})
	toolsets.RegisterProfile("support", []toolsets.Method{
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
	})
	toolsets.RegisterProfile("analyst", []toolsets.Method{
		twprojects.ToolsetProjects,
		twprojects.ToolsetTasks,
		twprojects.ToolsetPeople,
		twprojects.ToolsetTime,
		twprojects.ToolsetContent,
		twdesk.ToolsetTickets,
		twdesk.ToolsetCustomers,
		twdesk.ToolsetAdmin,
	})
	toolsets.RegisterProfile("knowledge-manager", []toolsets.Method{
		twspaces.ToolsetSpaces,
		twspaces.ToolsetPages,
		twspaces.ToolsetContent,
	})
	toolsets.RegisterProfile("ops", []toolsets.Method{
		toolsets.MethodAll,
	})
}

func main() {
	defer handleExit()

	flag.Var(&methods, "toolsets", "Comma-separated list of toolsets to enable")
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
		if info, err := auth.GetBearerInfo(ctx, resources, resources.Info.BearerToken); err != nil {
			resources.Logger().Error("failed to get bearer info",
				slog.String("error", err.Error()),
			)
		} else {
			authenticated = true
			// inject customer URL in the context
			ctx = config.WithCustomerURL(ctx, info.URL)
			// inject bearer token in the context (used by Desk SDK clients)
			ctx = config.WithBearerToken(ctx, resources.Info.BearerToken)
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

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				if err := ss.Ping(pingCtx, nil); err != nil {
					mcpError(resources.Logger(), fmt.Errorf("failed to ping: %s", err), jsonRPCErrorCodeInternalError)
				}
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := ss.Wait(); err != nil {
		mcpError(resources.Logger(), fmt.Errorf("failed to serve: %s", err), jsonRPCErrorCodeInternalError)
		exit(exitCodeSetupFailure)
	}
}

func newMCPServer(resources config.Resources) (*mcp.Server, error) {
	projectsGroup := twprojects.DefaultToolsetGroup(readOnly, false, resources.TeamworkEngine())
	if err := projectsGroup.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable projects toolsets: %w", err)
	}

	deskGroup := twdesk.DefaultToolsetGroup(readOnly, resources.TeamworkHTTPClient())
	if err := deskGroup.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable desk toolsets: %w", err)
	}

	spacesGroup := twspaces.DefaultToolsetGroup(readOnly, resources.TeamworkHTTPClient())
	if err := spacesGroup.EnableToolsets(methods...); err != nil {
		return nil, fmt.Errorf("failed to enable spaces toolsets: %w", err)
	}

	return config.NewMCPServer(resources, projectsGroup, deskGroup, spacesGroup), nil
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
	for token := range strings.SplitSeq(value, ",") {
		token = strings.TrimSpace(token)
		// expand named profiles into their constituent methods
		if profileMethods, ok := toolsets.LookupProfile(token); ok {
			*t = append(*t, profileMethods...)
			continue
		}
		if method := toolsets.Method(token); method.IsRegistered() {
			*t = append(*t, method)
		} else {
			errs = errors.Join(errs, fmt.Errorf(`
				invalid toolset: %q (use a sub-toolset key like "twprojects-tasks", 
				a profile like "project-manager", or "all")
			`, token))
		}
	}
	return errs
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
