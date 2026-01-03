package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/GlitchedNexus/filo/p2p"
)

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

type FileServerOptions struct {
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
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	fmt.Printf("don't have file (%s) locally, fetching from network...\n", key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Second * 3)

	for _, peer := range s.peers {

		fmt.Println("receiving stream from peer:", peer.RemoteAddr())

		fileBuffer := new(bytes.Buffer)

		n, err := io.CopyN(fileBuffer, peer, 22)

		if err != nil {
			return nil, err
		}

		fmt.Printf("received %d bytes over the network", n)

		fmt.Println(fileBuffer.String())
	}

	select {}

	return nil, nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
	// 1. Store the file to disk
	// 2. Broadcast this file to all known peers in the network

	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(key, tee)

	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Second * 3)

	// TODO: (@GlitchedNexus) use a multi-writer here.
	for _, peer := range s.peers {
		n, err := io.Copy(peer, fileBuffer)

		if err != nil {
			return err
		}

		fmt.Println("received and written bytes to disk:", n)
	}

	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

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
	}

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("need to serve file (%s) but it doesn't exist on disk", msg.Key)
	}

	fmt.Printf("serving file (%s) over the network\n", msg.Key)
	r, err := s.store.Read(msg.Key)

	if err != nil {
		return err
	}

	peer, ok := s.peers[from]

	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	n, err := io.Copy(peer, r)

	if err != nil {
		return err
	}

	fmt.Printf("wrote %d bytes over the network to %s\n", n, from)

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]

	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))

	if err != nil {
		return err
	}

	log.Printf("written %d bytes to disk\n", n)

	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {

		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Println("attempting to connect with remote:", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.bootstrapNetwork()

	s.loop()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
