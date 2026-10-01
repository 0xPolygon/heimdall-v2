package keeper

import (
	"context"
	"sync"
)

// spanEndFrontier caches the greatest observed span EndBlock. A CacheContext write
// that is later discarded can leave this value conservatively high, which only
// disables the fast rejection for some misses; SpanByBlockNumber still performs
// the authoritative state lookup and therefore cannot return a phantom span.
type spanEndFrontier struct {
	mu    sync.Mutex
	max   uint64
	ready bool
}

// WarmSpanEndFrontier initializes the derived frontier when span state is
// already available. An empty store remains uninitialized so state sync can
// restore spans before the lazy lookup derives the frontier.
func (k *Keeper) WarmSpanEndFrontier(ctx context.Context) error {
	hasLastSpan, err := k.latestSpan.Has(ctx)
	if err != nil {
		return err
	}
	if !hasLastSpan {
		return nil
	}

	_, err = k.maxSpanEndBlock(ctx)
	return err
}

func (k *Keeper) maxSpanEndBlock(ctx context.Context) (uint64, error) {
	k.spanFrontier.mu.Lock()
	defer k.spanFrontier.mu.Unlock()

	if k.spanFrontier.ready {
		return k.spanFrontier.max, nil
	}

	maxEnd, err := k.scanMaxSpanEndBlock(ctx)
	if err != nil {
		return 0, err
	}

	k.spanFrontier.max = maxEnd
	k.spanFrontier.ready = true
	return maxEnd, nil
}

func (k *Keeper) scanMaxSpanEndBlock(ctx context.Context) (maxEnd uint64, err error) {
	spans, err := k.GetAllSpans(ctx)
	if err != nil {
		return 0, err
	}

	for _, span := range spans {
		maxEnd = max(maxEnd, span.EndBlock)
	}

	return maxEnd, nil
}

func (k *Keeper) setSpanEndFrontier(maxEnd uint64) {
	k.spanFrontier.mu.Lock()
	defer k.spanFrontier.mu.Unlock()

	k.spanFrontier.max = max(k.spanFrontier.max, maxEnd)
	k.spanFrontier.ready = true
}

func (k *Keeper) advanceSpanEndFrontier(endBlock uint64) {
	k.spanFrontier.mu.Lock()
	defer k.spanFrontier.mu.Unlock()

	if k.spanFrontier.ready {
		k.spanFrontier.max = max(k.spanFrontier.max, endBlock)
	}
}
