# memdb
![Coverage](https://img.shields.io/badge/Coverage-63.7%25-yellow)

![ci](https://github.com/Mort4lis/memdb/actions/workflows/release.yaml/badge.svg)
[![semantic-release: angular](https://img.shields.io/badge/semantic--release-angular-e10079?logo=semantic-release)](https://github.com/semantic-release/semantic-release)
![license](https://img.shields.io/github/license/Mort4lis/memdb)
![go-version](https://img.shields.io/github/go-mod/go-version/Mort4lis/memdb)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mort4lis/memdb)](https://goreportcard.com/report/github.com/Mort4lis/memdb)

## 📖 Description

> **⚠️ Education Purpose Only:**  
> This project is intended for educational purposes and **should not** be used in production systems.

memdb is a simple in-memory key-value database designed for practicing software architecture, network programming, and concurrency in Go. 
The project includes both a server and client implementation communicating over TCP.

## 🚀 Features

* ✅ In-memory key-value database backend
* ✅ TCP server for handling multiple client connections
* ✅ WAL (Write-ahead logging)
* ✅ Segment-based master-slave replication
* ✅ CLI-based server and client applications
* ✅ Graceful shutdown and robust error handling
* ✅ Example configuration and easy customization

## 📈 How to use

### Standalone mode

There are several examples how to run `memdb` server and client using Docker.

```bash
# Run the server:
docker run -d \
  -v $(PWD)/data:/memdb/data \
  -p 7991:7991 \
  --name memdb mortalis/memdb:master
 
# Clint usage
docker run -i --rm \
  --name memdb-cli \
  --network=host \
  mortalis/memdb:master /memdb/memdb-cli --help
  
# Connect to server using the client
docker run -i --rm \
  --name memdb-cli \
  --network=host \
  mortalis/memdb:master /memdb/memdb-cli --address 127.0.0.1:7991
```

### Cluster mode

If you want to run `memdb` in cluster mode, you can do so with the following `docker-compose` file.

```yaml
services:
  master:
    image: mortalis/memdb:master
    container_name: "memdb_master"
    environment:
      REPLICA_TYPE: master
      REPLICA_MASTER_ADDR: :3232
      LOG_LEVEL: debug
    ports:
      - "7991:7991"
      - "3232:3232"
    volumes:
      - ./data:/memdb/data
    restart: on-failure
    networks:
      - memdb

  slave_1:
    image: mortalis/memdb:master
    container_name: "memdb_slave-1"
    environment:
      REPLICA_TYPE: slave
      REPLICA_MASTER_ADDR: master:3232
      LOG_LEVEL: debug
    ports:
      - "7992:7991"
    volumes:
      - ./data:/memdb/data
    depends_on:
      - master
    restart: on-failure
    networks:
      - memdb

networks:
  memdb:
    driver: bridge
```

## 🔧Build from source code

```bash
git clone git@github.com:Mort4lis/memdb.git && cd memdb
make build

# Server usage
./build/memdb --help
# Client usage
./build/memdb-cli --help
```

## ⚙️ Configuration

`memdb` is simply configured using YAML file with the optional section and environment variable (see the [example](./config.yaml)).
The volume `/memdb/config.yaml` is supported to override the default configuration.
