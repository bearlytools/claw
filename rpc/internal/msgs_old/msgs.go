package msgs

import (
	"fmt"
	"io"

	"github.com/bearlytools/claw/internal/binary"
)

// Type is used to determine the type of message.
type Type uint8

const (
	// TUnknown indicates a bug.
	TUnknown Type = 0
	// TOpen indicates this message is for opening the stream.
	TOpen Type = 1
	// TOpenAck indicates this message is an ack for opening the stream.
	TOpenAck Type = 2
	// TClose indicates this message is for closing the stream.
	TClose Type = 3
	// TPayload indicates this message is a payload message.
	TPayload Type = 4
	// TCancel indicates this message is a cancel message.
	TCancel Type = 5
)

// Msg represents any message that we decode on the wire.
type Msg struct {
	// Type is the type of message.
	Type Type

	// Open is an open message.
	Open Open
	// OpenAck is an open ack message.
	OpenAck OpenAck
	// Close is a close message.
	Close Close
	// Payload is a payload message.
	Payload Payload
	// Cancel is a Cancel message.
	Cancel Cancel
}

// MarshalWriter marshals Client onto io.Writer.
func (m *Msg) MarshalWriter(w io.Writer) error {
	switch m.Type {
	case TOpen:
		return m.Open.MarshalWriter(w)
	case TOpenAck:
		return m.OpenAck.MarshalWriter(w)
	case TClose:
		return m.Close.MarshalWriter(w)
	case TPayload:
		return m.Payload.MarshalWriter(w)
	}
	return fmt.Errorf("unknown type %T", m.Type)
}

// UnmarshalReader unmarshals an Client message from a io.Reader stream.
func (m *Msg) UnmarshalReader(r io.Reader) error {
	i, err := binary.GetBuffer[uint](r)
	if err != nil {
		return err
	}
	switch Type(i) {
	case TOpen:
		o := Open{}
		if err := o.UnmarshalReader(r); err != nil {
			return err
		}
		m.Type = TOpen
		m.Open = o
		return nil
	case TOpenAck:
		oa := OpenAck{}
		if err := oa.UnmarshalReader(r); err != nil {
			return err
		}
		m.Type = TOpenAck
		m.OpenAck = oa
		return nil
	case TClose:
		c := Close{}
		if err := c.UnmarshalReader(r); err != nil {
			return err
		}
		m.Type = TClose
		m.Close = c
		return nil
	case TPayload:
		p := Payload{}
		if err := p.UnmarshalReader(r); err != nil {
			return err
		}
		m.Type = TPayload
		m.Payload = p
		return nil
	}
	return fmt.Errorf("unknown message type %d", i)
}

// HandlerType describes the type of handler for a call.
type HandlerType uint8

const (
	// HTUnknown indicates a bug in the code.
	HTUnknown HandlerType = 0
	// SendStream indicates a stream where the client streams to the server.
	SendStream HandlerType = 1
	// RecvStream indicates a stream where the server streams to the client.
	// The client send a single message on open.
	RecvStream HandlerType = 2
	// BiderStream is a bi-directional stream. The client and server can send messages.
	BiderStream HandlerType = 3
	// Syncronous is when the client sends a message and the server responds.
	Syncronous HandlerType = 4
)
