package cache

import "sync/atomic"

// Stats is a point-in-time snapshot of cache activity counters.
type Stats struct {
	Hits      uint64
	Misses    uint64
	Evictions uint64 // capacity evictions and TTL expiries combined
}

// statsCounters holds the live atomic counters backing Stats snapshots.
type statsCounters struct {
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

func (s *statsCounters) recordGet(hit bool) {
	if hit {
		s.hits.Add(1)
	} else {
		s.misses.Add(1)
	}
}

func (s *statsCounters) recordEviction() {
	s.evictions.Add(1)
}

func (s *statsCounters) snapshot() Stats {
	return Stats{
		Hits:      s.hits.Load(),
		Misses:    s.misses.Load(),
		Evictions: s.evictions.Load(),
	}
}
