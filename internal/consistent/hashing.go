package consistent

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/spaolacci/murmur3"
)

type MurmurHash struct{}

func (h MurmurHash) Sum64(key string) uint64 {
	return murmur3.Sum64([]byte(key))
}

var (
	ErrEmptyRing   = errors.New("hash ring is empty")
	ErrNodeExists  = errors.New("node already exists")
	ErrNodeMissing = errors.New("node not found")
)

// HashRing manages nodes using consistent hashing
type HashRing struct {
	hashFunc MurmurHash
	nodes    map[uint64]Node // Hash -> Node mapping
	sorted   []uint64        // Sorted hashes
	mu       sync.RWMutex
}

func NewHashRing() *HashRing {
	return &HashRing{
		hashFunc: MurmurHash{},
		nodes:    make(map[uint64]Node),
	}
}

func (hr *HashRing) AddNode(node Node, numVirtualNodes int) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	for i := range numVirtualNodes {
		virtualKey := fmt.Sprintf("%s#%d", node.String(), i)
		hash := hr.hashFunc.Sum64(virtualKey)
		if _, exists := hr.nodes[hash]; exists {
			return ErrNodeExists
		}

		hr.nodes[hash] = node

		idx := sort.Search(len(hr.sorted), func(i int) bool { return hr.sorted[i] >= hash })
		hr.sorted = append(hr.sorted[:idx], append([]uint64{hash}, hr.sorted[idx:]...)...)
	}

	// Keep hashes sorted for efficient lookup
	slices.Sort(hr.sorted)

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
	fmt.Println("GetNode", hash, key)

	// Find the closest node using binary search
	idx := sort.Search(len(hr.sorted), func(i int) bool { return hr.sorted[i] >= hash })
	if idx == len(hr.sorted) {
		idx = 0 // Wrap around to the first node
	}

	return hr.nodes[hr.sorted[idx]], nil
}
