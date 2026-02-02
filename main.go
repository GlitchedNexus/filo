package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/GlitchedNexus/filo/p2p"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

var (
    filesStored = promauto.NewCounter(prometheus.CounterOpts{
        Name: "filo_files_stored_total",
        Help: "The total number of files stored on this node",
    })
    activePeers = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "filo_active_peers",
        Help: "The number of active peer connections in the cluster",
    })
)

func main() {
    // Port 8081 for metrics (as previously implemented)
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":8081", nil)
    }()

    listenAddr := os.Getenv("LISTEN_ADDR")
    if listenAddr == "" {
        listenAddr = ":3000"
    }

    // Capture bootstrap nodes from env to allow dynamic cluster joining
    bootstrapNodes := strings.Split(os.Getenv("BOOTSTRAP_NODES"), ",")

    s := makeServer(listenAddr, bootstrapNodes...)
    s.Start()
}