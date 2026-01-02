package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over an established TCP Connection.
//
// conn:
//   - Conn is the underlying connection of the peer
//
// outbound:
//   - If we make a connection with the peer
//     it will be an oubound peer. If we accept the connection
//     form a peer it will be an inbound peer.
//   - If we dial and retrieve a connection => outbound == true
//   - If we accept and retrieve a connection => outbound == false
type TCPPeer struct {
	conn     net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransportOptions struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOptions
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

// If we used the interface type (Transport) as a return value
// we will encounter the problem of having to type cast
// return value to a TCPTransport to get access to the
// internal fields, i.e.,
//
//	func Test() {
//	    tcp := NewTCPTransport().(*TCPTransport)
//
//	    // We cannot call this without the type cast.
//		tcp.listener.Accept()
//	}
func NewTCPTransport(options TCPTransportOptions) *TCPTransport {
	return &TCPTransport{
		TCPTransportOptions: options,
	}
}

func (t *TCPTransport) ListAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()

		if err != nil {
			fmt.Printf("TCP accept error %s\n", err)
		}

		fmt.Printf("Incoming connection: %+v\n", conn)

		go t.handleConn(conn)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {

	peer := NewTCPPeer(conn, true)

	if err := t.HandshakeFunc(peer); err != nil {
		conn.Close()

		fmt.Printf("TCP handshake error %s\n", err)

		return
	}

	// Read loop
	msg := &Temp{}
	for {
		if err := t.Decoder.Decode(conn, msg); err != nil {
			fmt.Printf("TCP error: %s\n", err)
			continue
		}
	}
}
