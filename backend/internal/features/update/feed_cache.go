package update

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type nativeFeedCacheEntry struct {
	body []byte
	etag string
}

type nativeFeedCache struct {
	mu    sync.RWMutex
	feeds map[nativeFeedCacheKey]nativeFeedCacheEntry
	loads singleflight.Group
}

type nativeFeedCacheKey struct {
	productID         uuid.UUID
	platform, channel string
}

func newNativeFeedCache() *nativeFeedCache {
	return &nativeFeedCache{feeds: make(map[nativeFeedCacheKey]nativeFeedCacheEntry)}
}

func (c *nativeFeedCache) get(key nativeFeedCacheKey) (nativeFeedCacheEntry, bool) {
	c.mu.RLock()
	feed, ok := c.feeds[key]
	c.mu.RUnlock()
	return feed, ok
}

func (c *nativeFeedCache) load(key nativeFeedCacheKey, generate func() ([]byte, error)) (nativeFeedCacheEntry, error) {
	if entry, ok := c.get(key); ok {
		return entry, nil
	}
	value, err, _ := c.loads.Do(fmt.Sprintf("%s/%s/%s", key.productID, key.platform, key.channel), func() (any, error) {
		if entry, ok := c.get(key); ok {
			return entry, nil
		}
		body, err := generate()
		if err != nil {
			return nativeFeedCacheEntry{}, err
		}
		entry := nativeFeedCacheEntry{body: body, etag: fmt.Sprintf("\"%x\"", sha256.Sum256(body))}
		c.mu.Lock()
		c.feeds[key] = entry
		c.mu.Unlock()
		return entry, nil
	})
	if err != nil {
		return nativeFeedCacheEntry{}, err
	}
	return value.(nativeFeedCacheEntry), nil
}

func (c *nativeFeedCache) invalidateProduct(productID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.feeds {
		if key.productID == productID {
			delete(c.feeds, key)
		}
	}
}
