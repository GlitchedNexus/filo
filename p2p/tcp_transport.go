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

type TCPTransport struct {
	listenAddress string
	listener      net.Listener

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
func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenAddr,
	}
}

func (t *TCPTransport) ListAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.listenAddress)

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
			fmt.Printf("TCP Accept Error: %s\n", err)
		}

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {

	peer := NewTCPPeer(conn, true)

	fmt.Printf("New incoming connection %+v\n", peer)
}
