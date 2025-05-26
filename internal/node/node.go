package node

import (
	"encoding/json"
	"log"
	"sync"
	"time"

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
	persist  bool
	dataFile string
}

func NewDataNode(id string, capacity int, pubsub *communication.PubSub, persist bool, dataFile string) *DataNode {
	dn := &DataNode{
		ID:       id,
		pubsub:   pubsub,
		logger:   log.New(log.Writer(), "NODE["+id+"] ", log.LstdFlags),
		shutdown: make(chan struct{}),
		persist:  persist,
		dataFile: dataFile,
	}

	dn.cache = cache.NewLRUCache(capacity)
	dn.LoadFromDisc()

	dn.registerHandlers()

	payload, _ := json.Marshal(id)
	pubsub.PublishSync("node_register", communication.Message{Payload: payload})

	return dn
}

func (dn *DataNode) Run() {
	go dn.startHeartbeat()
}

func (dn *DataNode) startHeartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	dn.wg.Add(1)
	defer dn.wg.Done()
	for {
		select {
		case <-dn.shutdown:
			dn.logger.Println("Stopping heartbeat...")
			return
		case <-ticker.C:
			payload, _ := json.Marshal(dn.ID)
			dn.pubsub.PublishSync("node_heartbeat", communication.Message{Payload: payload})
		}
	}
}

func (dn *DataNode) registerHandlers() {
	dn.pubsub.Subscribe("get_"+dn.ID, dn.handleGetRequest)
	dn.pubsub.Subscribe("set_"+dn.ID, dn.handleSetRequest)
	dn.pubsub.Subscribe("del_"+dn.ID, dn.handleDeleteRequest)
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
		Ttl     int    `json:"ttl"`
		ReplyTo string `json:"reply_to"`
	}

	if err := json.Unmarshal(msg.Payload, &request); err != nil {
		dn.logger.Printf("Error decoding set request: %v", err)
		return
	}

	dn.cache.SetWithTtl(request.Key, request.Value, time.Duration(request.Ttl)*time.Second)

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
	dn.SaveToDisk()
	close(dn.shutdown)
	dn.wg.Wait()
	dn.logger.Println("Shutdown complete")
}

func (dn *DataNode) SaveToDisk() {
	if !dn.persist {
		return
	}

	if err := dn.cache.SaveToFile(dn.dataFile); err != nil {
		dn.logger.Println("Failed to persist")
	}
}

func (dn *DataNode) LoadFromDisc() {
	if !dn.persist {
		return
	}

	if err := dn.cache.LoadFromFile(dn.dataFile); err != nil {
		dn.logger.Println("Failed to persist")
	}
}
