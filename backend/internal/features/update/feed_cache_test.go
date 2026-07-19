package update

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNativeFeedCacheInvalidatesOnlyProductFeeds(t *testing.T) {
	cache := newNativeFeedCache()
	product := uuid.New()
	otherProduct := uuid.New()
	key := nativeFeedCacheKey{productID: product, platform: "macos", channel: "stable"}
	otherKey := nativeFeedCacheKey{productID: otherProduct, platform: "macos", channel: "stable"}

	_, _ = cache.load(key, func() ([]byte, error) { return []byte("feed"), nil })
	_, _ = cache.load(otherKey, func() ([]byte, error) { return []byte("other"), nil })
	cache.invalidateProduct(product)

	if _, ok := cache.get(key); ok {
		t.Fatal("expected invalidated product feed to be removed")
	}
	if got, ok := cache.get(otherKey); !ok || string(got.body) != "other" {
		t.Fatalf("expected other product feed to remain, got %q (exists=%t)", got.body, ok)
	}
}

func TestNativeFeedCacheDeduplicatesConcurrentLoads(t *testing.T) {
	cache := newNativeFeedCache()
	key := nativeFeedCacheKey{productID: uuid.New(), platform: "macos", channel: "stable"}
	var calls atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entry, err := cache.load(key, func() ([]byte, error) {
				calls.Add(1)
				time.Sleep(10 * time.Millisecond)
				return []byte("feed"), nil
			})
			if err != nil || string(entry.body) != "feed" {
				t.Errorf("load = %q, %v", entry.body, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("generator called %d times", calls.Load())
	}
}
