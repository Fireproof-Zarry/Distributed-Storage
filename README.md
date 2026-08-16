# GoDFS — Decentralized P2P Storage System

GoDFS is a peer-to-peer distributed file storage system written in Go. Files are split into fixed-size, content-addressed chunks using SHA-256 and distributed across independent storage nodes over a custom TCP protocol. Chunks are replicated across peers and reconstructed and verified during download.

There is no central server in the data path.

## Features

* Content-addressed file chunking using SHA-256
* Custom length-prefixed JSON/TCP protocol
* P2P storage and retrieval with `STORE`, `GET`, and `HAVE` operations
* Deterministic replica placement across peers
* File manifests for ordered chunk reconstruction
* Replicated storage for node-failure tolerance
* CLI for starting nodes, uploading files, and downloading files
* End-to-end multi-node integration tests

## Architecture

The system is split into a few core components:

```text
                    ┌──────────────┐
                    │   godfs CLI  │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │    Client    │
                    └──────┬───────┘
                           │
              ┌────────────▼────────────┐
              │      Peer Registry      │
              │  Replica Selection      │
              └────────────┬────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
    │   Node 1  │    │   Node 2  │    │   Node 3  │
    │  TCP      │    │  TCP      │    │  TCP      │
    │  Storage  │    │  Storage  │    │  Storage  │
    └───────────┘    └───────────┘    └───────────┘
```

### Core packages

* `internal/chunk` — file chunking, SHA-256 hashing, and reassembly
* `internal/protocol` — length-prefixed JSON message format
* `internal/node` — TCP server and on-disk chunk storage
* `internal/peer` — peer registry and deterministic replica selection
* `internal/manifest` — ordered chunk metadata for file reconstruction
* `internal/client` — upload and download orchestration
* `cmd/godfs` — command-line interface

## Fault Tolerance

Files are replicated across multiple storage nodes. During download, the client retrieves chunks from available replicas and verifies the reconstructed file.

The current implementation has been tested with three independent node processes:

1. A file is split into chunks and replicated across the nodes.
2. The file is downloaded and verified byte-for-byte against the original.
3. One node is terminated.
4. The file is downloaded again using the remaining replicas.

The download continues to succeed after the node failure because each chunk has a replica on another node.

## Running GoDFS

Build the CLI:

```bash
go build -o godfs ./cmd/godfs
```

Start three storage nodes:

```bash
./godfs node -addr localhost:9001 -data ./data/node1
./godfs node -addr localhost:9002 -data ./data/node2
./godfs node -addr localhost:9003 -data ./data/node3
```

Upload a file:

```bash
./godfs upload \
  -file ./somefile.bin \
  -peers ./config/peers.json \
  -manifest ./manifest.json
```

Download it:

```bash
./godfs download \
  -manifest ./manifest.json \
  -peers ./config/peers.json \
  -out ./downloaded.bin
```

Verify the result:

```bash
diff somefile.bin downloaded.bin
```

A complete multi-node demonstration is also available:

```bash
./scripts/demo.sh
```

The demo starts the nodes, uploads and downloads a file, terminates a node, and verifies that the file can still be recovered from the remaining replicas.

## Tests

Run the full test suite with:

```bash
go test ./... -v
```

The test suite includes unit tests, multi-node TCP integration tests, and node-failure scenarios.

## Current Limitations

GoDFS is currently an MVP and intentionally keeps the system simple.

* **Static peer discovery:** peers are configured through `config/peers.json`; there is no DHT yet.
* **No replica healing:** replication occurs during upload, but there is no background process to detect failed replicas and restore the desired replication factor.
* **No encryption:** chunks are currently transferred and stored without encryption.
* **Local testing:** the current setup has been tested using multiple nodes on `localhost`. Multi-machine deployment has not yet been validated.

## Roadmap

* [ ] Kademlia-style DHT for peer discovery
* [ ] Background replica healing
* [ ] Merkle DAG-based file representation
* [ ] Encryption in transit and at rest
* [ ] Multi-machine deployment
* [ ] Storage and retrieval benchmarking
* [ ] Improved consistency and failure handling
* [ ] Additional CLI and operational tooling
