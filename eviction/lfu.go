package eviction

import "container/list"

// lfu is an O(1) least-frequently-used policy: each frequency has its own
// list of keys (ordered by recency within that frequency for tie-breaking),
// and minFreq always points at the lowest non-empty bucket so Evict never
// scans.
type lfu[K comparable] struct {
	nodes   map[K]*lfuNode[K]
	buckets map[int]*list.List
	minFreq int
}

type lfuNode[K comparable] struct {
	key  K
	freq int
	el   *list.Element
}

// NewLFU returns a Factory for a least-frequently-used Policy.
func NewLFU[K comparable]() Factory[K] {
	return func() Policy[K] {
		return &lfu[K]{
			nodes:   make(map[K]*lfuNode[K]),
			buckets: make(map[int]*list.List),
		}
	}
}

func (p *lfu[K]) bucket(freq int) *list.List {
	b, ok := p.buckets[freq]
	if !ok {
		b = list.New()
		p.buckets[freq] = b
	}
	return b
}

func (p *lfu[K]) Add(key K) {
	if n, ok := p.nodes[key]; ok {
		p.bump(n)
		return
	}
	n := &lfuNode[K]{key: key, freq: 1}
	n.el = p.bucket(1).PushFront(n)
	p.nodes[key] = n
	p.minFreq = 1
}

func (p *lfu[K]) Hit(key K) {
	if n, ok := p.nodes[key]; ok {
		p.bump(n)
	}
}

func (p *lfu[K]) bump(n *lfuNode[K]) {
	oldBucket := p.bucket(n.freq)
	oldBucket.Remove(n.el)
	if oldBucket.Len() == 0 {
		delete(p.buckets, n.freq)
		if p.minFreq == n.freq {
			p.minFreq++
		}
	}
	n.freq++
	n.el = p.bucket(n.freq).PushFront(n)
}

func (p *lfu[K]) Remove(key K) {
	n, ok := p.nodes[key]
	if !ok {
		return
	}
	b := p.bucket(n.freq)
	b.Remove(n.el)
	if b.Len() == 0 {
		delete(p.buckets, n.freq)
	}
	delete(p.nodes, key)
}

func (p *lfu[K]) Evict() (K, bool) {
	b, ok := p.buckets[p.minFreq]
	for !ok || b.Len() == 0 {
		var zero K
		if len(p.nodes) == 0 {
			return zero, false
		}
		// minFreq is stale (can happen if callers mutate out of order);
		// recover by scanning for the true minimum.
		p.minFreq = p.recomputeMinFreq()
		b, ok = p.buckets[p.minFreq]
	}
	el := b.Back()
	n := el.Value.(*lfuNode[K])
	b.Remove(el)
	if b.Len() == 0 {
		delete(p.buckets, p.minFreq)
	}
	delete(p.nodes, n.key)
	return n.key, true
}

func (p *lfu[K]) recomputeMinFreq() int {
	min := -1
	for f, b := range p.buckets {
		if b.Len() == 0 {
			continue
		}
		if min == -1 || f < min {
			min = f
		}
	}
	return min
}

func (p *lfu[K]) Len() int {
	return len(p.nodes)
}
