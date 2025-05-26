package node

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/its-saeed/distributed-cache/internal/cache"
	"github.com/its-saeed/distributed-cache/internal/communication"
)

type DataNode struct {
	ID       string
	cache    cache.Cache
	pubsub   *communication.PubSub
	logger   *log.Logger
	shutdown chan struct{}
	wg       sync.WaitGroup
}

func NewDataNode(id string, capacity int, pubsub *communication.PubSub) *DataNode {
	dn := &DataNode{
		ID:       id,
		pubsub:   pubsub,
		logger:   log.New(log.Writer(), "NODE["+id+"] ", log.LstdFlags),
		shutdown: make(chan struct{}),
	}

	dn.cache = cache.NewLRUCache(capacity)

	dn.registerHandlers()

	payload, _ := json.Marshal(id)
	pubsub.PublishSync("node_register", communication.Message{Payload: payload})

	return dn
}

func (dn *DataNode) Run() {
	dn.startWorkers()
}

func (dn *DataNode) registerHandlers() {
	dn.pubsub.Subscribe("get_"+dn.ID, dn.handleGetRequest)
	dn.pubsub.Subscribe("set_"+dn.ID, dn.handleSetRequest)
}

func (dn *DataNode) handleGetRequest(msg communication.Message) {
	var request struct {
		Key     string `json:"key"`
		ReplyTo string `json:"reply_to"`
	}

	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		dn.logger.Printf("Error decoding get request: %v", err)
		return
	}

	value, found := dn.cache.Get(request.Key)

	response := struct {
		Key   string `json:"key"`
		Value []byte `json:"value,omitempty"`
		Found bool   `json:"found"`
	}{
		Key:   request.Key,
		Value: value,
		Found: found,
	}

	payload, _ := json.Marshal(response)
	dn.pubsub.PublishSync(request.ReplyTo, communication.Message{
		Topic:   request.ReplyTo,
		Payload: payload,
		Sender:  dn.ID,
	})
}

func (dn *DataNode) handleSetRequest(msg communication.Message) {
	var request struct {
		Key     string `json:"key"`
		Value   []byte `json:"value"`
		ReplyTo string `json:"reply_to"`
	}

	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		dn.logger.Printf("Error decoding set request: %v", err)
		return
	}

	dn.cache.Set(request.Key, request.Value)

	response := struct {
		Error string `json:"error"`
	}{
		Error: "",
	}

	payload, _ := json.Marshal(response)
	dn.pubsub.PublishSync(request.ReplyTo, communication.Message{
		Topic:   request.ReplyTo,
		Payload: payload,
		Sender:  dn.ID,
	})
}

func (dn *DataNode) handleDeleteRequest(msg communication.Message) {
	var request struct {
		Key     string `json:"key"`
		ReplyTo string `json:"reply_to"`
	}

	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		dn.logger.Printf("Error decoding set request: %v", err)
		return
	}

	dn.cache.Delete(request.Key)

	response := struct {
		Error string `json:"error"`
	}{
		Error: "",
	}

	payload, _ := json.Marshal(response)
	dn.pubsub.PublishSync(request.ReplyTo, communication.Message{
		Topic:   request.ReplyTo,
		Payload: payload,
		Sender:  dn.ID,
	})
}

func (dn *DataNode) Shutdown() {
	close(dn.shutdown)
	dn.wg.Wait()
	dn.logger.Println("Shutdown complete")
}
