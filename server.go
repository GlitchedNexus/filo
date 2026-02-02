package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GlitchedNexus/filo/p2p"
)

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
	ID  string
}

type MessageDeleteFile struct {
	Key string
	ID  string
}

type FileServerOptions struct {
	ID                string
	EncKey            []byte
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOptions

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(options FileServerOptions) *FileServer {

	storeOptions := StoreOptions{
		Root:              options.StorageRoot,
		PathTransformFunc: options.PathTransformFunc,
	}

	if len(options.ID) == 0 {
		options.ID = generateID()
	}

	return &FileServer{
		FileServerOptions: options,
		store:             NewStore(storeOptions),
		quitch:            make(chan struct{}),
		peers:             make(map[string]p2p.Peer),
	}
}

func (s *FileServer) stream(msg *Message) error {

	// We collect peers as io.Writer because:
	// - TCPPeer embeds net.Conn
	// - net.Conn implements io.Writer (Write([]byte) (int, error))
	// - Embedded methods are promoted, so *TCPPeer has a Write method
	// - Therefore *TCPPeer implicitly satisfies io.Writer
	//
	// This allows us to treat peers as generic writers and broadcast
	// the same byte stream to all of them.
	peers := []io.Writer{}

	for _, peer := range s.peers {
		// Each peer is a *TCPPeer, which is valid as an io.Writer
		// as long as peer.Conn is non-nil.
		peers = append(peers, peer)
	}

	// io.MultiWriter returns a Writer that duplicates each Write
	// call to all provided writers sequentially.
	//
	// NOTE:
	// - MultiWriter does NOT add framing or synchronization.
	// - If multiple goroutines call broadcast concurrently,
	//   writes to the same underlying net.Conn may interleave.
	// - In production, consider per-peer write goroutines or
	//   message framing (length-prefix, etc.).
	mw := io.MultiWriter(peers...)

	// gob.NewEncoder writes encoded data to the provided io.Writer.
	// Since mw is a MultiWriter, the encoded payload is sent to
	// all peers simultaneously.
	//
	// Encode blocks until all writes complete or an error occurs.
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) broadcast(msg *Message) error {

	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})

		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk.\n", s.Transport.Addr(), key)

		_, r, err := s.store.Read(s.ID, key)
		return r, err
	}

	fmt.Printf("[%s] don't have file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range s.peers {

		// First read the file size so we can limit the amount of bytes that we read
		// from the connection so it will no keep hanging.

		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))

		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)", s.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := s.store.Read(s.ID, key)

	return r, err
}

func (s *FileServer) Delete(key string) error {
	if !s.store.Has(s.ID, key) {
		return fmt.Errorf("[%s] desired file (%s) doesn't exist.\n", s.Transport.Addr(), key)
	}

	msg := Message{
		Payload: MessageDeleteFile{
			Key: hashKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 500)

	if err := s.store.Delete(s.ID, key); err != nil {
		return err
	}

	return nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
	// 1. Store the file to disk
	// 2. Broadcast this file to all known peers in the network

	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(s.ID, key, tee)

	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			ID:   s.ID,
			Key:  hashKey(key),
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 500)

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)

	time.Sleep(time.Millisecond * 500)

	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written (%d) bytes to disk\n", s.Transport.Addr(), n)

	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	activePeers.Inc()

	log.Printf("connected with remote %s", p.RemoteAddr())

	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to error or user quit action.")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error:", err)
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("handle message error:", err)
			}

		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	case MessageDeleteFile:
		return s.handleMessageDeleteFile(from, v)
	}

	return nil
}

func (s *FileServer) handleMessageDeleteFile(from string, msg MessageDeleteFile) error {

	_, ok := s.peers[from]

	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] desired file (%s) doesn't exist.\n", s.Transport.Addr(), msg.Key)
	}

	if err := s.store.Delete(msg.ID, msg.Key); err != nil {
		return err
	}

	filesStored.Desc()

	fmt.Printf("[%s] File Deleted...", s.Transport.Addr())

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] need to serve file (%s) but it doesn't exist on disk", s.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file (%s) over the network\n", s.Transport.Addr(), msg.Key)

	fileSize, r, err := s.store.Read(msg.ID, msg.Key)

	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing readCloser")
		defer rc.Close()
	}

	peer, ok := s.peers[from]

	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	peer.Send([]byte{p2p.IncomingStream})

	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)

	if err != nil {
		return err
	}

	fmt.Printf("[%s] wrote (%d) bytes over the network to %s\n", s.Transport.Addr(), n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]

	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))

	if err != nil {
		return err
	}

	log.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	peer.CloseStream()

	filesStored.Inc()

	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {

		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s] attempting to connect with remote: %s", s.Transport.Addr(), addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}

			fmt.Printf("[%s] connected with remote: %s", s.Transport.Addr(), addr)
		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {
	fmt.Printf("[%s] starting fileserver...\n", s.Transport.Addr())
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	fmt.Printf("[%s] server started trying to bootstrap\n", s.Transport.Addr())

	s.bootstrapNetwork()

	fmt.Printf("[%s] starting server loop\n", s.Transport.Addr())

	s.loop()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessageDeleteFile{})
}

func (s *FileServer) bootstrapMigration(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            s.scanAndMigrate()
        }
    }()
}

func (s *FileServer) scanAndMigrate() {
    // Walk the storage root
    filepath.Walk(s.store.Root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() && time.Since(info.ModTime()) > 5 * time.Minute {
            // Logic to derive key from path and call s.store.MoveToS3(key)
        }
        return nil
    })
}