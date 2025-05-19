package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/its-saeed/distributed-cache/internal/api"
	"github.com/its-saeed/distributed-cache/internal/cache"
	"github.com/its-saeed/distributed-cache/internal/communication"
	"github.com/its-saeed/distributed-cache/internal/node"
)

func main() {
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	cacheSize := flag.Int("cache-size", 1000, "Cache size")
	flag.Parse()

	pubsub := communication.NewPubSub(*debugMode)
	manager := cache.NewCacheManager(pubsub)

	apiServer := api.NewServer(":8080", manager)
	log.Println("Starting API server on :8080")

	node1 := node.NewDataNode("node-1", *cacheSize, pubsub)
	node2 := node.NewDataNode("node-2", *cacheSize, pubsub)
	node3 := node.NewDataNode("node-3", *cacheSize, pubsub)

	node1.Run()
	node2.Run()
	node3.Run()

	if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("API server error: %v", err)
	}
}
