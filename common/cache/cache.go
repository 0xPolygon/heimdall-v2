package cache

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type CacheItem[T any] struct {
	Value      T
	Expiration int64
}

type Cache[T any] struct {
	items map[string]CacheItem[T]
	mu    sync.RWMutex
	ttl   time.Duration
}

func NewCache[T any](defaultTTL time.Duration) *Cache[T] {
	return &Cache[T]{
		items: make(map[string]CacheItem[T]),
		ttl:   defaultTTL,
	}
}

func (c *Cache[T]) Set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheItem[T]{
		Value:      value,
		Expiration: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *Cache[T]) Get(key string) (T, error) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	var zero T
	if !found || time.Now().UnixNano() > item.Expiration {
		if found {
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return zero, errors.New("item not found or expired")
	}

	logToFile(key, item.Value)

	return item.Value, nil
}

func logToFile(key string, value any) {
	// Open a file "/home/ubuntu/cache.log" in append mode
	file, err := os.OpenFile("/home/ubuntu/cache.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("Failed to open cache log file", "error", err)
		return
	}
	defer file.Close()
	// Write the key and value to the file
	_, err = fmt.Fprintf(file, "Cache hit: key=%s, value=%v\n", key, value)
	if err != nil {
		log.Error("Failed to write to cache log file", "error", err)
		return
	}
	log.Info("Cache hit logged to file", "key", key, "value", value)
}
