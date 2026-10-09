package peerpolicy

import "time"

type tokenBucket struct {
	tokens, rate, capacity float64
	last                   time.Duration
}

func newBucket(rate, capacity uint64, now time.Duration) tokenBucket {
	return tokenBucket{float64(capacity), float64(rate), float64(capacity), now}
}

func (b *tokenBucket) available(now time.Duration) float64 {
	if now > b.last {
		b.tokens = min(b.capacity, b.tokens+(now-b.last).Seconds()*b.rate)
		b.last = now
	}
	return b.tokens
}
func (b *tokenBucket) spend(n uint64)  { b.tokens -= float64(n) }
func (b *tokenBucket) refund(n uint64) { b.tokens = min(b.capacity, b.tokens+float64(n)) }
