package msgs

import (
	"context"
	"fmt"
	"io"

	"github.com/bearlytools/claw/internal/binary"
)

// Payload is a payload message.
type Payload struct {
	// StreamID is the ID of the stream on this connection.
	StreamID uint32
	// Words is the number of 64 words in the Payload. This can be 64 bit words because
	// all payloads are aligned to 8 bytes.
	words uint16
	// ReqID is the ID of the request. If this is not Syncronous, this is 0.
	ReqID uint32
	// Payload is the payload of a request.
	Payload []byte
}

// Validate validates the Payload message.
func (c *Payload) Validate(ctx context.Context) error {
	if c.StreamID == 0 {
		return fmt.Errorf("invalid StreamID: cannot be 0")
	}
	if c.ReqID == 0 {
		return fmt.Errorf("invalid ReqID: cannot be 0")
	}
	if len(c.Payload)%8 != 0 {
		return fmt.Errorf("invalid Payload: length must be multiple of 8")
	}
	c.words = uint16(len(c.Payload) / 8)
	return nil
}

// MarshalWriter marshals Client onto io.Writer.
func (c *Payload) MarshalWriter(w io.Writer) error {
	if err := binary.PutBuffer[uint32](w, c.StreamID); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint32](w, c.ReqID); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint16](w, c.words); err != nil {
		return err
	}
	if _, err := w.Write(c.Payload); err != nil {
		return err
	}
	return nil
}

// UnmarshalReader unmarshals an Client message from a io.Reader stream.
func (c *Payload) UnmarshalReader(r io.Reader) error {
	var err error
	c.StreamID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}

	c.ReqID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}
	c.words, err = binary.GetBuffer[uint16](r)
	if err != nil {
		return err
	}

	c.Payload = make([]byte, 8*c.words)
	_, err = io.ReadFull(r, c.Payload)
	if err != nil {
		return err
	}
	return nil
}
