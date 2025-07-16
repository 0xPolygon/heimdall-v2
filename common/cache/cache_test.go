package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	c := NewCache[string](1 * time.Second)
	c.Set("foo", "bar")

	val, err := c.Get("foo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != "bar" {
		t.Fatalf("expected 'bar', got %v", val)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := NewCache[string](50 * time.Millisecond)
	c.Set("temp", "value")

	if _, err := c.Get("temp"); err != nil {
		t.Fatalf("expected value before expiration, got error: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	_, err := c.Get("temp")
	if err == nil {
		t.Fatal("expected error after expiration, got none")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := NewCache[int](1 * time.Second)
	c.Set("num", 42)
	c.Set("num", 99)

	val, err := c.Get("num")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != 99 {
		t.Fatalf("expected 99, got %v", val)
	}
}

func TestCacheExpiredItemIsDeleted(t *testing.T) {
	c := NewCache[string](10 * time.Millisecond)
	c.Set("expired", "gone")

	time.Sleep(20 * time.Millisecond)
	_, _ = c.Get("expired")

	c.mu.RLock()
	_, exists := c.items["expired"]
	c.mu.RUnlock()

	if exists {
		t.Fatal("expected expired item to be deleted from cache")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := NewCache[int](100 * time.Millisecond)
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			c.Set("key", i)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	for j := 0; j < 5; j++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				_, _ = c.Get("key")
				time.Sleep(1 * time.Millisecond)
			}
		}(j)
	}

	wg.Wait()
}
