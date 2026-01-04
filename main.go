package main

import (
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

	// data := bytes.NewReader([]byte("my big data file here!"))

	// s2.Store("coolPicture.jpg", data)

	// time.Sleep(time.Millisecond * 5)

	r, err := s2.Get("coolPicture.jpg")

	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(r)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

}
