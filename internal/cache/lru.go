package cache

import (
	"container/list"
	"encoding/json"
	"os"
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
		l.Set(e.Key, e.Value)
	}
	return nil
}
