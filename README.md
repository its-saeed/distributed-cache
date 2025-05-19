# 🚀 Distributed Cache System

## 📖 Overview

Distributed Cache is a scalable caching solution that distributes data across multiple nodes using consistent hashing. It ensures efficient storage, fast retrieval, and fault tolerance, making it ideal for high-performance distributed applications.

## ✨ Features

- ✅ Consistent Hashing → Efficient load distribution among nodes
- ✅ Pub/Sub Communication → Nodes synchronize using a publish-subscribe system
- ✅ API Support → RESTful API for managing cache operations (GET, SET)
- ✅ Fault-Tolerant & Scalable → Nodes can dynamically join or leave the cache ring

## 📌 Project Structure

```plain
distributed-cache/
│── internal/
│   ├── node/          # Handles individual cache nodes
│   ├── cache/         # Implements caching 
│   ├── communication/ # Manages Pub/Sub messaging
│   ├── consistent/    # Implements consistent hashing
│── api/               # REST API server
│── cmd/               # Entry point for running the application
│── README.md          # Project documentation
```

## 🔧 Installation

### Prerequisites

✅ Go 1.18+ → Install from golang.org

### Steps
```bash
git clone https://github.com/its-saeed/distributed-cache.git
cd distributed-cache
go mod tidy
go build -o distributed-cache cmd/main.go
```


## 🚀 Running the Project
Start the API server:

```bash
./distributed-cache --port=8080
```
It starts the API server on port 8080. It also starts three caching nodes, `node1`, `node2`, and `node3`, which are responsible for storing and retrieving data.

## 🔗 API Usage
### 📥 Set Key

```bash
curl -X POST "http://localhost:8080/set?key=foo&value=bar"
```

Response:
```json
{
    "message": "Value set successfully",
    "key": "foo",
    "node": "node-1"
}
```


### 🔎 Get Key

```bash
curl -X GET "http://localhost:8080/get?key=foo"
```


Response:
```json
{
    "key": "foo",
    "value": "bar",
    "node": "node-1"
}
```


## 🧪 Running Tests

```bash
go test -v ./internal/... 
```

