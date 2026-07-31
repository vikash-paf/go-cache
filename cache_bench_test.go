package cache_test

import (
	"strconv"
	"testing"

	cache "github.com/vikash-paf/go-cache"
)

func BenchmarkGetUnbounded(b *testing.B) {
	c := cache.New[string, int]()
	defer c.Close()

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
		c.Set(keys[i], i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(keys[i%len(keys)])
			i++
		}
	})
}

func BenchmarkGetLRUBounded(b *testing.B) {
	c := cache.New[string, int](cache.WithCapacity[string, int](1000))
	defer c.Close()

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
		c.Set(keys[i], i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Get(keys[i%len(keys)])
			i++
		}
	})
}

func BenchmarkSet(b *testing.B) {
	c := cache.New[string, int]()
	defer c.Close()

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(keys[i%len(keys)], i)
			i++
		}
	})
}
