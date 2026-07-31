package eviction_test

import (
	"testing"

	"github.com/vikash-paf/go-cache/eviction"
)

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	p := eviction.NewLRU[int]()()
	p.Add(1)
	p.Add(2)
	p.Add(3)
	p.Hit(1) // 1 becomes most recent; 2 is now the LRU victim

	key, ok := p.Evict()
	if !ok || key != 2 {
		t.Fatalf("Evict() = (%v, %v), want (2, true)", key, ok)
	}
	if p.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", p.Len())
	}
}

func TestFIFOEvictsInsertionOrderIgnoringHits(t *testing.T) {
	p := eviction.NewFIFO[int]()()
	p.Add(1)
	p.Add(2)
	p.Hit(1) // FIFO ignores hits entirely

	key, ok := p.Evict()
	if !ok || key != 1 {
		t.Fatalf("Evict() = (%v, %v), want (1, true)", key, ok)
	}
}

func TestLFUEvictsLeastFrequentlyUsed(t *testing.T) {
	p := eviction.NewLFU[int]()()
	p.Add(1)
	p.Add(2)
	p.Hit(1)
	p.Hit(1) // key 1: 3 accesses, key 2: 1 access

	key, ok := p.Evict()
	if !ok || key != 2 {
		t.Fatalf("Evict() = (%v, %v), want (2, true)", key, ok)
	}
}

func TestRemove(t *testing.T) {
	p := eviction.NewLRU[string]()()
	p.Add("a")
	p.Add("b")
	p.Remove("a")

	if p.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", p.Len())
	}
	key, ok := p.Evict()
	if !ok || key != "b" {
		t.Fatalf("Evict() = (%v, %v), want (b, true)", key, ok)
	}
}

func TestEvictOnEmptyReportsFalse(t *testing.T) {
	for _, f := range []func() eviction.Policy[int]{
		eviction.NewLRU[int](), eviction.NewFIFO[int](), eviction.NewLFU[int](),
	} {
		p := f()
		if _, ok := p.Evict(); ok {
			t.Fatalf("Evict() on empty policy should report false")
		}
	}
}
