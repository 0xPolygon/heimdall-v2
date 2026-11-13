package peerpolicy

import (
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
)

const peersPerPool = 3

type pool struct {
	bytes, stateBytes, work                  tokenBucket
	familyWork                               [2]tokenBucket
	memory, inflight, memoryCap, inflightCap uint64
	stateMemory, stateInflight               uint64
}

func newPool(cfg Config, divisor uint64, catchup bool, now time.Duration) pool {
	rate, burst := cfg.BytesPerSecond/divisor, cfg.BurstBytes/divisor
	work := cfg.RequestsPerSecond / divisor
	workBurst := max(work, min(cfg.MaxInflight/divisor, peersPerPool*maxPeerInflight))
	p := pool{work: newBucket(work, workBurst, now), memoryCap: cfg.MemoryBytes / divisor, inflightCap: cfg.MaxInflight / divisor}
	familyWork := work
	if !catchup {
		rate, burst, familyWork = rate/2, burst/2, work/2
		p.stateBytes = newBucket(rate, burst, now)
	}
	p.bytes = newBucket(rate, burst, now)
	// Families share the work burst but cannot consume each other's refill.
	for i := range p.familyWork {
		p.familyWork[i] = newBucket(familyWork, workBurst, now)
	}
	return p
}

func (p *pool) allowance(family servebudget.Family) (*tokenBucket, *tokenBucket) {
	if stateFamily(family) {
		return &p.stateBytes, &p.familyWork[1]
	}
	return &p.bytes, &p.familyWork[0]
}

func (p *pool) hasCapacity(memory uint64, family servebudget.Family, now time.Duration) bool {
	if stateFamily(family) && (p.stateMemory+memory > p.memoryCap/2 || p.stateInflight >= p.inflightCap/2) {
		return false
	}
	_, work := p.allowance(family)
	return p.memory+memory <= p.memoryCap && p.inflight < p.inflightCap && p.work.available(now) >= 1 && work.available(now) >= 1
}

func stateFamily(family servebudget.Family) bool {
	return family == servebudget.Snapshot || family == servebudget.Chunk
}
