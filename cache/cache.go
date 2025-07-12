package cache

import (
	"sync"
	"time"
)

var CacheInstance *Cache

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

type Cache struct {
	data sync.Map
}

func (c *Cache) Get(key string) (interface{}, bool) {
	entry, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}

	// Check if the entry has expired
	if entry.(cacheEntry).expiration.Before(time.Now()) {
		c.data.Delete(key)
		return nil, false
	}

	return entry.(cacheEntry).value, true
}

func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	entry := cacheEntry{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	c.data.Store(key, entry)
}

func (c *Cache) purgeExpiredEntries() {
	for {
		<-time.After(1 * time.Second) // Adjust the interval as needed
		c.data.Range(func(key, value interface{}) bool {
			if value.(cacheEntry).expiration.Before(time.Now()) {
				c.data.Delete(key)
			}
			return true
		})
	}
}

func NewCache() *Cache {
	cache := &Cache{}
	go cache.purgeExpiredEntries()
	return cache
}

func init() {
	CacheInstance = NewCache()
}
