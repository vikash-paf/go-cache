package cache

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
)

// keyHasher maps an arbitrary comparable key to a shard index. maphash gives
// us a fast, randomly-seeded hash without asking callers to implement
// anything, at the cost of a fmt.Sprint fallback for key types it can't
// hash directly (structs, etc.). Common key types (strings, ints, and other
// fixed-width scalars) take a fast path with no allocation.
type keyHasher[K comparable] struct {
	seed maphash.Seed
}

func newKeyHasher[K comparable]() keyHasher[K] {
	return keyHasher[K]{seed: maphash.MakeSeed()}
}

func (h keyHasher[K]) hash(key K) uint64 {
	switch k := any(key).(type) {
	case string:
		return maphash.String(h.seed, k)
	case []byte:
		return maphash.Bytes(h.seed, k)
	case int:
		return h.hashUint64(uint64(k))
	case int8:
		return h.hashUint64(uint64(k))
	case int16:
		return h.hashUint64(uint64(k))
	case int32:
		return h.hashUint64(uint64(k))
	case int64:
		return h.hashUint64(uint64(k))
	case uint:
		return h.hashUint64(uint64(k))
	case uint8:
		return h.hashUint64(uint64(k))
	case uint16:
		return h.hashUint64(uint64(k))
	case uint32:
		return h.hashUint64(uint64(k))
	case uint64:
		return h.hashUint64(k)
	default:
		return maphash.String(h.seed, fmt.Sprint(k))
	}
}

func (h keyHasher[K]) hashUint64(v uint64) uint64 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	return maphash.Bytes(h.seed, buf[:])
}
