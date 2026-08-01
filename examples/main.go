package main

import (
	"fmt"
	"time"

	cache "github.com/vikash-paf/go-cache"
	"github.com/vikash-paf/go-cache/eviction"
)

type UserProfile struct {
	ID    int
	Name  string
	Email string
}

func main() {
	fmt.Println("=== go-cache Example Application ===")

	// 1. Initialize a cache with capacity of 100 entries per shard, LRU eviction, and 2s default TTL
	userCache := cache.New[string, UserProfile](
		cache.WithCapacity[string, UserProfile](100),
		cache.WithEvictionPolicy[string, UserProfile](eviction.NewLRU[string]()),
		cache.WithDefaultTTL[string, UserProfile](2*time.Second),
		cache.WithOnEvict[string, UserProfile](func(key string, val UserProfile) {
			fmt.Printf("[Evict/Expire Callback] Key evicted/expired: %s (User: %s)\n", key, val.Name)
		}),
	)
	defer userCache.Close()

	// 2. Set items in cache
	fmt.Println("\nSetting user profiles in cache...")
	userCache.Set("user:101", UserProfile{ID: 101, Name: "Alice", Email: "alice@example.com"})
	userCache.Set("user:102", UserProfile{ID: 102, Name: "Bob", Email: "bob@example.com"})

	// Set with custom TTL override (10 seconds for user:103)
	userCache.Set("user:103", UserProfile{ID: 103, Name: "Charlie", Email: "charlie@example.com"}, cache.WithTTL(10*time.Second))

	// 3. Retrieve item
	if u, ok := userCache.Get("user:101"); ok {
		fmt.Printf("Found user:101 -> Name: %s, Email: %s\n", u.Name, u.Email)
	} else {
		fmt.Println("user:101 not found")
	}

	// 4. Verify TTL expiry
	fmt.Println("\nWaiting 2.5 seconds for default TTL to expire...")
	time.Sleep(2500 * time.Millisecond)

	if _, ok := userCache.Get("user:101"); ok {
		fmt.Println("user:101 still found")
	} else {
		fmt.Println("user:101 has expired!")
	}

	// Key with longer TTL (user:103) should still exist
	if u, ok := userCache.Get("user:103"); ok {
		fmt.Printf("user:103 still present -> Name: %s\n", u.Name)
	}

	// 5. Check Cache Statistics
	stats := userCache.Stats()
	fmt.Printf("\nCache Stats -> Hits: %d, Misses: %d, Evictions: %d\n", stats.Hits, stats.Misses, stats.Evictions)
}
