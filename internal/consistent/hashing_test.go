package consistent

import (
	"testing"
)

type MockNode struct {
	ID string
}

func (n *MockNode) String() string {
	return n.ID
}

func TestNewHashRing(t *testing.T) {
	ring := NewHashRing()

	if len(ring.nodes) != 0 {
		t.Errorf("Expected empty hash ring, got %d nodes", len(ring.nodes))
	}
}

func TestAddNode(t *testing.T) {
	ring := NewHashRing()
	node := &MockNode{ID: "node1"}

	err := ring.AddNode(node)
	if err != nil {
		t.Errorf("Error adding node: %v", err)
	}

	if len(ring.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(ring.nodes))
	}
}

func TestDuplicateNode(t *testing.T) {
	ring := NewHashRing()
	node := &MockNode{ID: "node1"}

	_ = ring.AddNode(node)
	err := ring.AddNode(node)

	if err != ErrNodeExists {
		t.Errorf("Expected duplicate node error, got: %v", err)
	}
}

func TestRemoveNode(t *testing.T) {
	ring := NewHashRing()
	node := &MockNode{ID: "node1"}

	_ = ring.AddNode(node)
	err := ring.RemoveNode(node)
	if err != nil {
		t.Errorf("Error removing node: %v", err)
	}

	if len(ring.nodes) != 0 {
		t.Errorf("Expected empty hash ring after removal, got %d nodes", len(ring.nodes))
	}
}

func TestGetNode(t *testing.T) {
	ring := NewHashRing()
	node1 := &MockNode{ID: "node1"}
	node2 := &MockNode{ID: "node2"}

	_ = ring.AddNode(node1)
	_ = ring.AddNode(node2)

	key := "testKey"
	node, err := ring.GetNode(key)

	if err != nil {
		t.Errorf("Error getting node for key: %v", err)
	}

	if node == nil {
		t.Errorf("Expected node assignment for key: %s", key)
	}
}

func TestEmptyRing(t *testing.T) {
	ring := NewHashRing()

	_, err := ring.GetNode("testKey")
	if err != ErrEmptyRing {
		t.Errorf("Expected empty ring error, got: %v", err)
	}
}
