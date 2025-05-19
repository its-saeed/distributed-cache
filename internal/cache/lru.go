package cache

import (
	"container/list"
	"sync"
)

type Entry struct {
	Key   string
	Value []byte
}

func NewEntry(key string, value []byte) *Entry {
	e := &Entry{
		Key:   key,
		Value: value,
	}
	return e
}

type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.RWMutex
}

func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		panic("capacity must be positive")
	}
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (l *LRUCache) Get(key string) ([]byte, bool) {
	l.mu.RLock()
	entryWrapper, ok := l.cache[key]
	l.mu.RUnlock()

	if !ok {
		return nil, false
	}

	entry := entryWrapper.Value.(*Entry)

	l.mu.Lock()
	l.list.MoveToFront(entryWrapper)
	l.mu.Unlock()

	return entry.Value, true
}

func (l *LRUCache) Set(key string, value []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if key already exists
	if elem, ok := l.cache[key]; ok {
		entry := elem.Value.(*Entry)
		entry.Value = value
		l.list.MoveToFront(elem)
		return
	}

	entry := NewEntry(key, value)
	elem := l.list.PushFront(entry)
	l.cache[key] = elem

	// Evict if necessary
	if l.list.Len() > l.capacity {
		l.evict()
	}
}

// evict removes the least recently used item
func (l *LRUCache) evict() {
	elem := l.list.Back()
	if elem == nil {
		return
	}

	entry := elem.Value.(*Entry)
	l.list.Remove(elem)
	delete(l.cache, entry.Key)
}

// Delete removes a key from the cache
func (l *LRUCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.cache[key]; ok {
		l.list.Remove(elem)
		delete(l.cache, key)
	}
}

func (l *LRUCache) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.list.Len()
}
