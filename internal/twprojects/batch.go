package twprojects

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/twapi-go-sdk"
)

const (
	// batchMaxItems bounds a batch tool's fan-out. Each item is one request, so a
	// longer list would outlive the tool call.
	batchMaxItems = 100

	// batchConcurrency is how many requests of one batch run at once, kept low so
	// a batch does not trip the API's rate limits.
	batchConcurrency = 4
)

// batchItemSchema derives the item schema of a batch tool from the input schema
// of its single-record tool, so both advertise the same parameters.
func batchItemSchema(inputSchema any, drop ...string) *jsonschema.Schema {
	single := inputSchema.(*jsonschema.Schema)
	item := &jsonschema.Schema{
		Type:       "object",
		Properties: make(map[string]*jsonschema.Schema, len(single.Properties)),
	}
	for name, property := range single.Properties {
		if !slices.Contains(drop, name) {
			item.Properties[name] = property
		}
	}
	for _, name := range single.Required {
		if !slices.Contains(drop, name) {
			item.Required = append(item.Required, name)
		}
	}
	return item
}

// batchItems reads a batch tool's array of items.
func batchItems(arguments map[string]any, name string, maxItems int) ([]map[string]any, *mcp.CallToolResult) {
	raw, _ := arguments[name].([]any)
	if len(raw) == 0 {
		return nil, helpers.NewToolResultTextError("%s must contain at least one item", name)
	}
	if len(raw) > maxItems {
		return nil, helpers.NewToolResultTextError(
			"%s accepts at most %d items, got %d; split the list across several calls", name, maxItems, len(raw))
	}
	items := make([]map[string]any, len(raw))
	for i, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, helpers.NewToolResultTextError("%s: item %d must be an object", name, i+1)
		}
		items[i] = item
	}
	return items, nil
}

// batchInvalidItems rejects a batch before anything is written, so one bad item
// never leaves the rest half applied.
func batchInvalidItems(invalid []string) *mcp.CallToolResult {
	return helpers.NewToolResultTextError(
		"Nothing was written. Fix these items and send the whole list again:\n%s", strings.Join(invalid, "\n"))
}

// toolResultText returns the text of a result built by helpers.NewToolResultText
// or helpers.NewToolResultTextError.
func toolResultText(result *mcp.CallToolResult) string {
	var texts []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			texts = append(texts, text.Text)
		}
	}
	return strings.Join(texts, " ")
}

var (
	// errBatchNotStarted marks an item the batch never sent, so sending it
	// again is safe.
	errBatchNotStarted = errors.New("not attempted")

	// errBatchInterrupted marks an item cancelled while its request was in
	// flight. The server may have applied it before the client gave up.
	errBatchInterrupted = errors.New("interrupted")
)

// runBatch calls do for each of n items, at most concurrency at a time, and
// returns each item's error in input order. Once ctx ends, items not yet
// started report errBatchNotStarted and items cut off mid-request report
// errBatchInterrupted.
func runBatch(ctx context.Context, n, concurrency int, do func(ctx context.Context, i int) error) []error {
	errs := make([]error, n)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range n {
		sem <- struct{}{}
		if err := ctx.Err(); err != nil {
			<-sem
			errs[i] = fmt.Errorf("%w: %w", errBatchNotStarted, err)
			continue
		}
		wg.Go(func() {
			defer func() { <-sem }()
			err := do(ctx, i)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("%w: %w", errBatchInterrupted, err)
			}
			errs[i] = err
		})
	}
	wg.Wait()
	return errs
}

// batchReport summarises a batch in one result: how many items succeeded, which
// ones, and why each failure failed. It is an error result when any item failed,
// even though the others were written.
//
// Items a cancellation stopped are listed apart from failures, because a model
// retries what it reads as failed: those never sent are safe to resend, while
// those cut off mid-request may exist already, and recheck says how to find out.
func batchReport(verb, noun, recheck string, labels, details []string, errs []error) *mcp.CallToolResult {
	var succeeded, failed, notStarted, interrupted []string
	for i, err := range errs {
		switch {
		case errors.Is(err, errBatchNotStarted):
			notStarted = append(notStarted, labels[i])
		case errors.Is(err, errBatchInterrupted):
			interrupted = append(interrupted, labels[i])
		case err != nil:
			failed = append(failed, fmt.Sprintf("%s: %s", labels[i], err.Error()))
		default:
			label := labels[i]
			if details != nil && details[i] != "" {
				label += " → " + details[i]
			}
			succeeded = append(succeeded, label)
		}
	}

	var report strings.Builder
	fmt.Fprintf(&report, "%s %d of %d %s.", verb, len(succeeded), len(errs), noun)
	if len(succeeded) > 0 {
		fmt.Fprintf(&report, "\n%s: %s.", verb, strings.Join(succeeded, ", "))
	}
	for _, failure := range failed {
		fmt.Fprintf(&report, "\nFailed: %s.", failure)
	}
	if len(interrupted) > 0 {
		fmt.Fprintf(&report, "\nInterrupted, outcome unknown (%s): %s.", recheck, strings.Join(interrupted, ", "))
	}
	if len(notStarted) > 0 {
		fmt.Fprintf(&report, "\nNot attempted, safe to send again: %s.", strings.Join(notStarted, ", "))
	}
	if len(succeeded) < len(errs) {
		return helpers.NewToolResultTextError("%s", report.String())
	}
	return helpers.NewToolResultText("%s", report.String())
}

// bulkFailureReasons joins the reasons a bulk endpoint gave for one entry.
func bulkFailureReasons(errs []twapi.V3Error) string {
	reasons := make([]string, 0, len(errs))
	for _, e := range errs {
		reason := e.Detail
		if reason == "" {
			reason = e.Title
		}
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		return "no reason given"
	}
	return strings.Join(reasons, "; ")
}

// retryIdempotent runs a write that is safe to repeat, retrying it on 429, 500
// and 503. Never use it for a create: a retry can add a duplicate.
func retryIdempotent(ctx context.Context, write func() error) error {
	_, err := helpers.Retry(ctx, helpers.TransientRetry, func() (struct{}, error) {
		return struct{}{}, write()
	})
	return err
}

// retryThrottled retries a write on 429 only, which the API answers before
// applying anything. Use it for writes whose repeat after a 500 that did apply
// may fail or have side effects, such as an email.
func retryThrottled(ctx context.Context, write func() error) error {
	_, err := helpers.Retry(ctx, helpers.ThrottledRetry, func() (struct{}, error) {
		return struct{}{}, write()
	})
	return err
}

// parseBatchRecords binds each item of a batch that writes existing records
// with the single tool's parser, and rejects a record listed twice, since
// concurrent writes to it race. It returns the requests and their labels.
func parseBatchRecords[T any](
	items []map[string]any, noun string,
	parse func(map[string]any) (T, *mcp.CallToolResult), id func(T) int64,
) ([]T, []string, *mcp.CallToolResult) {
	requests := make([]T, len(items))
	labels := make([]string, len(items))
	seen := make(map[int64]bool, len(items))
	var invalid []string
	for i, item := range items {
		req, toolResult := parse(item)
		if toolResult != nil {
			invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, toolResultText(toolResult)))
			continue
		}
		if seen[id(req)] {
			invalid = append(invalid, fmt.Sprintf("item %d: %s %d is listed more than once; "+
				"merge it into one item", i+1, noun, id(req)))
			continue
		}
		seen[id(req)] = true
		requests[i] = req
		labels[i] = fmt.Sprintf("%s %d", noun, id(req))
	}
	if len(invalid) > 0 {
		return nil, nil, batchInvalidItems(invalid)
	}
	return requests, labels, nil
}

// rejectDroppedParams fails a batch item that sends a parameter the batch
// item schema left out, since nothing would tell the caller it was ignored.
func rejectDroppedParams(item map[string]any, dropped ...string) *mcp.CallToolResult {
	for _, name := range dropped {
		if _, ok := item[name]; ok {
			return helpers.NewToolResultTextError("%s is not accepted here", name)
		}
	}
	return nil
}

// uniqueIDs removes repeated IDs, keeping the caller's order.
func uniqueIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}

// runBatchInOrder is runBatch for items that must be written in the order
// listed within a group, such as records landing in one project: groups run
// concurrently, and the items of one group one after another.
func runBatchInOrder(
	ctx context.Context, groups []int64, concurrency int, do func(ctx context.Context, i int) error,
) []error {
	var keys []int64
	members := make(map[int64][]int)
	for i, group := range groups {
		if _, ok := members[group]; !ok {
			keys = append(keys, group)
		}
		members[group] = append(members[group], i)
	}

	errs := make([]error, len(groups))
	groupErrs := runBatch(ctx, len(keys), concurrency, func(ctx context.Context, g int) error {
		for _, i := range members[keys[g]] {
			if err := ctx.Err(); err != nil {
				errs[i] = fmt.Errorf("%w: %w", errBatchNotStarted, err)
				continue
			}
			err := do(ctx, i)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("%w: %w", errBatchInterrupted, err)
			}
			errs[i] = err
		}
		return nil
	})
	for g, err := range groupErrs {
		if errors.Is(err, errBatchNotStarted) {
			for _, i := range members[keys[g]] {
				errs[i] = err
			}
		}
	}
	return errs
}
