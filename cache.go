// Package cache provides a fast, in-memory, generic key-value cache for Go
// backend services. It shards its keyspace across independent locks for
// concurrent throughput, supports per-entry TTLs, and evicts via a
// pluggable eviction.Policy (LRU, LFU, FIFO, or a custom implementation)
// once a capacity is set.
package cache

import (
	"sync"
	"time"

	"github.com/vikash-paf/go-cache/eviction"
)

// Cache is the public interface implemented by *Cache. It exists so
// consumers can depend on an interface (and swap in a mock, or an
// alternative implementation) instead of a concrete type.
type Cache[K comparable, V any] interface {
	// Get returns the value stored under key and true, or the zero value
	// and false if the key is absent or has expired.
	Get(key K) (V, bool)
	// Set stores value under key, applying the cache's default TTL unless
	// a SetOption overrides it. Set never fails: on a full, capacity-bounded
	// cache it evicts via the configured Policy to make room.
	Set(key K, value V, opts ...SetOption)
	// Delete removes key and reports whether it was present.
	Delete(key K) bool
	// Has reports whether key is present and unexpired, without affecting
	// eviction order the way Get does.
	Has(key K) bool
	// Len returns the number of entries currently stored, including any
	// not-yet-swept expired entries.
	Len() int
	// Keys returns a snapshot of all live (unexpired at call time) keys.
	// It allocates and copies, so avoid it on hot paths.
	Keys() []K
	// Clear removes every entry from the cache.
	Clear()
	// Close stops the cache's background janitor goroutine. Safe to call
	// once; the cache remains usable afterwards, it just stops sweeping
	// expired entries proactively (Get still treats them as absent).
	Close() error
	// Stats returns a snapshot of hit/miss/eviction counters accumulated
	// since the cache was created.
	Stats() Stats
}

// cacheImpl is the concrete, sharded implementation of Cache.
type cacheImpl[K comparable, V any] struct {
	shards []*shard[K, V]
	hasher keyHasher[K]
	cfg    config[K, V]
	stats  statsCounters

	closeOnce sync.Once
	stopCh    chan struct{}
}

// New builds a Cache configured by the given options. With no options it
// returns an unbounded cache with 32 shards and no TTL: entries live until
// explicitly deleted.
func New[K comparable, V any](opts ...Option[K, V]) Cache[K, V] {
	cfg := defaultConfig[K, V]()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.capacity > 0 && cfg.policy == nil {
		cfg.policy = eviction.NewLRU[K]()
	}

	c := &cacheImpl[K, V]{
		shards: make([]*shard[K, V], cfg.shards),
		hasher: newKeyHasher[K](),
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
	onEvict := func(key K, value V) {
		c.stats.recordEviction()
		if cfg.onEvict != nil {
			cfg.onEvict(key, value)
		}
	}
	for i := range c.shards {
		c.shards[i] = newShard(cfg.capacity, cfg.policy, onEvict)
	}

	if cfg.janitorTick > 0 {
		go c.runJanitor(cfg.janitorTick)
	}
	return c
}

func (c *cacheImpl[K, V]) shardFor(key K) *shard[K, V] {
	idx := c.hasher.hash(key) % uint64(len(c.shards))
	return c.shards[idx]
}

func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	v, ok := c.shardFor(key).get(key, time.Now().UnixNano())
	c.stats.recordGet(ok)
	return v, ok
}

func (c *cacheImpl[K, V]) Stats() Stats {
	return c.stats.snapshot()
}

func (c *cacheImpl[K, V]) Set(key K, value V, opts ...SetOption) {
	sc := setConfig{ttl: c.cfg.defaultTTL}
	for _, opt := range opts {
		opt(&sc)
	}
	c.shardFor(key).set(key, value, sc.ttl)
}

func (c *cacheImpl[K, V]) Delete(key K) bool {
	return c.shardFor(key).delete(key)
}

func (c *cacheImpl[K, V]) Has(key K) bool {
	return c.shardFor(key).has(key, time.Now().UnixNano())
}

func (c *cacheImpl[K, V]) Len() int {
	total := 0
	for _, s := range c.shards {
		total += s.len()
	}
	return total
}

func (c *cacheImpl[K, V]) Keys() []K {
	now := time.Now().UnixNano()
	keys := make([]K, 0, c.Len())
	for _, s := range c.shards {
		keys = s.keys(keys, now)
	}
	return keys
}

func (c *cacheImpl[K, V]) Clear() {
	for _, s := range c.shards {
		s.clear()
	}
}

func (c *cacheImpl[K, V]) Close() error {
	c.closeOnce.Do(func() {
		close(c.stopCh)
	})
	return nil
}

func (c *cacheImpl[K, V]) runJanitor(tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
			now := time.Now().UnixNano()
			for _, s := range c.shards {
				s.sweep(now)
			}
		}
	}
}
