package p2p

// RPC holds any arbitrary data that is being sent over
// each transport between two nodes.
type RPC struct {
	From    string
	Payload []byte
}
