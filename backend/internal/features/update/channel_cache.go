package update

import (
	"sync"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

type productChannelCache struct {
	mu       sync.RWMutex
	products map[uuid.UUID][]db.UpdateChannel
}

func newProductChannelCache() *productChannelCache {
	return &productChannelCache{products: make(map[uuid.UUID][]db.UpdateChannel)}
}
func (c *productChannelCache) get(id uuid.UUID) ([]db.UpdateChannel, bool) {
	c.mu.RLock()
	v, ok := c.products[id]
	c.mu.RUnlock()
	return v, ok
}
func (c *productChannelCache) set(id uuid.UUID, channels []db.UpdateChannel) {
	c.mu.Lock()
	c.products[id] = channels
	c.mu.Unlock()
}
func (c *productChannelCache) invalidate(id uuid.UUID) {
	c.mu.Lock()
	delete(c.products, id)
	c.mu.Unlock()
}
