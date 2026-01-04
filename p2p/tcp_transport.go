package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

//
//	============================== TCP PEER ===============================
//

// TCPPeer represents the remote node over an established TCP Connection.
//
// net.Conn:
//   - The underlying connection of the peer which in this case is a
//     TCP connection.
//
// outbound:
//   - If we make a connection with the peer
//     it will be an oubound peer. If we accept the connection
//     form a peer it will be an inbound peer.
//   - If we dial and retrieve a connection => outbound == true
//   - If we accept and retrieve a connection => outbound == false
type TCPPeer struct {
	net.Conn
	outbound bool
	Wg       *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

//
//	============================== TCP TRANSPORT ===============================
//

type TCPTransportOptions struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOptions
	listener net.Listener
	rpcch    chan RPC

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(options TCPTransportOptions) *TCPTransport {
	return &TCPTransport{
		TCPTransportOptions: options,
		rpcch:               make(chan RPC, 1024),
	}
}

// Consume implements the Transport interface, which will return
// a read-only channelfor reading the incoming messages received
// from another peer in the network.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddr)

	return nil
}

// Addr implements the transport interface and returns the address of
// the transport which is accepting connections.
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP accept error %s\n", err)
		}

		fmt.Printf("Incoming connection: %+v\n", conn)

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection: %s\n", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop

	for {
		rpc := RPC{}

		err = t.Decoder.Decode(conn, &rpc)

		if err != nil {
			fmt.Printf("TCP read error: %s\n", err)
			continue
		}

		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.Wg.Add(1)

			fmt.Printf("[%s] incoming stream, waiting ...\n", conn.RemoteAddr())

			peer.Wg.Wait()

			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())

			continue
		}

		t.rpcch <- rpc
	}

}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}
