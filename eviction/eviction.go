// Package eviction defines the pluggable eviction strategy used by a cache
// shard once it reaches capacity, plus a set of ready-to-use policies
// (LRU, LFU, FIFO). Consumers of github.com/vikash-paf/go-cache can implement
// Policy themselves to plug in any strategy the built-ins don't cover.
package eviction

// Policy tracks key access order/frequency for a single shard and decides
// which key to evict when the shard is full. Implementations must be safe to
// call only under the caller's lock: the cache shard already serializes
// access, so a Policy does not need its own internal locking.
type Policy[K comparable] interface {
	// Add registers a newly inserted key with the policy.
	Add(key K)
	// Hit records an access (read or update) to an existing key.
	Hit(key K)
	// Remove drops a key from the policy's bookkeeping, e.g. after a
	// deletion or expiry that did not go through Evict.
	Remove(key K)
	// Evict picks a victim key to remove and stops tracking it. ok is
	// false when the policy has nothing to evict.
	Evict() (key K, ok bool)
	// Len reports how many keys the policy is currently tracking.
	Len() int
}

// Factory builds a fresh Policy instance. The cache calls this once per
// shard so every shard gets an independent policy instance.
type Factory[K comparable] func() Policy[K]
