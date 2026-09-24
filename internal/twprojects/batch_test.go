package twprojects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBatchBoundsConcurrency(t *testing.T) {
	var running, peak atomic.Int32
	errs := runBatch(t.Context(), 20, 3, func(context.Context, int) error {
		current := running.Add(1)
		for {
			seen := peak.Load()
			if current <= seen || peak.CompareAndSwap(seen, current) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		return nil
	})
	if got := peak.Load(); got > 3 {
		t.Errorf("expected at most 3 concurrent items, got %d", got)
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("item %d: unexpected error %v", i, err)
		}
	}
}

func TestRunBatchKeepsErrorsInInputOrder(t *testing.T) {
	failure := errors.New("boom")
	errs := runBatch(t.Context(), 5, 2, func(_ context.Context, i int) error {
		if i%2 == 1 {
			return failure
		}
		return nil
	})
	for i, err := range errs {
		if want := i%2 == 1; (err != nil) != want {
			t.Errorf("item %d: expected failure=%v, got %v", i, want, err)
		}
	}
}

// TestRunBatchStopsStartingItemsWhenCancelled reports the items never started
// as failed rather than silently skipping them.
func TestRunBatchStopsStartingItemsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	errs := runBatch(ctx, 5, 1, func(_ context.Context, i int) error {
		if i == 1 {
			cancel()
		}
		return nil
	})
	for i := 2; i < len(errs); i++ {
		if !errors.Is(errs[i], errBatchNotStarted) || !errors.Is(errs[i], context.Canceled) {
			t.Errorf("item %d: expected a cancelled not-started error, got %v", i, errs[i])
		}
	}
}

// TestRunBatchMarksInFlightItemsInterrupted keeps an item cut off mid-request
// apart from one never sent: the server may already have applied it.
func TestRunBatchMarksInFlightItemsInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	go func() {
		<-started
		cancel()
	}()
	errs := runBatch(ctx, 3, 1, func(ctx context.Context, i int) error {
		if i > 0 {
			t.Errorf("item %d: expected no item to start after the cancellation", i)
		}
		close(started)
		<-ctx.Done()
		// The SDK wraps the transport error, which wraps the context's.
		return fmt.Errorf("failed to execute request: %w", ctx.Err())
	})
	if !errors.Is(errs[0], errBatchInterrupted) {
		t.Errorf("item 0: expected errBatchInterrupted, got %v", errs[0])
	}
	for i := 1; i < len(errs); i++ {
		if !errors.Is(errs[i], errBatchNotStarted) {
			t.Errorf("item %d: expected errBatchNotStarted, got %v", i, errs[i])
		}
	}
}

// TestRunBatchKeepsAPIErrorsAsFailures stops an ordinary failure being read as
// a cancellation.
func TestRunBatchKeepsAPIErrorsAsFailures(t *testing.T) {
	errs := runBatch(t.Context(), 1, 1, func(context.Context, int) error {
		return errors.New("403 forbidden")
	})
	if errors.Is(errs[0], errBatchInterrupted) || errors.Is(errs[0], errBatchNotStarted) {
		t.Errorf("expected a plain failure, got %v", errs[0])
	}
}

func TestBatchReportSeparatesCancelledItems(t *testing.T) {
	result := batchReport("Created", "tasks", "check the list", []string{"a", "b", "c", "d"}, nil, []error{
		nil,
		errors.New("403 forbidden"),
		fmt.Errorf("%w: %w", errBatchInterrupted, context.Canceled),
		fmt.Errorf("%w: %w", errBatchNotStarted, context.Canceled),
	})
	if !result.IsError {
		t.Error("expected an error result")
	}
	text := toolResultText(result)
	for _, want := range []string{
		"Created 1 of 4 tasks.",
		"Failed: b: 403 forbidden.",
		"Interrupted, outcome unknown (check the list): c.",
		"Not attempted, safe to send again: d.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Failed: c") || strings.Contains(text, "Failed: d") {
		t.Errorf("expected cancelled items not to be listed as failed:\n%s", text)
	}
}
