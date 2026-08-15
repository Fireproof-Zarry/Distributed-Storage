# GoDFS — Decentralized P2P Storage System

A working MVP: files are split into content-addressed chunks (SHA-256),
distributed across multiple independent peer nodes over a custom TCP
protocol, replicated for fault tolerance, and reassembled + verified on
download. No central server in the data path.

## Status: Day 1 + Day 2 MVP complete and verified

**Day 1** — core engine, no networking:
- `internal/chunk` — split a file into fixed-size chunks, hash (SHA-256), reassemble.
- `internal/protocol` — length-prefixed JSON message envelope (STORE/GET/HAVE).
- `internal/node` (`store.go`) — a node's on-disk chunk storage.

**Day 2** — real P2P networking, end-to-end verified:
- `internal/node` (`node.go`) — TCP server: `Listen()`/`Serve()`, one goroutine per connection, dispatches STORE/GET/HAVE.
- `internal/peer` — static peer registry + deterministic peer selection for replica placement.
- `internal/manifest` — tracks a file's ordered chunk hashes so it can be reconstructed.
- `internal/client` — upload/download orchestration: chunk → replicate to peers → manifest; manifest → fetch from peers → verify → reassemble.
- `cmd/godfs` — CLI: `godfs node`, `godfs upload`, `godfs download`.
- `scripts/demo.sh` — full scripted demo, see below.

31 unit + integration tests passing across all packages, including a real
multi-node TCP integration test and a node-failure fault-tolerance test.

## What's verified end-to-end (not just unit tests)

Ran as real separate OS processes (not just in-test goroutines):
1. Started 3 independent node processes on `localhost:9001/9002/9003`.
2. Uploaded a 600KB random file — split into 3 chunks, each replicated onto
   2 of the 3 nodes, confirmed by inspecting each node's actual data
   directory on disk.
3. Downloaded the file back and diffed it against the original —
   byte-for-byte identical.
4. **Killed one node process (`kill -9`) and re-downloaded** — still
   succeeded, still byte-for-byte identical, because replication meant
   every chunk still had a live copy on one of the remaining two nodes.

This is the concrete proof the system is decentralized (no single node is
required) and fault-tolerant (survives a node loss), not just "networked."

## Running it yourself

```bash
go build -o godfs ./cmd/godfs

# Terminal 1-3: start 3 nodes
./godfs node -addr localhost:9001 -data ./data/node1
./godfs node -addr localhost:9002 -data ./data/node2
./godfs node -addr localhost:9003 -data ./data/node3

# Terminal 4: upload and download
./godfs upload   -file ./somefile.bin -peers ./config/peers.json -manifest ./manifest.json
./godfs download -manifest ./manifest.json -peers ./config/peers.json -out ./downloaded.bin
diff somefile.bin downloaded.bin   # should be identical
```

Or just run the whole thing (start nodes, upload, download, kill a node,
re-download, verify) with one command:

```bash
./scripts/demo.sh
```

## Running tests

```bash
go test ./... -v
```

## Explicit scope / honesty notes

This is an MVP, deliberately scoped to be small and *fully working* rather
than large and partially working:

- **Peer discovery is static** (`config/peers.json`), not a real DHT
  (Kademlia). Nodes must know each other's addresses in advance.
- **Replication is upfront only** (store on N peers at upload time) — there
  is no background process detecting a dead replica and re-replicating it.
- **No encryption** — chunks are stored and transferred as plaintext.
- **`localhost` only** — no NAT traversal or real multi-machine networking
  tested yet, though the TCP layer itself would work across machines with
  reachable addresses.

See planned extensions below for what's next.

## Planned extensions

- Kademlia-style DHT for real peer discovery.
- Background replication healing (detect under-replicated chunks, repair).
- Merkle DAG file representation for whole-file verification from one root hash.
- Encryption at rest and in transit.
- Multi-machine deployment (currently `localhost`-only, tested).
- Benchmarking suite (upload/download throughput, chunk retrieval latency).
