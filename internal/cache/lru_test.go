package cache

import (
	"testing"
)

func TestNewLRUCache(t *testing.T) {
	cache := NewLRUCache(2)

	if cache.capacity != 2 {
		t.Errorf("Expected capacity 2, got %d", cache.capacity)
	}
	if cache.Len() != 0 {
		t.Errorf("Expected empty cache, got %d items", cache.Len())
	}
}

func TestCacheSetGet(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("foo", []byte("bar"))

	value, found := cache.Get("foo")
	if !found || string(value) != "bar" {
		t.Errorf("Expected 'bar', got '%s'", string(value))
	}
}

func TestCacheEviction(t *testing.T) {
	cache := NewLRUCache(2)

	cache.Set("a", []byte("1"))
	cache.Set("b", []byte("2"))
	cache.Set("c", []byte("3")) // Should evict "a"

	_, found := cache.Get("a")
	if found {
		t.Errorf("Expected 'a' to be evicted")
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("key", []byte("value1"))
	cache.Set("key", []byte("value2")) // Should overwrite

	value, _ := cache.Get("key")
	if string(value) != "value2" {
		t.Errorf("Expected 'value2', got '%s'", string(value))
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewLRUCache(2)
	cache.Set("deleteKey", []byte("data"))
	cache.Delete("deleteKey")

	_, found := cache.Get("deleteKey")
	if found {
		t.Errorf("Expected 'deleteKey' to be removed")
	}
}
