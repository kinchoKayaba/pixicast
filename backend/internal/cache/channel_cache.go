package cache

import (
	"fmt"
	"time"

	"github.com/kinchoKayaba/pixicast/backend/internal/youtube"
)

// ChannelCache はチャンネル情報のキャッシュ
type ChannelCache struct {
	cache *MemoryCache
}

// NewChannelCache は新しいChannelCacheを作成
func NewChannelCache() *ChannelCache {
	return &ChannelCache{
		cache: NewMemoryCache(),
	}
}

// GetChannelDetails はキャッシュからチャンネル詳細を取得
func (cc *ChannelCache) GetChannelDetails(channelID string) (*youtube.ChannelDetails, bool) {
	key := fmt.Sprintf("channel:%s", channelID)
	value, exists := cc.cache.Get(key)
	if !exists {
		return nil, false
	}

	details, ok := value.(*youtube.ChannelDetails)
	return details, ok
}

// SetChannelDetails はキャッシュにチャンネル詳細を保存
// priority: high=1h, medium=3h, low=6h
func (cc *ChannelCache) SetChannelDetails(channelID string, details *youtube.ChannelDetails, priority string) {
	key := fmt.Sprintf("channel:%s", channelID)

	var ttl time.Duration
	switch priority {
	case "high":
		ttl = 1 * time.Hour
	case "medium":
		ttl = 3 * time.Hour
	default: // low
		ttl = 6 * time.Hour
	}

	cc.cache.Set(key, details, ttl)
}

// DeleteChannelDetails はキャッシュからチャンネル詳細を削除
func (cc *ChannelCache) DeleteChannelDetails(channelID string) {
	key := fmt.Sprintf("channel:%s", channelID)
	cc.cache.Delete(key)
}

// Clear はすべてのキャッシュをクリア
func (cc *ChannelCache) Clear() {
	cc.cache.Clear()
}

// Len はキャッシュアイテム数を返す
func (cc *ChannelCache) Len() int {
	return cc.cache.Len()
}
