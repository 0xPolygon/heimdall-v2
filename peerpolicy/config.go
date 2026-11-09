// Package peerpolicy owns native serving admission and observational reputation.
package peerpolicy

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	Observe               bool
	BytesPerSecond        uint64
	BurstBytes            uint64
	MemoryBytes           uint64
	MaxInflight           uint64
	RequestsPerSecond     uint64
	PeerBytesPerSecond    uint64
	PeerBurstBytes        uint64
	PeerRequestsPerSecond uint64
	RepeatBytes           uint64
	ProtectedIDs          []string
}

func DefaultConfig() Config {
	return Config{
		Observe: true, BytesPerSecond: 32 << 20, BurstBytes: 3 << 30, MemoryBytes: 3 << 30, MaxInflight: 12 * maxPeerInflight, RequestsPerSecond: 128,
		PeerBytesPerSecond: 8 << 20, PeerBurstBytes: 128 << 20, PeerRequestsPerSecond: 32, RepeatBytes: 32 << 20,
	}
}

func (c Config) Validate() error {
	for _, v := range []uint64{c.BytesPerSecond, c.BurstBytes, c.MemoryBytes, c.MaxInflight, c.RequestsPerSecond, c.PeerBytesPerSecond, c.PeerBurstBytes, c.PeerRequestsPerSecond, c.RepeatBytes} {
		if v == 0 || v > 1<<40 {
			return fmt.Errorf("serving limits must be positive and at most 1 TiB")
		}
	}
	if c.MaxInflight < 24 || c.MaxInflight > 4096 {
		return fmt.Errorf("serving concurrency must be between 24 and 4096")
	}
	if c.BytesPerSecond < 8 || c.BurstBytes < 24 || c.MemoryBytes < 48 || c.RequestsPerSecond < 8 {
		return fmt.Errorf("node-wide byte and work rates must be at least eight, burst at least 24 and memory at least 48 to fund reserved shares")
	}
	objects, history := c.historyLimits()
	if objects > maxPeerHistory || history > maxNodeHistory {
		return fmt.Errorf("request rates exceed bounded serving history capacity")
	}
	return validateProtectedIDs(c.ProtectedIDs)
}

func (c Config) historyLimits() (uint64, uint64) {
	// Retain both peer allowance classes for the horizon plus their initial
	// burst and older pending responses, without allocating the tables upfront.
	seconds := uint64(windowCount*windowWidth/time.Second) + 1
	return max(defaultPeerHistory, 2*c.PeerRequestsPerSecond*seconds+2*maxPeerInflight),
		max(defaultNodeHistory, c.RequestsPerSecond*seconds+c.MaxInflight)
}

func validateProtectedIDs(ids []string) error {
	for _, id := range ids {
		if _, err := hex.DecodeString(strings.ToLower(id)); err != nil || len(id) != 40 {
			return fmt.Errorf("protected peer ID must contain 40 hexadecimal characters")
		}
	}
	return nil
}
