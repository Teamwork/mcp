package twprojects

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/teamwork/mcp/internal/request"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// resolveCustomItemFields fetches the field definitions for a custom item
// type once and reuses them across record reads, record writes, and field
// inspections within a short TTL window. This keeps the common "create
// record" flow at one extra GET per custom item type per minute, not per
// record.
//
// Keying: customItemID is only unique within an installation, so the cache
// is keyed by {installationID, customItemID}. Under STDIO the installation
// ID is 0 (single-tenant process), which is safe — every entry collapses
// under one tenant.
//
// Eviction: TTL alone would let the map grow without bound (entries are
// never removed on expiry, only overwritten). To bound memory, every
// resolveCustomItemFields call that takes the miss path opportunistically
// sweeps expired entries if the previous sweep is older than
// customItemFieldCacheSweepEvery. A CAS on customItemFieldCacheLastSweep
// ensures only one goroutine sweeps per cycle.
//
// Invalidation: any field create/update/delete invalidates the entry for the
// affected (installation, custom item) pair. Type delete also invalidates.
// Other writes do not touch the cache.

const (
	customItemFieldCacheTTL        = 60 * time.Second
	customItemFieldCacheSweepEvery = 5 * time.Minute
)

type customItemFieldCacheKey struct {
	installationID int64
	customItemID   int64
}

type customItemFieldCacheEntry struct {
	fields  []projects.CustomItemField
	expires time.Time
}

var (
	customItemFieldCache          sync.Map // customItemFieldCacheKey -> *customItemFieldCacheEntry
	customItemFieldCacheLastSweep atomic.Int64
)

// customItemFieldCacheKeyFromContext builds the cache key from the request
// context. InstallationID() returns 0 when no request.Info is on the context
// (STDIO), which is the correct single-tenant key.
func customItemFieldCacheKeyFromContext(ctx context.Context, customItemID int64) customItemFieldCacheKey {
	var installationID int64
	if info, ok := request.InfoFromContext(ctx); ok {
		installationID = info.InstallationID()
	}
	return customItemFieldCacheKey{installationID: installationID, customItemID: customItemID}
}

// resolveCustomItemFields returns the fields defined on the given custom
// item type, hitting an in-process cache first. The result must be treated
// as read-only — it is shared across goroutines.
func resolveCustomItemFields(
	ctx context.Context,
	engine *twapi.Engine,
	customItemID int64,
) ([]projects.CustomItemField, error) {
	now := time.Now()
	key := customItemFieldCacheKeyFromContext(ctx, customItemID)

	if v, ok := customItemFieldCache.Load(key); ok {
		entry := v.(*customItemFieldCacheEntry)
		if now.Before(entry.expires) {
			return entry.fields, nil
		}
		// Stale entry — drop it (only if still the same pointer) and fall
		// through to refetch.
		customItemFieldCache.CompareAndDelete(key, v)
	}

	sweepExpiredCustomItemFields(now)

	// Page through the full set so callers get every field regardless of
	// pagination defaults.
	req := projects.NewCustomItemFieldListRequest(customItemID)
	req.Filters.PageSize = 100

	var all []projects.CustomItemField
	for {
		resp, err := projects.CustomItemFieldList(ctx, engine, req)
		if err != nil {
			return nil, fmt.Errorf("failed to load fields for custom item %d: %w", customItemID, err)
		}
		all = append(all, resp.CustomItemFields...)
		next := resp.Iterate()
		if next == nil {
			break
		}
		req = *next
	}

	customItemFieldCache.Store(key, &customItemFieldCacheEntry{
		fields:  all,
		expires: now.Add(customItemFieldCacheTTL),
	})

	return all, nil
}

// sweepExpiredCustomItemFields walks the cache and removes entries whose TTL
// has elapsed. The CAS on customItemFieldCacheLastSweep ensures at most one
// goroutine sweeps per customItemFieldCacheSweepEvery window, so the cost
// stays bounded even under heavy parallelism.
func sweepExpiredCustomItemFields(now time.Time) {
	last := customItemFieldCacheLastSweep.Load()
	if now.UnixNano()-last < int64(customItemFieldCacheSweepEvery) {
		return
	}
	if !customItemFieldCacheLastSweep.CompareAndSwap(last, now.UnixNano()) {
		return
	}
	customItemFieldCache.Range(func(k, v any) bool {
		if entry, ok := v.(*customItemFieldCacheEntry); ok && !now.Before(entry.expires) {
			customItemFieldCache.CompareAndDelete(k, v)
		}
		return true
	})
}

// invalidateCustomItemFieldCache drops the cached field list for a custom
// item type so the next resolveCustomItemFields call refetches. Called on
// every field write and on type delete.
func invalidateCustomItemFieldCache(ctx context.Context, customItemID int64) {
	customItemFieldCache.Delete(customItemFieldCacheKeyFromContext(ctx, customItemID))
}
