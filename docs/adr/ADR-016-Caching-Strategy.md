# ADR-016: Caching Strategy

## Status

Accepted

## Context

Caching is critical for achieving sub-100ms response times defined in ADR-000. Without effective caching, database queries and computations on every request will exceed performance budgets. This ADR establishes caching patterns at multiple levels: HTTP caching, in-memory caching, and precomputation.

The goal is to minimize database queries and expensive computations while maintaining data freshness.

## Decision

### 1. Multi-Level Caching Strategy

#### Level 1: HTTP Caching (Browser/CDN)
- **Static assets**: Aggressive caching (1 year)
- **Dynamic content**: Short TTL with validation
- **Private data**: No-cache or private cache-control

#### Level 2: In-Memory Caching (Application)
- **Frequently accessed data**: Cache in Go application memory
- **Computed results**: Cache expensive calculations
- **Session data**: Store in memory for fast access

#### Level 3: Build-Time Precomputation
- **Static content**: Generate at build time, not runtime
- **Navigation structures**: Compute once at startup
- **Aggregations**: Precompute and store

### 2. HTTP Caching Headers

```go
// Static asset caching (immutable assets with hashed filenames)
func serveStaticAssets(w http.ResponseWriter, r *http.Request) {
    // Aggressive caching for versioned/hashed assets
    w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
    w.Header().Set("Vary", "Accept-Encoding")
    
    // Serve pre-compressed files if available
    serveCompressedAsset(w, r)
}

// Dynamic page caching (short TTL with revalidation)
func serveDynamicPage(w http.ResponseWriter, r *http.Request) {
    // Cache for 5 minutes, revalidate after
    w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
    w.Header().Set("ETag", generateETag(content))
    
    // Check If-None-Match for 304 responses
    if r.Header.Get("If-None-Match") == etag {
        w.WriteHeader(http.StatusNotModified)
        return
    }
    
    renderPage(w, content)
}

// Private/authenticated content (no caching)
func serveAuthenticatedContent(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")
    
    renderPrivateContent(w, r)
}
```

### 3. In-Memory Caching

#### Simple In-Memory Cache

```go
package cache

import (
    "sync"
    "time"
)

type CacheItem struct {
    Value      interface{}
    Expiration time.Time
}

type Cache struct {
    items map[string]CacheItem
    mu    sync.RWMutex
    ttl   time.Duration
}

func NewCache(ttl time.Duration) *Cache {
    c := &Cache{
        items: make(map[string]CacheItem),
        ttl:   ttl,
    }
    
    // Start cleanup goroutine
    go c.cleanup()
    
    return c
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.items[key] = CacheItem{
        Value:      value,
        Expiration: time.Now().Add(c.ttl),
    }
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    item, exists := c.items[key]
    if !exists {
        return nil, false
    }
    
    // Check expiration
    if time.Now().After(item.Expiration) {
        return nil, false
    }
    
    return item.Value, true
}

func (c *Cache) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.items, key)
}

func (c *Cache) cleanup() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, item := range c.items {
            if now.After(item.Expiration) {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}
```

#### Usage Example

```go
var (
    postCache = cache.NewCache(5 * time.Minute)
)

func GetPost(slug string) (*Post, error) {
    // Try cache first
    if cached, ok := postCache.Get(slug); ok {
        return cached.(*Post), nil
    }
    
    // Cache miss: query database
    post, err := db.QueryPost(slug)
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    postCache.Set(slug, post)
    
    return post, nil
}

// Invalidate cache on update
func UpdatePost(slug string, post *Post) error {
    if err := db.UpdatePost(slug, post); err != nil {
        return err
    }
    
    // Invalidate cache
    postCache.Delete(slug)
    
    return nil
}
```

### 4. Build-Time Precomputation

#### Precompute Navigation at Startup

```go
// Precompute expensive operations at startup
var (
    postNavigation map[string]Navigation
    postIndex      []Post
)

func init() {
    log.Info().Msg("precomputing post navigation")
    
    // Fetch all posts once
    posts, err := fetchAllPosts()
    if err != nil {
        log.Fatal().Err(err).Msg("failed to fetch posts")
    }
    
    // Sort by publish date
    sort.Slice(posts, func(i, j int) bool {
        return posts[i].PublishedAt.After(posts[j].PublishedAt)
    })
    
    // Build navigation map (O(n) instead of O(n²))
    postNavigation = make(map[string]Navigation)
    for i, post := range posts {
        nav := Navigation{}
        if i > 0 {
            nav.Prev = &posts[i-1]
        }
        if i < len(posts)-1 {
            nav.Next = &posts[i+1]
        }
        postNavigation[post.Slug] = nav
    }
    
    postIndex = posts
    
    log.Info().Int("count", len(posts)).Msg("post navigation precomputed")
}

// Handler uses precomputed data (O(1) lookup)
func HandleBlogPost(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "slug")
    
    post, err := getPost(slug)
    if err != nil {
        http.Redirect(w, r, "/404", http.StatusSeeOther)
        return
    }
    
    // O(1) navigation lookup
    nav := postNavigation[slug]
    
    render(w, post, nav)
}
```

#### Precompute Aggregations

```go
// Cache expensive aggregations
var (
    statsCache     *Stats
    statsCacheMu   sync.RWMutex
    statsLastUpdate time.Time
)

func GetStats() (*Stats, error) {
    statsCacheMu.RLock()
    // Return cached stats if fresh (< 1 hour old)
    if time.Since(statsLastUpdate) < time.Hour {
        defer statsCacheMu.RUnlock()
        return statsCache, nil
    }
    statsCacheMu.RUnlock()
    
    // Recompute stats
    statsCacheMu.Lock()
    defer statsCacheMu.Unlock()
    
    // Double-check after acquiring write lock
    if time.Since(statsLastUpdate) < time.Hour {
        return statsCache, nil
    }
    
    stats, err := computeStats()
    if err != nil {
        return nil, err
    }
    
    statsCache = stats
    statsLastUpdate = time.Now()
    
    return stats, nil
}
```

### 5. Cache Invalidation Strategy

#### Time-Based Expiration (TTL)
- **Short TTL**: 5 minutes for frequently changing data
- **Medium TTL**: 1 hour for moderately changing data
- **Long TTL**: 24 hours for rarely changing data

#### Event-Based Invalidation
```go
// Invalidate cache on data mutation
func CreatePost(post *Post) error {
    if err := db.InsertPost(post); err != nil {
        return err
    }
    
    // Invalidate affected caches
    postCache.Delete(post.Slug)
    postListCache.Delete("all")
    
    return nil
}

// Invalidate related caches
func UpdatePost(slug string, post *Post) error {
    if err := db.UpdatePost(slug, post); err != nil {
        return err
    }
    
    // Invalidate post cache
    postCache.Delete(slug)
    
    // Invalidate list caches that might contain this post
    postListCache.Delete("all")
    postListCache.Delete("recent")
    
    return nil
}
```

#### Cache Stampede Prevention
```go
import "golang.org/x/sync/singleflight"

var (
    requestGroup singleflight.Group
)

// Prevent cache stampede using singleflight
func GetPostWithDedup(slug string) (*Post, error) {
    // Multiple concurrent requests for same slug will be deduplicated
    v, err, _ := requestGroup.Do(slug, func() (interface{}, error) {
        // Check cache
        if cached, ok := postCache.Get(slug); ok {
            return cached, nil
        }
        
        // Cache miss: query database once
        post, err := db.QueryPost(slug)
        if err != nil {
            return nil, err
        }
        
        // Cache result
        postCache.Set(slug, post)
        
        return post, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return v.(*Post), nil
}
```

### 6. CDN Caching (Cloudflare)

#### Cloudflare Cache Configuration

```go
// Set Cloudflare-specific headers
func cloudflareOptimizedHandler(w http.ResponseWriter, r *http.Request) {
    // Cache on Cloudflare edge for 1 hour
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.Header().Set("CDN-Cache-Control", "max-age=3600")
    
    // Tell Cloudflare what to vary on
    w.Header().Set("Vary", "Accept-Encoding")
    
    // Cloudflare-specific: cache even with cookies
    w.Header().Set("Cache-Tag", "blog-posts")
    
    renderContent(w, r)
}

// Purge Cloudflare cache on content update
func purgeCloudflareCache(tags []string) error {
    // Use Cloudflare API to purge cache by tag
    // Implementation depends on Cloudflare SDK
    return nil
}
```

### 7. Cache Monitoring

```go
// Cache metrics
var (
    cacheHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Total number of cache hits",
        },
        []string{"cache"},
    )
    
    cacheMisses = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_misses_total",
            Help: "Total number of cache misses",
        },
        []string{"cache"},
    )
)

// Track cache performance
func GetWithMetrics(key string) (interface{}, bool) {
    value, ok := cache.Get(key)
    
    if ok {
        cacheHits.WithLabelValues("post").Inc()
    } else {
        cacheMisses.WithLabelValues("post").Inc()
    }
    
    return value, ok
}
```

## Consequences

### Positive

- **Performance**: Achieves sub-100ms response times via reduced database load
- **Scalability**: Handles more requests with same infrastructure
- **Cost Efficiency**: Reduced database queries = lower infrastructure costs
- **User Experience**: Faster page loads improve satisfaction
- **CDN Benefits**: Global edge caching improves latency for distributed users

### Negative

- **Complexity**: Caching adds complexity to code
- **Stale Data**: Risk of serving outdated content
- **Memory Usage**: In-memory cache consumes RAM
- **Invalidation Difficulty**: Cache invalidation is hard to get right

### Risks

- **Cache Stampede**: Many requests hitting cold cache simultaneously
- **Memory Exhaustion**: Unbounded cache can exhaust memory
- **Stale Content**: Users may see outdated information
- **Debugging**: Cached responses harder to debug

## Alternatives Considered

### 1. Redis for Distributed Caching
- **Deferred**: Start with in-memory cache, add Redis if needed for multi-instance
- **Note**: Redis adds operational complexity and latency

### 2. No Caching
- **Rejected**: Cannot meet performance budgets without caching

### 3. Database Query Caching Only
- **Rejected**: Still requires database connection, not fast enough

## Implementation Checklist

- [ ] Implement simple in-memory cache with TTL
- [ ] Add HTTP caching headers for static assets
- [ ] Add ETag support for dynamic content
- [ ] Implement cache invalidation on data mutations
- [ ] Add cache stampede prevention (singleflight)
- [ ] Precompute navigation structures at startup
- [ ] Configure Cloudflare cache rules
- [ ] Add cache hit/miss metrics
- [ ] Document cache invalidation patterns
- [ ] Test cache behavior under load

## Performance Impact

### Before Caching (Baseline)
- Average response time: 150ms (over budget)
- Database queries per request: 5-10
- RPS capacity: 500 requests/second

### After Caching (Target)
- Average response time: 30ms (well under budget)
- Database queries per request: 0-1 (cache hit)
- RPS capacity: 5000+ requests/second

### Cache Hit Rate Targets
- **80%+ cache hit rate** for public content
- **90%+ cache hit rate** for static assets
- **50%+ cache hit rate** for authenticated content

## References

- [ADR-000: Performance Budgets](./ADR-000-Performance-Budgets-and-Quality-Attributes.md)
- [HTTP Caching - MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching)
- [Cloudflare Cache](https://developers.cloudflare.com/cache/)
- [Go sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [Cache Stampede Prevention](https://en.wikipedia.org/wiki/Cache_stampede)

## Review Cadence

**Review Date**: 2026-05-15

---

**Date**: 2025-11-15
**Author**: System Architecture Team
