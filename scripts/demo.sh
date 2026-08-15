#!/usr/bin/env bash
# GoDFS demo script.
#
# Starts 3 local P2P nodes, uploads a file (which gets chunked, hashed, and
# replicated across the nodes), downloads it back and verifies it's a
# byte-for-byte match, then kills one node and re-downloads to prove
# replication provides fault tolerance.
#
# Run from the repo root: ./scripts/demo.sh

set -euo pipefail

cd "$(dirname "$0")/.."

echo "== Building godfs =="
go build -o godfs ./cmd/godfs

WORKDIR=$(mktemp -d)
trap 'kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null; rm -rf "$WORKDIR" ./data' EXIT

echo "== Creating a sample file to upload =="
head -c 600000 /dev/urandom > "$WORKDIR/sample.bin"
ls -la "$WORKDIR/sample.bin"

echo
echo "== Starting 3 nodes =="
./godfs node -addr localhost:9001 -data ./data/node1 > "$WORKDIR/node1.log" 2>&1 &
NODE1_PID=$!
./godfs node -addr localhost:9002 -data ./data/node2 > "$WORKDIR/node2.log" 2>&1 &
NODE2_PID=$!
./godfs node -addr localhost:9003 -data ./data/node3 > "$WORKDIR/node3.log" 2>&1 &
NODE3_PID=$!
sleep 1
echo "node1 pid=$NODE1_PID, node2 pid=$NODE2_PID, node3 pid=$NODE3_PID"

echo
echo "== Uploading sample.bin (splits into chunks, replicates across nodes) =="
./godfs upload -file "$WORKDIR/sample.bin" -peers ./config/peers.json -manifest "$WORKDIR/manifest.json"

echo
echo "== Chunk distribution across nodes (proof the file was actually scattered) =="
echo "node1: $(ls data/node1 | wc -l) chunk(s)"
echo "node2: $(ls data/node2 | wc -l) chunk(s)"
echo "node3: $(ls data/node3 | wc -l) chunk(s)"

echo
echo "== Downloading and verifying against original =="
./godfs download -manifest "$WORKDIR/manifest.json" -peers ./config/peers.json -out "$WORKDIR/downloaded.bin"
if diff -q "$WORKDIR/sample.bin" "$WORKDIR/downloaded.bin" > /dev/null; then
  echo "MATCH: downloaded file is byte-for-byte identical to the original."
else
  echo "MISMATCH: downloaded file differs from the original!"
  exit 1
fi

echo
echo "== Killing node1 to simulate a node going offline =="
kill -9 "$NODE1_PID"
sleep 1

echo
echo "== Re-downloading with node1 dead (proves replication fault tolerance) =="
./godfs download -manifest "$WORKDIR/manifest.json" -peers ./config/peers.json -out "$WORKDIR/downloaded_after_kill.bin"
if diff -q "$WORKDIR/sample.bin" "$WORKDIR/downloaded_after_kill.bin" > /dev/null; then
  echo "MATCH: download still succeeds and is byte-for-byte identical, even with a node down."
else
  echo "MISMATCH: download failed to tolerate node loss!"
  exit 1
fi

echo
echo "== Demo complete. =="
