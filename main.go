package main

import (
	"log"

	"github.com/GlitchedNexus/filo/p2p"
)

func main() {
	tr := p2p.NewTCPTransport(":3000")

	if err := tr.ListAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
