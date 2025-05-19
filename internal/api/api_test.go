package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/its-saeed/distributed-cache/internal/cache"
	"github.com/its-saeed/distributed-cache/internal/communication"
	"github.com/its-saeed/distributed-cache/internal/node"
	"github.com/stretchr/testify/assert"
)

func TestServerInitialization(t *testing.T) {
	pubsub := communication.NewPubSub(false)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handleRoot(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Equal(t, "Distributed Cache API", response["message"])
}

func TestSetKeyEndpoint(t *testing.T) {
	pubsub := communication.NewPubSub(true)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	// Simulate node registration
	cacheSize := 1000
	node1 := node.NewDataNode("node-1", cacheSize, pubsub)
	node1.Run()

	req := httptest.NewRequest(http.MethodPost, "/set?key=testKey&value=testValue", nil)
	rec := httptest.NewRecorder()
	server.handleSet(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestGetKeyEndpoint(t *testing.T) {
	pubsub := communication.NewPubSub(true)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	// Simulate node registration
	cacheSize := 1000
	nodeAddr := "node-1"
	node1 := node.NewDataNode(nodeAddr, cacheSize, pubsub)
	node1.Run()

	// Set a key manually
	cacheManager.SetKeyOnNode(nodeAddr, "testKey", []byte("testValue"))

	req := httptest.NewRequest(http.MethodGet, "/get?key=testKey", nil)
	rec := httptest.NewRecorder()
	server.handleGet(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	assert.Equal(t, "testKey", response["key"])
	assert.Equal(t, "testValue", response["value"])
	assert.Equal(t, nodeAddr, response["node"])
}

func TestGetNonExistentKey(t *testing.T) {
	pubsub := communication.NewPubSub(true)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	// Simulate node registration
	cacheSize := 1000
	nodeAddr := "node-1"
	node1 := node.NewDataNode(nodeAddr, cacheSize, pubsub)
	node1.Run()

	req := httptest.NewRequest(http.MethodGet, "/get?key=missingKey", nil)
	rec := httptest.NewRecorder()
	server.handleGet(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestInvalidMethodGetEndpoint(t *testing.T) {
	pubsub := communication.NewPubSub(true)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	req := httptest.NewRequest(http.MethodPost, "/get?key=testKey", nil) // Invalid method
	rec := httptest.NewRecorder()
	server.handleGet(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestInvalidMethodSetEndpoint(t *testing.T) {
	pubsub := communication.NewPubSub(true)
	cacheManager := cache.NewCacheManager(pubsub)
	server := NewServer(":8080", cacheManager)

	req := httptest.NewRequest(http.MethodGet, "/set?key=testKey&value=testValue", nil) // Invalid method
	rec := httptest.NewRecorder()
	server.handleSet(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
