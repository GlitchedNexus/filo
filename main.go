package main

import (
	"log"

	"github.com/GlitchedNexus/filo/p2p"
)

func main() {
	tcpOpts := p2p.TCPTransportOptions{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	if err := tr.ListAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
