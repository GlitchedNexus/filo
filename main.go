package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/GlitchedNexus/filo/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOptions := p2p.TCPTransportOptions{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOptions)

	fileServerOptions := FileServerOptions{
		EncKey:            newEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOptions)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(time.Second * 2)

	go s2.Start()

	time.Sleep(time.Second * 2)

	for i := range 20 {

		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("my big data file here!"))

		s2.Store(key, data)

		if err := s2.store.Delete(key); err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Millisecond * 5)

		r, err := s2.Get(key)

		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}
}
