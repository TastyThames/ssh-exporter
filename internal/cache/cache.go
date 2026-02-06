package cache

import (
	"sync"
	"time"
)

type Result struct {
	At     time.Time
	Labels map[string]string
	Values map[string]float64
	Err    error
}

// Cache is the interface used by scheduler/metrics.
type Cache interface {
	Set(target string, r Result)
	Snapshot() map[string]Result
}

// MemCache is an in-memory implementation of Cache.
type MemCache struct {
	mu   sync.RWMutex
	data map[string]Result
}

func NewMemCache() *MemCache {
	return &MemCache{
		data: make(map[string]Result),
	}
}

func (c *MemCache) Set(target string, r Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[target] = r
}

func (c *MemCache) Snapshot() map[string]Result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]Result, len(c.data))
	for k, v := range c.data {
		out[k] = v
	}
	return out
}
