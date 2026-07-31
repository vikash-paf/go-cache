package eviction

import "container/list"

// fifo evicts in strict insertion order and ignores hits, unlike LRU.
type fifo[K comparable] struct {
	ll    *list.List
	index map[K]*list.Element
}

// NewFIFO returns a Factory for a first-in-first-out Policy.
func NewFIFO[K comparable]() Factory[K] {
	return func() Policy[K] {
		return &fifo[K]{
			ll:    list.New(),
			index: make(map[K]*list.Element),
		}
	}
}

func (p *fifo[K]) Add(key K) {
	if _, ok := p.index[key]; ok {
		return
	}
	p.index[key] = p.ll.PushBack(key)
}

func (p *fifo[K]) Hit(K) {}

func (p *fifo[K]) Remove(key K) {
	if el, ok := p.index[key]; ok {
		p.ll.Remove(el)
		delete(p.index, key)
	}
}

func (p *fifo[K]) Evict() (K, bool) {
	el := p.ll.Front()
	if el == nil {
		var zero K
		return zero, false
	}
	key := el.Value.(K)
	p.ll.Remove(el)
	delete(p.index, key)
	return key, true
}

func (p *fifo[K]) Len() int {
	return p.ll.Len()
}
