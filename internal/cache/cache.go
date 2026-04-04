package cache

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache"},
	)
	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache"},
	)
)

type item struct {
	value     interface{}
	expiresAt time.Time
}

// Cache is a TTL-based in-memory cache with automatic cleanup.
// Per ADR-016 Caching Strategy, Level 2 (application-level).
type Cache struct {
	name     string
	mu       sync.RWMutex
	items    map[string]item
	ttl      time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

// New creates a cache with the given default TTL and starts a background
// cleanup goroutine that runs at the specified interval.
func New(name string, ttl, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		name:  name,
		items: make(map[string]item),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}

	go c.cleanup(cleanupInterval)
	return c
}

// Get retrieves a value by key. Returns the value and true if found and not expired.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(it.expiresAt) {
		cacheMisses.WithLabelValues(c.name).Inc()
		if ok {
			// Expired — remove lazily
			c.mu.Lock()
			delete(c.items, key)
			c.mu.Unlock()
		}
		return nil, false
	}

	cacheHits.WithLabelValues(c.name).Inc()
	return it.value, true
}

// Set stores a value with the cache's default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = item{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Len returns the number of items (including expired) in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stop halts the background cleanup goroutine. Safe to call multiple times.
func (c *Cache) Stop() {
	c.stopOnce.Do(func() { close(c.stop) })
}

func (c *Cache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, it := range c.items {
				if now.After(it.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}
