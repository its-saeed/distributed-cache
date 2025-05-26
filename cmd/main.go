package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/its-saeed/distributed-cache/internal/api"
	"github.com/its-saeed/distributed-cache/internal/cache"
	"github.com/its-saeed/distributed-cache/internal/communication"
	"github.com/its-saeed/distributed-cache/internal/node"
)

func main() {
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	cacheSize := flag.Int("cache-size", 1000, "Cache size")
	port := flag.String("port", "8080", "API server port")
	flag.Parse()

	pubsub := communication.NewPubSub(*debugMode)
	manager := cache.NewCacheManager(pubsub)

	apiServer := api.NewServer(":"+*port, manager)
	log.Println("Starting API server on :8080")

	ctx, cancel := context.WithCancel(context.Background())

	node1 := node.NewDataNode("node-1", *cacheSize, pubsub, true, "node1.json")
	node2 := node.NewDataNode("node-2", *cacheSize, pubsub, true, "node2.json")
	node3 := node.NewDataNode("node-3", *cacheSize, pubsub, true, "node3.json")

	go node1.Run()
	go node2.Run()
	go node3.Run()

	go apiServer.Run(ctx)

	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs // Wait for termination signal
	log.Println("Shutting down system...")

	// Cancel context to signal all goroutines to stop
	cancel()

	node1.Shutdown()
	node2.Shutdown()
	node3.Shutdown()

	// Shutdown API server gracefully
	apiServer.Shutdown()

	log.Println("Shutdown complete.")
}
