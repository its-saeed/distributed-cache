package consistent

import (
	"errors"
	"hash/fnv"
	"slices"
	"sort"
	"sync"
)

type FNVHash struct{}

func (h FNVHash) Sum64(key string) uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	return hasher.Sum64()
}

var (
	ErrEmptyRing   = errors.New("hash ring is empty")
	ErrNodeExists  = errors.New("node already exists")
	ErrNodeMissing = errors.New("node not found")
)

// HashRing manages nodes using consistent hashing
type HashRing struct {
	hashFunc FNVHash
	nodes    map[uint64]Node // Hash -> Node mapping
	sorted   []uint64        // Sorted hashes
	mu       sync.RWMutex
}

func NewHashRing() *HashRing {
	return &HashRing{
		hashFunc: FNVHash{},
		nodes:    make(map[uint64]Node),
	}
}

func (hr *HashRing) AddNode(node Node) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hash := hr.hashFunc.Sum64(node.String())
	if _, exists := hr.nodes[hash]; exists {
		return ErrNodeExists
	}

	hr.nodes[hash] = node
	hr.sorted = append(hr.sorted, hash)

	// Keep hashes sorted for efficient lookup
	sort.Slice(hr.sorted, func(i, j int) bool { return hr.sorted[i] < hr.sorted[j] })

	return nil
}

func (hr *HashRing) RemoveNode(node Node) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hash := hr.hashFunc.Sum64(node.String())
	if _, exists := hr.nodes[hash]; !exists {
		return ErrNodeMissing
	}

	delete(hr.nodes, hash)

	// Refresh sorted keys
	hr.sorted = nil
	for k := range hr.nodes {
		hr.sorted = append(hr.sorted, k)
	}
	slices.Sort(hr.sorted)

	return nil
}

// GetNode finds the closest node for a given key
func (hr *HashRing) GetNode(key string) (Node, error) {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	if len(hr.sorted) == 0 {
		return nil, ErrEmptyRing
	}

	hash := hr.hashFunc.Sum64(key)

	// Find the closest node using binary search
	idx := sort.Search(len(hr.sorted), func(i int) bool { return hr.sorted[i] >= hash })
	if idx == len(hr.sorted) {
		idx = 0 // Wrap around to the first node
	}

	return hr.nodes[hr.sorted[idx]], nil
}
