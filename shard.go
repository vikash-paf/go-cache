package cache

import (
	"sync"
	"time"

	"github.com/vikash-paf/go-cache/eviction"
)

// shard is one partition of the cache's keyspace. Splitting the cache into
// shards lets independent goroutines touch different shards without
// contending on a single lock, which is where most of the cache's
// concurrent throughput comes from.
//
// Locking strategy: when no eviction policy is configured (unbounded cache)
// reads take shard.mu in read mode only, so concurrent Gets never block each
// other. As soon as a Policy is attached (capacity-bounded cache) every Get
// must also record a Hit against the policy, which mutates shared state, so
// reads are promoted to a full write lock. This is a deliberate tradeoff:
// unbounded caches get lock-free-reader-style scaling, bounded ones pay for
// accurate recency/frequency bookkeeping.
type shard[K comparable, V any] struct {
	mu       sync.RWMutex
	data     map[K]item[V]
	policy   eviction.Policy[K]
	factory  eviction.Factory[K]
	capacity int
	onEvict  func(key K, value V)
}

func newShard[K comparable, V any](capacity int, factory eviction.Factory[K], onEvict func(K, V)) *shard[K, V] {
	s := &shard[K, V]{
		data:     make(map[K]item[V]),
		factory:  factory,
		capacity: capacity,
		onEvict:  onEvict,
	}
	if capacity > 0 && factory != nil {
		s.policy = factory()
	}
	return s
}

func (s *shard[K, V]) get(key K, now int64) (V, bool) {
	if s.policy == nil {
		s.mu.RLock()
		it, ok := s.data[key]
		s.mu.RUnlock()
		if !ok || it.expired(now) {
			var zero V
			return zero, false
		}
		return it.value, true
	}

	s.mu.Lock()
	it, ok := s.data[key]
	if !ok {
		s.mu.Unlock()
		var zero V
		return zero, false
	}
	if it.expired(now) {
		delete(s.data, key)
		s.policy.Remove(key)
		s.mu.Unlock()
		var zero V
		return zero, false
	}
	s.policy.Hit(key)
	s.mu.Unlock()
	return it.value, true
}

func (s *shard[K, V]) set(key K, value V, ttl time.Duration) {
	it := newItem(value, ttl)

	s.mu.Lock()
	defer s.mu.Unlock()

	_, existed := s.data[key]
	if !existed && s.capacity > 0 && s.policy != nil && len(s.data) >= s.capacity {
		if victim, ok := s.policy.Evict(); ok {
			if v, ok := s.data[victim]; ok {
				delete(s.data, victim)
				if s.onEvict != nil {
					s.onEvict(victim, v.value)
				}
			}
		}
	}

	s.data[key] = it
	if s.policy != nil {
		s.policy.Add(key)
	}
}

func (s *shard[K, V]) delete(key K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.data[key]
	if !ok {
		return false
	}
	delete(s.data, key)
	if s.policy != nil {
		s.policy.Remove(key)
	}
	return true
}

func (s *shard[K, V]) has(key K, now int64) bool {
	s.mu.RLock()
	it, ok := s.data[key]
	s.mu.RUnlock()
	return ok && !it.expired(now)
}

func (s *shard[K, V]) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *shard[K, V]) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[K]item[V])
	if s.capacity > 0 && s.factory != nil {
		s.policy = s.factory()
	}
}

// keys appends every non-expired key in the shard to dst and returns it.
func (s *shard[K, V]) keys(dst []K, now int64) []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, it := range s.data {
		if !it.expired(now) {
			dst = append(dst, k)
		}
	}
	return dst
}

// sweep removes every expired entry from the shard. It is called
// periodically by the janitor and also opportunistically ignorable on Get,
// which already treats expired entries as absent.
func (s *shard[K, V]) sweep(now int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, it := range s.data {
		if it.expired(now) {
			delete(s.data, k)
			if s.policy != nil {
				s.policy.Remove(k)
			}
			if s.onEvict != nil {
				s.onEvict(k, it.value)
			}
		}
	}
}
