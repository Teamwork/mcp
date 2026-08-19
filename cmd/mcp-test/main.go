// mcp-test walks MCP tool handlers through a real Teamwork.com site, creating
// and deleting real data. It bypasses the MCP server transport and invokes each
// tool's Handler directly with the same JSON payload an LLM would send, so you
// can see end-to-end behaviour without standing up an MCP server or LLM client.
//
// The checks are grouped into suites, one per area of the tool surface, chosen
// with -suite. See README.md for configuration and for how to add a suite.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/session"
)

// cleanupTimeout bounds the delete calls made after the steps finish. Cleanup
// runs on a context detached from the signal handler, so it needs its own
// deadline to stay interruptible.
const cleanupTimeout = 30 * time.Second

func main() {
	defer handleExit()

	server := flag.String("server", os.Getenv("TWAPI_SERVER"), "Teamwork base URL (env TWAPI_SERVER)")
	token := flag.String("token", os.Getenv("TWAPI_TOKEN"), "Bearer token / API key (env TWAPI_TOKEN)")
	projectID := flag.Int64("project", envInt64("PROJECT_ID"), "Project ID to test on (env PROJECT_ID)")
	suiteName := flag.String("suite", defaultSuite, "Suite to run: "+strings.Join(suiteNames(), ", ")+", all")
	keep := flag.Bool("keep", false, "Don't delete created artefacts so you can inspect them in the UI")
	step := flag.Bool("step", false, "Pause for ENTER between steps")
	flag.Parse()

	if *server == "" || *token == "" || *projectID == 0 {
		fmt.Fprintln(os.Stderr, "server, token and project are all required (set as env vars or flags)")
		flag.Usage()
		exit(exitCodeSetupFailure)
	}
	*server = strings.TrimSuffix(*server, "/")

	selected, err := selectSuites(*suiteName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		exit(exitCodeSetupFailure)
	}

	sess, err := newSession(*server, *token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		exit(exitCodeSetupFailure)
	}

	// Ctrl-C cancels the steps but still runs cleanup, so an interrupted run
	// doesn't strand artefacts on a real site.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Handlers read the customer URL from the context to attach meta.webLink,
	// the way the servers inject it after detecting the installation. Without it
	// helpers.WebLinker silently returns the payload untouched.
	ctx = twctx.WithCustomerURL(ctx, *server)

	r := &runner{
		engine:    twapi.NewEngine(sess),
		projectID: *projectID,
		keep:      *keep,
		step:      *step,
		stdin:     bufio.NewReader(os.Stdin),
	}

	if err := r.run(ctx, selected); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		exit(exitCodeRunFailure)
	}
	fmt.Println("DONE.")
}

// newSession builds the Teamwork session, picking the auth mode from TWAPI_AUTH
// or, when unset, from the token prefix.
func newSession(server, token string) (twapi.Session, error) {
	authMode := strings.ToLower(strings.TrimSpace(os.Getenv("TWAPI_AUTH")))
	if authMode == "" {
		if strings.HasPrefix(token, "twp_") {
			authMode = "basic"
		} else {
			authMode = "bearer"
		}
	}
	switch authMode {
	case "basic":
		fmt.Println("auth: HTTP Basic (personal API key)")
		return session.NewBasicAuth(token, "x", server), nil
	case "bearer":
		fmt.Println("auth: Bearer token")
		return session.NewBearerToken(token, server), nil
	default:
		return nil, fmt.Errorf("unknown TWAPI_AUTH %q (want basic|bearer)", authMode)
	}
}

// ---------------------------------------------------------------------------
// Suites
// ---------------------------------------------------------------------------

// suite exercises one area of the MCP tool surface. To cover another toolset,
// implement this in its own file and register it in suiteRegistry.
type suite interface {
	// steps are the checks to run, in order. A step returning an error aborts
	// the suite.
	steps() []step

	// cleanup deletes everything the steps created. It runs even when a step
	// failed, and is skipped when -keep is set.
	cleanup(ctx context.Context)

	// artefacts describes what the steps created, one line each, printed
	// instead of cleanup when -keep is set.
	artefacts() []string
}

// step is a single labelled check within a suite.
type step struct {
	label string
	fn    func(ctx context.Context) error
}

// registeredSuite pairs the value -suite accepts with the suite's constructor.
type registeredSuite struct {
	name string
	new  func(r *runner) suite
}

// suiteRegistry lists every suite, in the order -suite=all runs them. Add a new
// toolset's suite here; the first entry is the default.
var suiteRegistry = []registeredSuite{
	{name: "custom-items", new: newCustomItemsSuite},
}

// defaultSuite is what -suite runs when not given.
var defaultSuite = suiteRegistry[0].name

// suiteNames lists the registered names, for flag help and errors.
func suiteNames() []string {
	names := make([]string, len(suiteRegistry))
	for i, registered := range suiteRegistry {
		names[i] = registered.name
	}
	return names
}

// selectSuites resolves a -suite value to the suites to run.
func selectSuites(name string) ([]registeredSuite, error) {
	if name == "all" {
		return suiteRegistry, nil
	}
	for _, registered := range suiteRegistry {
		if registered.name == name {
			return []registeredSuite{registered}, nil
		}
	}
	return nil, fmt.Errorf("unknown suite %q (want %s or all)", name, strings.Join(suiteNames(), ", "))
}

// ---------------------------------------------------------------------------
// Runner
// ---------------------------------------------------------------------------

type runner struct {
	engine    *twapi.Engine
	projectID int64
	keep      bool
	step      bool
	stdin     *bufio.Reader
}

func (r *runner) run(ctx context.Context, suites []registeredSuite) error {
	for _, registered := range suites {
		fmt.Printf("\n########## suite: %s ##########\n", registered.name)
		if err := r.runSuite(ctx, registered.new(r)); err != nil {
			return fmt.Errorf("suite %s: %w", registered.name, err)
		}
	}
	return nil
}

func (r *runner) runSuite(ctx context.Context, s suite) error {
	defer r.finish(ctx, s)

	for _, st := range s.steps() {
		if err := ctx.Err(); err != nil {
			return err
		}
		fmt.Printf("\n=== %s ===\n", st.label)
		if r.step {
			fmt.Print("press ENTER to continue: ")
			if _, err := r.stdin.ReadString('\n'); err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		}
		if err := st.fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// finish either reports the artefacts for inspection or deletes them. Cleanup
// gets a context detached from ctx so it still runs after a Ctrl-C.
func (r *runner) finish(ctx context.Context, s suite) {
	artefacts := s.artefacts()
	if len(artefacts) == 0 {
		// Nothing was created — usually the first step failed — so there is
		// nothing to report or delete.
		return
	}

	if r.keep {
		fmt.Println("\n--- KEEP set, leaving artefacts in place ---")
		for _, artefact := range artefacts {
			fmt.Printf("  %s\n", artefact)
		}
		return
	}

	fmt.Println("\n--- cleanup ---")
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	s.cleanup(cleanupCtx)
}

// ---------------------------------------------------------------------------
// MCP invocation helpers
// ---------------------------------------------------------------------------

// callTool marshals args to JSON and invokes the tool's handler directly.
// Returns the structured text result and any error from the handler, plus a
// bool indicating whether the result was an error result (IsError).
func (r *runner) callTool(
	ctx context.Context,
	tool toolsets.ToolWrapper,
	args map[string]any,
) (text string, isError bool, err error) {
	raw, err := json.Marshal(args)
	if err != nil {
		return "", false, fmt.Errorf("marshal args: %w", err)
	}
	request := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      tool.Tool.Name,
			Arguments: json.RawMessage(raw),
		},
	}
	result, err := tool.Handler(ctx, request)
	if err != nil {
		return "", false, err
	}
	if result == nil {
		return "", false, errors.New("nil result")
	}
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			return text.Text, result.IsError, nil
		}
	}
	return "", result.IsError, errors.New("no text content in result")
}

// callToolExpectOK calls a tool, prints the response, and returns an error
// if the handler returned err or an IsError result.
func (r *runner) callToolExpectOK(
	ctx context.Context,
	label string,
	tool toolsets.ToolWrapper,
	args map[string]any,
) (string, error) {
	text, isError, err := r.callTool(ctx, tool, args)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	if isError {
		return "", fmt.Errorf("%s returned error result: %s", label, text)
	}
	prettyPrint(text)
	return text, nil
}

// callToolIgnoreError calls a tool for its side effect only, reporting a
// failure on stderr instead of returning it. Used by cleanup, which must keep
// going after a single delete fails.
func (r *runner) callToolIgnoreError(ctx context.Context, label string, tool toolsets.ToolWrapper, args map[string]any) {
	text, isError, err := r.callTool(ctx, tool, args)
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "  ! %s failed: %v\n", label, err)
	case isError:
		fmt.Fprintf(os.Stderr, "  ! %s failed: %s\n", label, strings.TrimSpace(text))
	}
}

func prettyPrint(text string) {
	// If the body is JSON, pretty-print it; otherwise emit raw.
	var anyVal any
	if err := json.Unmarshal([]byte(text), &anyVal); err == nil {
		formatted, _ := json.MarshalIndent(anyVal, "  ", "  ")
		fmt.Printf("  %s\n", string(formatted))
		return
	}
	fmt.Printf("  %s\n", text)
}

// asAnyInts converts []int64 to []any (what JSON-encoded numeric arrays look
// like after a typical map[string]any unmarshal).
func asAnyInts(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// extractTrailingID pulls the last whitespace-separated token from the text,
// assuming a result like "Custom item created successfully with ID 1234". The
// create handlers return that text and no structured content, so it is the only
// place the new ID is published.
func extractTrailingID(text string) (int64, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, errors.New("empty text")
	}
	parts := strings.Fields(trimmed)
	last := parts[len(parts)-1]
	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("trailing token %q is not an integer: %w", last, err)
	}
	return id, nil
}

// envInt64 reads an int64 env var, treating unset and unparseable alike so the
// flag default falls back to zero.
func envInt64(key string) int64 {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

// ---------------------------------------------------------------------------
// Exit handling
// ---------------------------------------------------------------------------

type exitCode int

const (
	exitCodeOK exitCode = iota
	exitCodeRunFailure
	exitCodeSetupFailure
)

type exitData struct {
	code exitCode
}

// exit allows to abort the program while still executing all defer statements,
// so a failure after the steps started still runs cleanup.
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
