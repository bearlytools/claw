package msgs

// Cancel is a message sent to cancel a Syncronous RPC. It does not work for any of the other RPC types.
type Cancel struct {
	// StreamID is the ID of the stream that is closing.
	StreamID uint32
	// ReqID is the ID of the request that is being cancelled.
	ReqID uint32
}
