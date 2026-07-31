package cache

import "time"

// item is the internal wrapper stored in a shard's map.
// It carries expiry information alongside the raw value so the hot Get path
// never needs a second lookup to check TTL.
type item[V any] struct {
	value     V
	expiresAt int64 // unix nano; 0 means "no expiry"
}

func (it *item[V]) expired(now int64) bool {
	return it.expiresAt != 0 && it.expiresAt <= now
}

func newItem[V any](value V, ttl time.Duration) item[V] {
	if ttl <= 0 {
		return item[V]{value: value}
	}
	return item[V]{value: value, expiresAt: time.Now().Add(ttl).UnixNano()}
}
