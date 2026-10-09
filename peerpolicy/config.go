// Package peerpolicy owns native serving admission and observational reputation.
package peerpolicy

import (
	"encoding/hex"
	"fmt"
	"strings"
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
		Observe: true, BytesPerSecond: 32 << 20, BurstBytes: 512 << 20, MemoryBytes: 1 << 30, MaxInflight: 80, RequestsPerSecond: 128,
		PeerBytesPerSecond: 8 << 20, PeerBurstBytes: 128 << 20, PeerRequestsPerSecond: 32, RepeatBytes: 32 << 20,
	}
}

func (c Config) Validate() error {
	for _, v := range []uint64{c.BytesPerSecond, c.BurstBytes, c.MemoryBytes, c.MaxInflight, c.RequestsPerSecond, c.PeerBytesPerSecond, c.PeerBurstBytes, c.PeerRequestsPerSecond, c.RepeatBytes} {
		if v == 0 || v > 1<<40 {
			return fmt.Errorf("serving limits must be positive and at most 1 TiB")
		}
	}
	if c.MaxInflight < 4 || c.MaxInflight > 4096 {
		return fmt.Errorf("serving concurrency must be between 4 and 4096")
	}
	if c.BytesPerSecond < 4 || c.BurstBytes < 4 || c.MemoryBytes < 4 || c.RequestsPerSecond < 4 {
		return fmt.Errorf("node-wide limits must be at least four to fund each reserved pool")
	}
	return validateProtectedIDs(c.ProtectedIDs)
}

func validateProtectedIDs(ids []string) error {
	if len(ids) > 256 {
		return fmt.Errorf("at most 256 protected peer IDs are allowed")
	}
	for _, id := range ids {
		if _, err := hex.DecodeString(strings.ToLower(id)); err != nil || len(id) != 40 {
			return fmt.Errorf("protected peer ID must contain 40 hexadecimal characters")
		}
	}
	return nil
}
