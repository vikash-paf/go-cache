package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	cache "github.com/vikash-paf/go-cache"
	"github.com/vikash-paf/go-cache/eviction"
)

func TestGetSetDelete(t *testing.T) {
	c := cache.New[string, int]()
	defer c.Close()

	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected miss on empty cache")
	}

	c.Set("a", 1)
	v, ok := c.Get("a")
	if !ok || v != 1 {
		t.Fatalf("got (%v, %v), want (1, true)", v, ok)
	}

	if !c.Delete("a") {
		t.Fatalf("expected Delete to report existing key")
	}
	if c.Delete("a") {
		t.Fatalf("expected second Delete to report absence")
	}
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected miss after delete")
	}
}

func TestHasAndLen(t *testing.T) {
	c := cache.New[string, int]()
	defer c.Close()

	c.Set("a", 1)
	c.Set("b", 2)

	if !c.Has("a") || !c.Has("b") {
		t.Fatalf("expected both keys present")
	}
	if c.Has("c") {
		t.Fatalf("expected missing key to report absent")
	}
	if got := c.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
}

func TestKeys(t *testing.T) {
	c := cache.New[string, int]()
	defer c.Close()

	want := map[string]bool{"a": true, "b": true, "c": true}
	for k := range want {
		c.Set(k, 1)
	}

	got := c.Keys()
	if len(got) != len(want) {
		t.Fatalf("Keys() returned %d keys, want %d", len(got), len(want))
	}
	for _, k := range got {
		if !want[k] {
			t.Fatalf("unexpected key %q", k)
		}
	}
}

func TestClear(t *testing.T) {
	c := cache.New[string, int]()
	defer c.Close()

	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()

	if got := c.Len(); got != 0 {
		t.Fatalf("Len() after Clear = %d, want 0", got)
	}
}

func TestTTLExpiry(t *testing.T) {
	c := cache.New[string, int](cache.WithDefaultTTL[string, int](20 * time.Millisecond))
	defer c.Close()

	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("expected immediate hit, got (%v, %v)", v, ok)
	}

	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected key to have expired")
	}
}

func TestPerItemTTLOverridesDefault(t *testing.T) {
	c := cache.New[string, int](cache.WithDefaultTTL[string, int](time.Hour))
	defer c.Close()

	c.Set("a", 1, cache.WithTTL(10*time.Millisecond))
	time.Sleep(30 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected short per-item TTL to override long default")
	}
}

func TestJanitorSweepsExpiredEntries(t *testing.T) {
	c := cache.New[string, int](
		cache.WithDefaultTTL[string, int](10*time.Millisecond),
		cache.WithJanitorInterval[string, int](5*time.Millisecond),
	)
	defer c.Close()

	c.Set("a", 1)
	time.Sleep(50 * time.Millisecond)

	if got := c.Len(); got != 0 {
		t.Fatalf("Len() = %d after janitor sweep, want 0", got)
	}
}

func TestCapacityEvictsWithLRU(t *testing.T) {
	c := cache.New[int, int](
		cache.WithShards[int, int](1),
		cache.WithCapacity[int, int](2),
		cache.WithEvictionPolicy[int, int](eviction.NewLRU[int]()),
	)
	defer c.Close()

	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1) // touch 1 so 2 becomes the LRU victim
	c.Set(3, 3)

	if _, ok := c.Get(2); ok {
		t.Fatalf("expected key 2 to be evicted as least recently used")
	}
	if _, ok := c.Get(1); !ok {
		t.Fatalf("expected key 1 to survive (recently touched)")
	}
	if _, ok := c.Get(3); !ok {
		t.Fatalf("expected newly inserted key 3 to be present")
	}
	if got := c.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
}

func TestCapacityEvictsWithFIFO(t *testing.T) {
	c := cache.New[int, int](
		cache.WithShards[int, int](1),
		cache.WithCapacity[int, int](2),
		cache.WithEvictionPolicy[int, int](eviction.NewFIFO[int]()),
	)
	defer c.Close()

	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1) // FIFO ignores hits, so 1 is still the oldest
	c.Set(3, 3)

	if _, ok := c.Get(1); ok {
		t.Fatalf("expected key 1 (first inserted) to be evicted under FIFO")
	}
	if _, ok := c.Get(2); !ok {
		t.Fatalf("expected key 2 to survive")
	}
}

func TestCapacityEvictsWithLFU(t *testing.T) {
	c := cache.New[int, int](
		cache.WithShards[int, int](1),
		cache.WithCapacity[int, int](2),
		cache.WithEvictionPolicy[int, int](eviction.NewLFU[int]()),
	)
	defer c.Close()

	c.Set(1, 1)
	c.Set(2, 2)
	c.Get(1)
	c.Get(1) // key 1 now has more hits than key 2
	c.Set(3, 3)

	if _, ok := c.Get(2); ok {
		t.Fatalf("expected key 2 (least frequently used) to be evicted")
	}
	if _, ok := c.Get(1); !ok {
		t.Fatalf("expected frequently used key 1 to survive")
	}
}

func TestOnEvictCallback(t *testing.T) {
	var mu sync.Mutex
	evicted := map[int]int{}

	c := cache.New[int, int](
		cache.WithShards[int, int](1),
		cache.WithCapacity[int, int](1),
		cache.WithOnEvict[int, int](func(k, v int) {
			mu.Lock()
			evicted[k] = v
			mu.Unlock()
		}),
	)
	defer c.Close()

	c.Set(1, 100)
	c.Set(2, 200) // evicts key 1

	mu.Lock()
	defer mu.Unlock()
	if evicted[1] != 100 {
		t.Fatalf("expected eviction callback for key 1 with value 100, got %v", evicted)
	}
}

func TestStats(t *testing.T) {
	c := cache.New[string, int]()
	defer c.Close()

	c.Set("a", 1)
	c.Get("a")
	c.Get("missing")

	s := c.Stats()
	if s.Hits != 1 || s.Misses != 1 {
		t.Fatalf("Stats() = %+v, want 1 hit and 1 miss", s)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := cache.New[int, int](cache.WithCapacity[int, int](1000))
	defer c.Close()

	const goroutines = 32
	const opsPerG = 2000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		go func(g int) {
			defer wg.Done()
			for i := range opsPerG {
				k := (g*opsPerG + i) % 500
				c.Set(k, k)
				c.Get(k)
				if k%7 == 0 {
					c.Delete(k)
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestGenericValueTypes(t *testing.T) {
	type user struct {
		Name string
		Age  int
	}
	c := cache.New[int, user]()
	defer c.Close()

	c.Set(1, user{Name: "ada", Age: 30})
	v, ok := c.Get(1)
	if !ok || v.Name != "ada" || v.Age != 30 {
		t.Fatalf("got %+v, %v", v, ok)
	}
}

func ExampleNew() {
	c := cache.New[string, int](cache.WithDefaultTTL[string, int](time.Minute))
	defer c.Close()

	c.Set("visits", 1)
	v, ok := c.Get("visits")
	fmt.Println(v, ok)
	// Output: 1 true
}
