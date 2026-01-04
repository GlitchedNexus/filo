package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {

	// Initial buffer implementation used
	// buf := make([]byte, 1028)
	// The problem with this is that file can be larger than
	// 1Kb and upping buffer size consumes memory so we
	// will stream data instead to circumevent this problem.

	peekBuf := make([]byte, 1)

	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	stream := peekBuf[0] == IncomingStream

	// In case of a stream, we are not decoding what is being
	// sent over the network.
	//
	// We are just setting stream = true so that we can handle
	// it in our logic.
	if stream {
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 1028)
	n, err := r.Read(buf)

	if err != nil {
		return nil
	}

	msg.Payload = buf[:n]

	return nil
}
