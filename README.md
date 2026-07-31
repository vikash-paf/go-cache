# go-cache

A native, in-memory, generic cache for Go backend services. Think of it as a
miniaturized, single-process Redis: fast reads, TTL expiry, pluggable
eviction, and a small interface anyone can extend or mock.

No dependencies outside the standard library.

## Install

```sh
go get github.com/vikash-paf/go-cache
```

## Quick start

```go
package main

import (
	"fmt"
	"time"

	cache "github.com/vikash-paf/go-cache"
)

func main() {
	c := cache.New[string, int](
		cache.WithDefaultTTL[string, int](time.Minute),
	)
	defer c.Close()

	c.Set("visits", 1)
	if v, ok := c.Get("visits"); ok {
		fmt.Println(v) // 1
	}
}
```

## Design

- **Generic**: `Cache[K comparable, V any]`. Keys and values are whatever
  your code already uses, no `interface{}` boxing.
- **Sharded**: the keyspace is split across N independent shards (default
  32), each with its own lock, so concurrent goroutines rarely contend.
  Reads on an unbounded cache take a read lock only; capacity-bounded caches
  promote to a write lock on Get because recency/frequency bookkeeping needs
  to mutate shared state. See [shard.go](shard.go) for the full tradeoff.
- **TTL**: set a cache-wide default with `WithDefaultTTL`, or override per
  entry with `cache.WithTTL(d)` on a single `Set` call. Expired entries are
  treated as absent on read even before a sweep removes them.
- **Extensible eviction**: capacity limits are enforced through the
  `eviction.Policy[K]` interface. Built-in policies are `eviction.NewLRU`,
  `eviction.NewLFU`, and `eviction.NewFIFO` (all O(1) per operation).
  Implement `eviction.Policy[K]` yourself for anything else (2-random,
  size-weighted, TinyLFU, ...) and pass it via `WithEvictionPolicy`.
- **Background janitor**: a goroutine per cache periodically sweeps expired
  entries so memory isn't held by keys nobody reads again. Configurable via
  `WithJanitorInterval`, or disable it entirely by passing `0`.
- **Observability**: `Stats()` returns hit/miss/eviction counters.
  `WithOnEvict` registers a callback for capacity evictions and TTL
  expiries, useful for logging or cascading invalidation.

## Extending eviction

```go
type myPolicy[K comparable] struct{ /* ... */ }

func (p *myPolicy[K]) Add(key K)          { /* ... */ }
func (p *myPolicy[K]) Hit(key K)          { /* ... */ }
func (p *myPolicy[K]) Remove(key K)       { /* ... */ }
func (p *myPolicy[K]) Evict() (K, bool)   { /* ... */ }
func (p *myPolicy[K]) Len() int           { /* ... */ }

c := cache.New[string, User](
	cache.WithCapacity[string, User](10_000),
	cache.WithEvictionPolicy[string, User](func() eviction.Policy[string] {
		return &myPolicy[string]{}
	}),
)
```

Because each shard gets its own `Policy` instance (built by the `Factory`
you pass in), implementations never need to worry about concurrency inside
`Policy` itself: the shard's lock already serializes every call into it.

## Options reference

| Option | Effect | Default |
|---|---|---|
| `WithShards(n)` | number of internal shards | 32 |
| `WithCapacity(n)` | max entries per shard before eviction | 0 (unbounded) |
| `WithEvictionPolicy(f)` | eviction strategy factory | LRU, if capacity is set |
| `WithDefaultTTL(d)` | TTL applied to `Set` unless overridden | 0 (no expiry) |
| `WithJanitorInterval(d)` | background sweep frequency | 1s |
| `WithOnEvict(fn)` | callback on eviction/expiry | none |

Per-`Set` option: `WithTTL(d)` overrides the cache's default TTL for that
one entry.

## Benchmarks

`go test -bench . -benchmem`:

```
BenchmarkGetUnbounded-10     16.4M ops    67 ns/op    16 B/op   1 allocs/op
BenchmarkGetLRUBounded-10    15.5M ops    77 ns/op    16 B/op   1 allocs/op
BenchmarkSet-10              27.3M ops    45 ns/op    32 B/op   2 allocs/op
```

Numbers will vary by hardware and key type (string keys hash through
`maphash`; integer keys take an allocation-free fast path).

## Status

Core cache, TTL, sharding, and LRU/LFU/FIFO eviction are implemented and
tested (including `-race`). No external dependencies.
