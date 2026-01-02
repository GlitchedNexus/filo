package main

import (
	"log"
	"time"

	"github.com/GlitchedNexus/filo/p2p"
)

func main() {

	tcpTransportOptions := p2p.TCPTransportOptions{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		// TODO: onPeer Func
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOptions)

	fileServerOptions := FileServerOptions{
		StorageRoot:       "store",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
	}

	s := NewFileServer(fileServerOptions)

	go func() {
		time.Sleep((time.Second * 3))
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
