# Filo - A Lightweight Distributed File System (Go)

## Overview

This project is a minimal peer-to-peer file storage prototype written in Go. Each node runs a `FileServer` that:

- Stores files on local disk using a content-addressable path scheme.
- Connects to peers over TCP and exchanges messages for storing and fetching files.
- Streams file data between peers, optionally encrypted in transit.

The `main.go` entrypoint is a runnable demo that spins up three nodes, stores files on one node, deletes local copies, and then fetches them back over the network.

## Architecture

### Core packages and files

- `main.go`: Demo harness that creates three servers and exercises store/get.
- `server.go`: `FileServer` implementation, peer management, message handling, and network I/O.
- `store.go`: Local disk storage with configurable path transform (default is CAS).
- `crypto.go`: ID generation, key hashing, and AES-CTR stream encryption helpers.
- `p2p/`: Minimal transport layer and wire framing for messages and streams.

### Key components

- **FileServer**
  - Owns a local `Store`, a `p2p.Transport`, and a peer map.
  - Broadcasts metadata messages (store/get) to peers.
  - Streams file contents over TCP when needed.
- **Store**
  - Writes and reads file contents under `StorageRoot/<nodeID>/<path>`.
  - Uses `CASPathTransformFunc` by default, which SHA1-hashes keys into sharded folders.
- **Transport (p2p)**
  - `TCPTransport` accepts and dials TCP connections.
  - `DefaultDecoder` distinguishes between framed messages and raw streams.
  - `TCPPeer` wraps a `net.Conn` and supports a "stream pause" mechanism.

### Storage layout

Files are stored under a per-node namespace:

```
<StorageRoot>/<nodeID>/<sha1-sharded-path>/<sha1-hash>
```

The path sharding is derived from `CASPathTransformFunc`, which splits the SHA1 hash into fixed-size directory segments.

## How interactions work

### Startup and peer discovery

1. A `FileServer` is created with a `TCPTransport` and a list of `BootstrapNodes`.
2. `Start()` begins listening, then dials bootstrap peers.
3. When a connection is established, `OnPeer` registers the peer in the server’s map.

### Store flow (broadcast + stream)

1. `Store(key, reader)` writes the file locally under the node’s ID.
2. The server broadcasts `MessageStoreFile` with:
   - `ID`: the sender’s node ID.
   - `Key`: `hashKey(key)` (MD5).
   - `Size`: encrypted payload size (includes IV).
3. The sender streams encrypted bytes to all peers.
4. Each peer receives a stream and writes raw bytes to disk under `<senderID>/<hashedKey>`.

### Get flow (request + stream)

1. `Get(key)` checks local disk. If present, it reads locally.
2. If missing, the node broadcasts `MessageGetFile` with:
   - `ID`: the requester’s node ID.
   - `Key`: `hashKey(key)` (MD5).
3. A peer that has the file under that ID streams it back.
4. The requester decrypts the stream and writes a local copy.

### Wire protocol details

- **Message frames**: `IncomingMessage` byte prefix + gob-encoded payload.
- **Streams**: `IncomingStream` byte prefix, followed by a file size (int64), then raw bytes.
- **Stream gating**: `TCPPeer` pauses its read loop while a stream is in progress.

## Notes and assumptions

- The current flow assumes the original uploader’s node ID is used as the storage namespace on peers.
- Files are encrypted in transit using AES-CTR; the IV is prepended to the stream.

## Build and run

```
make build
./bin/fs
```

Tests:

```
make test
```
