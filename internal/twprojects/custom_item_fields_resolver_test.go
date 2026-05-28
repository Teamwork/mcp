package twprojects

import (
	"context"
	"testing"
	"time"

	"github.com/teamwork/mcp/internal/request"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// resetCustomItemFieldCache clears the package-level cache between subtests
// so they don't leak state into each other.
func resetCustomItemFieldCache(t *testing.T) {
	t.Helper()
	customItemFieldCache.Range(func(k, _ any) bool {
		customItemFieldCache.Delete(k)
		return true
	})
	customItemFieldCacheLastSweep.Store(0)
}

// ctxWithInstallation returns a context carrying a request.Info with the
// given installation ID, mirroring what the HTTP auth middleware injects.
func ctxWithInstallation(installationID int64) context.Context {
	info := &request.Info{}
	info.SetAuth(installationID, "https://example.com", 1)
	return request.WithInfo(context.Background(), info)
}

// TestCustomItemFieldCacheKeyScoping is the regression guard for cross-tenant
// collision: two installations with the same customItemID must not see each
// other's cached fields.
func TestCustomItemFieldCacheKeyScoping(t *testing.T) {
	resetCustomItemFieldCache(t)

	const customItemID int64 = 42

	a := &customItemFieldCacheEntry{
		fields:  []projects.CustomItemField{{ID: 100}},
		expires: time.Now().Add(time.Minute),
	}
	b := &customItemFieldCacheEntry{
		fields:  []projects.CustomItemField{{ID: 200}},
		expires: time.Now().Add(time.Minute),
	}
	customItemFieldCache.Store(customItemFieldCacheKeyFromContext(ctxWithInstallation(1), customItemID), a)
	customItemFieldCache.Store(customItemFieldCacheKeyFromContext(ctxWithInstallation(2), customItemID), b)

	gotA, ok := customItemFieldCache.Load(customItemFieldCacheKeyFromContext(ctxWithInstallation(1), customItemID))
	if !ok || gotA.(*customItemFieldCacheEntry).fields[0].ID != 100 {
		t.Fatalf("installation 1 lookup leaked across tenant: got %+v ok=%v", gotA, ok)
	}
	gotB, ok := customItemFieldCache.Load(customItemFieldCacheKeyFromContext(ctxWithInstallation(2), customItemID))
	if !ok || gotB.(*customItemFieldCacheEntry).fields[0].ID != 200 {
		t.Fatalf("installation 2 lookup leaked across tenant: got %+v ok=%v", gotB, ok)
	}

	// Invalidating one tenant must not touch the other.
	invalidateCustomItemFieldCache(ctxWithInstallation(1), customItemID)
	if _, ok := customItemFieldCache.Load(customItemFieldCacheKeyFromContext(ctxWithInstallation(1), customItemID)); ok {
		t.Fatal("installation 1 entry should have been invalidated")
	}
	if _, ok := customItemFieldCache.Load(customItemFieldCacheKeyFromContext(ctxWithInstallation(2), customItemID)); !ok {
		t.Fatal("installation 2 entry should still be present")
	}
}

// TestCustomItemFieldCacheKeyMissingInfo confirms that a context without
// request.Info (the STDIO path) collapses to installationID=0, which is the
// correct single-tenant key.
func TestCustomItemFieldCacheKeyMissingInfo(t *testing.T) {
	got := customItemFieldCacheKeyFromContext(context.Background(), 7)
	want := customItemFieldCacheKey{installationID: 0, customItemID: 7}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

// TestSweepExpiredCustomItemFields confirms that expired entries are evicted
// by the opportunistic sweep so the map can't grow without bound.
func TestSweepExpiredCustomItemFields(t *testing.T) {
	resetCustomItemFieldCache(t)

	now := time.Now()
	freshKey := customItemFieldCacheKey{installationID: 1, customItemID: 1}
	staleKey := customItemFieldCacheKey{installationID: 1, customItemID: 2}
	customItemFieldCache.Store(freshKey, &customItemFieldCacheEntry{expires: now.Add(time.Minute)})
	customItemFieldCache.Store(staleKey, &customItemFieldCacheEntry{expires: now.Add(-time.Minute)})

	// Force the sweep to run by pretending the last sweep was long ago.
	customItemFieldCacheLastSweep.Store(0)
	sweepExpiredCustomItemFields(now)

	if _, ok := customItemFieldCache.Load(staleKey); ok {
		t.Fatal("stale entry should have been swept")
	}
	if _, ok := customItemFieldCache.Load(freshKey); !ok {
		t.Fatal("fresh entry should have survived the sweep")
	}
}

// TestSweepExpiredCustomItemFieldsRateLimited confirms the CAS gate prevents
// back-to-back sweeps from doing work, which is what bounds the per-call
// cost on the miss path.
func TestSweepExpiredCustomItemFieldsRateLimited(t *testing.T) {
	resetCustomItemFieldCache(t)

	now := time.Now()
	staleKey := customItemFieldCacheKey{installationID: 1, customItemID: 1}
	customItemFieldCache.Store(staleKey, &customItemFieldCacheEntry{expires: now.Add(-time.Minute)})

	// First sweep claims the window and evicts.
	customItemFieldCacheLastSweep.Store(0)
	sweepExpiredCustomItemFields(now)
	if _, ok := customItemFieldCache.Load(staleKey); ok {
		t.Fatal("first sweep should have evicted the stale entry")
	}

	// Re-insert and call again immediately — the rate limiter should skip.
	customItemFieldCache.Store(staleKey, &customItemFieldCacheEntry{expires: now.Add(-time.Minute)})
	sweepExpiredCustomItemFields(now)
	if _, ok := customItemFieldCache.Load(staleKey); !ok {
		t.Fatal("second sweep within the rate-limit window should have been skipped")
	}
}
