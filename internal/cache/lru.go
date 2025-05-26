package cache

import (
	"container/list"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Entry struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

func NewEntry(key string, value []byte, ttl time.Duration) *Entry {
	e := &Entry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
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
	if time.Now().After(entry.ExpiresAt) {
		l.list.Remove(entryWrapper)
		delete(l.cache, key)
		return nil, false
	}

	l.mu.Lock()
	l.list.MoveToFront(entryWrapper)
	l.mu.Unlock()

	return entry.Value, true
}

func (l *LRUCache) SetWithTtl(key string, value []byte, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if key already exists
	if elem, ok := l.cache[key]; ok {
		entry := elem.Value.(*Entry)
		entry.Value = value
		entry.ExpiresAt = time.Now().Add(ttl)
		l.list.MoveToFront(elem)
		return
	}

	entry := NewEntry(key, value, ttl)
	elem := l.list.PushFront(entry)
	l.cache[key] = elem

	// Evict if necessary
	if l.list.Len() > l.capacity {
		l.evict()
	}
}

func (l *LRUCache) Set(key string, value []byte) {
	l.SetWithTtl(key, value, time.Duration(100)*time.Hour)
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

type PersistedCache struct {
	Capacity int
	Entries  []Entry // ordered front (MRU) to back (LRU)
}

func (l *LRUCache) SaveToFile(filename string) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var entries []Entry
	for elem := l.list.Front(); elem != nil; elem = elem.Next() {
		entry := elem.Value.(*Entry)
		entries = append(entries, *entry)
	}

	persist := PersistedCache{
		Capacity: l.capacity,
		Entries:  entries,
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(persist)
}

func (l *LRUCache) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var persist PersistedCache
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&persist); err != nil {
		return err
	}

	for i := len(persist.Entries) - 1; i >= 0; i-- {
		e := persist.Entries[i]
		if time.Now().Before(e.ExpiresAt) {
			l.SetWithTtl(e.Key, e.Value, time.Until(e.ExpiresAt))
		}
	}
	return nil
}
