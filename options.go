package cache

import (
	"time"

	"github.com/vikash-paf/go-cache/eviction"
)

// config holds the fully-resolved cache configuration built from Option
// values. It is unexported: callers only ever see Option constructors.
type config[K comparable, V any] struct {
	shards      int
	capacity    int // per-shard capacity; 0 means unbounded
	defaultTTL  time.Duration
	janitorTick time.Duration
	policy      eviction.Factory[K]
	onEvict     func(key K, value V)
}

func defaultConfig[K comparable, V any]() config[K, V] {
	return config[K, V]{
		shards:      32,
		capacity:    0,
		defaultTTL:  0,
		janitorTick: time.Second,
		policy:      nil,
		onEvict:     nil,
	}
}

// Option configures a Cache at construction time.
type Option[K comparable, V any] func(*config[K, V])

// WithShards sets the number of internal shards used to partition the
// keyspace. More shards reduce lock contention under concurrent access at
// the cost of slightly higher memory overhead and less precise global
// capacity accounting. Must be a positive number; non-positive values are
// ignored. Default: 32.
func WithShards[K comparable, V any](n int) Option[K, V] {
	return func(c *config[K, V]) {
		if n > 0 {
			c.shards = n
		}
	}
}

// WithCapacity bounds each shard to at most n entries, evicting via the
// configured eviction Policy once full. Capacity is enforced per shard
// rather than globally so a hot shard can never block on a global counter;
// with N shards, total capacity is approximately n*N. A value of 0 (the
// default) means unbounded.
func WithCapacity[K comparable, V any](perShard int) Option[K, V] {
	return func(c *config[K, V]) {
		if perShard > 0 {
			c.capacity = perShard
		}
	}
}

// WithEvictionPolicy sets the eviction.Factory used to build a fresh
// eviction.Policy for every shard. Only meaningful together with
// WithCapacity. Defaults to eviction.NewLRU when a capacity is set but no
// policy is chosen.
func WithEvictionPolicy[K comparable, V any](f eviction.Factory[K]) Option[K, V] {
	return func(c *config[K, V]) {
		c.policy = f
	}
}

// WithDefaultTTL sets the time-to-live applied to entries written with Set
// when no per-item TTL is given via WithTTL. Zero (the default) means
// entries never expire unless given an explicit TTL.
func WithDefaultTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *config[K, V]) {
		c.defaultTTL = ttl
	}
}

// WithJanitorInterval sets how often each shard sweeps for expired entries
// in the background. Expired entries are also skipped lazily on Get
// regardless of this setting, so the janitor only affects how quickly
// memory for expired-but-unread entries is reclaimed. A value <= 0 disables
// the background sweep entirely. Default: 1 second.
func WithJanitorInterval[K comparable, V any](d time.Duration) Option[K, V] {
	return func(c *config[K, V]) {
		c.janitorTick = d
	}
}

// WithOnEvict registers a callback invoked whenever an entry is removed due
// to capacity eviction or TTL expiry (not on explicit Delete). The callback
// runs synchronously on the goroutine that triggered the eviction, so it
// must be fast and must not call back into the same Cache.
func WithOnEvict[K comparable, V any](fn func(key K, value V)) Option[K, V] {
	return func(c *config[K, V]) {
		c.onEvict = fn
	}
}

// SetOption configures a single Set call.
type SetOption func(*setConfig)

type setConfig struct {
	ttl    time.Duration
	setTTL bool
}

// WithTTL overrides the cache's default TTL for a single Set call. A TTL of
// 0 means the entry never expires.
func WithTTL(ttl time.Duration) SetOption {
	return func(c *setConfig) {
		c.ttl = ttl
		c.setTTL = true
	}
}
