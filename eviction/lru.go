package eviction

import "container/list"

// lru is a classic doubly-linked-list LRU: Hit moves a key to the front,
// Evict removes from the back. All operations are O(1).
type lru[K comparable] struct {
	ll    *list.List
	index map[K]*list.Element
}

// NewLRU returns a Factory for a least-recently-used Policy.
func NewLRU[K comparable]() Factory[K] {
	return func() Policy[K] {
		return &lru[K]{
			ll:    list.New(),
			index: make(map[K]*list.Element),
		}
	}
}

func (p *lru[K]) Add(key K) {
	if el, ok := p.index[key]; ok {
		p.ll.MoveToFront(el)
		return
	}
	p.index[key] = p.ll.PushFront(key)
}

func (p *lru[K]) Hit(key K) {
	if el, ok := p.index[key]; ok {
		p.ll.MoveToFront(el)
	}
}

func (p *lru[K]) Remove(key K) {
	if el, ok := p.index[key]; ok {
		p.ll.Remove(el)
		delete(p.index, key)
	}
}

func (p *lru[K]) Evict() (K, bool) {
	el := p.ll.Back()
	if el == nil {
		var zero K
		return zero, false
	}
	key := el.Value.(K)
	p.ll.Remove(el)
	delete(p.index, key)
	return key, true
}

func (p *lru[K]) Len() int {
	return p.ll.Len()
}
