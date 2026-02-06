package cache

import (
	"sync"
	"time"
)

// CacheItem はキャッシュアイテムの構造
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// MemoryCache はインメモリキャッシュ
type MemoryCache struct {
	items map[string]CacheItem
	mu    sync.RWMutex
}

// NewMemoryCache は新しいMemoryCacheを作成
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]CacheItem),
	}

	// 5分ごとに期限切れアイテムを削除
	go cache.startCleanupRoutine(5 * time.Minute)

	return cache
}

// Set はキャッシュに値を設定
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Get はキャッシュから値を取得
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 期限切れチェック
	if time.Now().After(item.Expiration) {
		return nil, false
	}

	return item.Value, true
}

// Delete はキャッシュから値を削除
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear はすべてのキャッシュをクリア
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]CacheItem)
}

// Len はキャッシュアイテム数を返す
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// startCleanupRoutine は期限切れアイテムを定期的に削除
func (c *MemoryCache) startCleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup は期限切れアイテムを削除
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.Expiration) {
			delete(c.items, key)
		}
	}
}
