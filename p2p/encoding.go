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
	buf := make([]byte, 1028)
	n, err := r.Read(buf)

	if err != nil {
		return nil
	}

	msg.Payload = buf[:n]

	return nil
}
