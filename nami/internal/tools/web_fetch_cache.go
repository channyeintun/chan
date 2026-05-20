package tools

import (
	"container/list"
	"sync"
	"time"
)

type webFetchCache struct {
	mu        sync.Mutex
	maxBytes  int64
	ttl       time.Duration
	usedBytes int64
	entries   map[string]*list.Element
	recency   *list.List
}

type webFetchCacheEntry struct {
	key       string
	value     webFetchContent
	size      int64
	expiresAt time.Time
}

func newWebFetchCache(maxBytes int64, ttl time.Duration) *webFetchCache {
	return &webFetchCache{
		maxBytes: maxBytes,
		ttl:      ttl,
		entries:  make(map[string]*list.Element),
		recency:  list.New(),
	}
}

func (c *webFetchCache) Get(key string) (webFetchContent, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[key]
	if !ok {
		return webFetchContent{}, false
	}
	entry := element.Value.(*webFetchCacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.removeElement(element)
		return webFetchContent{}, false
	}
	c.recency.MoveToFront(element)
	return entry.value, true
}

func (c *webFetchCache) Set(key string, value webFetchContent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.entries[key]; ok {
		c.removeElement(element)
	}

	entry := &webFetchCacheEntry{
		key:       key,
		value:     value,
		size:      int64(len(value.Markdown) + len(value.ContentType) + len(value.StatusText) + len(value.URL)),
		expiresAt: time.Now().Add(c.ttl),
	}
	element := c.recency.PushFront(entry)
	c.entries[key] = element
	c.usedBytes += entry.size

	for c.usedBytes > c.maxBytes && c.recency.Len() > 0 {
		c.removeElement(c.recency.Back())
	}
}

func (c *webFetchCache) removeElement(element *list.Element) {
	if element == nil {
		return
	}
	entry := element.Value.(*webFetchCacheEntry)
	delete(c.entries, entry.key)
	c.usedBytes -= entry.size
	c.recency.Remove(element)
	if c.usedBytes < 0 {
		c.usedBytes = 0
	}
}
