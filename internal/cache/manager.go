package cache

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/its-saeed/distributed-cache/internal/communication"
	"github.com/its-saeed/distributed-cache/internal/consistent"
)

// Manager manages cache nodes and request distribution
type Manager struct {
	hashRing  *consistent.HashRing
	pubsub    *communication.PubSub
	nodeAddrs map[string]bool
	mu        sync.RWMutex
	logger    *log.Logger
}

func NewCacheManager(pubsub *communication.PubSub) *Manager {
	cm := &Manager{
		pubsub:    pubsub,
		hashRing:  consistent.NewHashRing(),
		nodeAddrs: make(map[string]bool),
		logger:    log.New(os.Stdout, "CACHE-MANAGER: ", log.LstdFlags),
	}

	pubsub.Subscribe("node_register", cm.onNodeRegister)
	pubsub.Subscribe("node_unregister", cm.onNodeUnregister)
	pubsub.Subscribe("node_heartbeat", cm.onNodeHeartbeat)

	return cm
}

func (cm *Manager) onNodeRegister(msg communication.Message) {
	var addr string
	if err := json.Unmarshal(msg.Payload, &addr); err != nil {
		cm.logger.Printf("Failed to unmarshal node registration: %v", err)
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.nodeAddrs[addr] {
		return
	}

	node := &simpleNode{addr: addr}
	if err := cm.hashRing.AddNode(node); err != nil {
		cm.logger.Printf("Failed to add node to hash ring: %v", err)
		return
	}

	cm.nodeAddrs[addr] = true
	cm.logger.Printf("Node registered: %s (Total nodes: %d)", addr, len(cm.nodeAddrs))
}

func (cm *Manager) onNodeUnregister(msg communication.Message) {
	var addr string
	if err := json.Unmarshal(msg.Payload, &addr); err != nil {
		cm.logger.Printf("Failed to unmarshal node unregistration: %v", err)
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.nodeAddrs[addr] {
		return
	}

	node := &simpleNode{addr: addr}
	if err := cm.hashRing.RemoveNode(node); err != nil {
		cm.logger.Printf("Failed to remove node from hash ring: %v", err)
		return
	}

	delete(cm.nodeAddrs, addr)
	cm.logger.Printf("Node unregistered: %s (Remaining nodes: %d)", addr, len(cm.nodeAddrs))
}

func (cm *Manager) onNodeHeartbeat(msg communication.Message) {
	var addr string
	if err := json.Unmarshal(msg.Payload, &addr); err != nil {
		cm.logger.Printf("Failed to unmarshal node heartbeat: %v", err)
		return
	}

	cm.logger.Printf("Node heartbeat: %s", addr)
}

func (cm *Manager) SetKeyOnNode(nodeAddr, key string, value []byte) error {
	cm.logger.Printf("SetKeyOnNode: %s %s", nodeAddr, key)

	respChan := make(chan error, 1)
	defer close(respChan)

	subID := fmt.Sprintf("set_resp_%s", generateRandomID(8))
	var handler communication.MessageHandler
	handler = func(msg communication.Message) {
		var resp struct {
			Error string `json:"error"`
		}

		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			respChan <- err
		} else if resp.Error != "" {
			respChan <- errors.New(resp.Error)
		} else {
			respChan <- nil
		}

		cm.pubsub.Unsubscribe(subID, handler)
	}

	cm.pubsub.Subscribe(subID, handler)

	request := struct {
		Key     string `json:"key"`
		Value   []byte `json:"value"`
		ReplyTo string `json:"reply_to"`
	}{
		Key:     key,
		Value:   value,
		ReplyTo: subID,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal set request: %w", err)
	}

	cm.pubsub.PublishSync(fmt.Sprintf("set_%s", nodeAddr), communication.Message{
		Topic:   fmt.Sprintf("set_%s", nodeAddr),
		Payload: payload,
	})

	select {
	case err := <-respChan:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("set operation timed out")
	}
}

func (cm *Manager) DeleteKeyOnNode(nodeAddr, key string) error {
	cm.logger.Printf("DeleteKeyOnNode: %s %s", nodeAddr, key)

	respChan := make(chan error, 1)
	defer close(respChan)

	subID := fmt.Sprintf("del_resp_%s", generateRandomID(8))
	var handler communication.MessageHandler
	handler = func(msg communication.Message) {
		var resp struct {
			Error string `json:"error"`
		}

		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			respChan <- err
		} else if resp.Error != "" {
			respChan <- errors.New(resp.Error)
		} else {
			respChan <- nil
		}

		cm.pubsub.Unsubscribe(subID, handler)
	}

	cm.pubsub.Subscribe(subID, handler)

	request := struct {
		Key     string `json:"key"`
		ReplyTo string `json:"reply_to"`
	}{
		Key:     key,
		ReplyTo: subID,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal set request: %w", err)
	}

	cm.pubsub.PublishSync(fmt.Sprintf("del_%s", nodeAddr), communication.Message{
		Topic:   fmt.Sprintf("del_%s", nodeAddr),
		Payload: payload,
	})

	select {
	case err := <-respChan:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("delete operation timed out")
	}
}

// GetNodeForKey finds the best node for a given key
func (cm *Manager) GetNodeForKey(key string) (consistent.Node, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.hashRing.GetNode(key)
}

// GetKeyFromNode retrieves a key-value pair from a node
func (cm *Manager) GetKeyFromNode(nodeAddr, key string) ([]byte, bool) {
	respChan := make(chan []byte, 1)
	defer close(respChan)

	subID := fmt.Sprintf("resp_%s", generateRandomID(8))
	handler := func(msg communication.Message) {
		var resp struct {
			Value []byte `json:"value"`
			Found bool   `json:"found"`
		}
		if err := json.Unmarshal(msg.Payload, &resp); err == nil && resp.Found {
			respChan <- resp.Value
		}
	}

	cm.pubsub.Subscribe(subID, handler)

	request := map[string]string{
		"key":      key,
		"reply_to": subID,
	}
	payload, _ := json.Marshal(request)

	cm.pubsub.PublishSync(fmt.Sprintf("get_%s", nodeAddr), communication.Message{
		Topic:   fmt.Sprintf("get_%s", nodeAddr),
		Payload: payload,
	})

	select {
	case value := <-respChan:
		return value, true
	case <-time.After(2 * time.Second):
		return nil, false
	}
}

// simpleNode implements the consistent.Node interface
type simpleNode struct {
	addr string
}

func (n *simpleNode) String() string {
	return n.addr
}

func generateRandomID(size int) string {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
