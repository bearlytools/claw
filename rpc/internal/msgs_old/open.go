package msgs

import (
	"errors"
	"fmt"
	"io"

	"github.com/bearlytools/claw/internal/binary"

	"unsafe"
)

//go:generate go tool github.com/johnsiilver/stringer -type=RPCType

// RPCType is the type of RPC being performed.
type RPCType uint8

const (
	// RTUnknown is always a bug.
	RTUnknown = 0
	// RTSyncronous represents a request-response style RPC.
	RTSyncronous RPCType = 1
	// RTSend represents a send-only RPC.
	RTSend RPCType = 2
	// RTRecv represents a receive-only RPC.
	RTRecv RPCType = 3
	// RTBiDirectional represents a bi-directional streaming RPC.
	RTBiDirectional RPCType = 4
)

// Descr gives the description of the RPC sot that it can match up with the other side.
type Descr struct {
	// Package is the package the RPC is defined in.
	Package string
	// Service is the name of the service the RPC is defined in.
	Service string
	// Call is the name of the call the RPC defines.
	Call string
	// Type is the type of RPC being performed.
	Type RPCType
}

func (d *Descr) validate() error {
	if d.Package == "" {
		return errors.New("Descr.Package was empty string")
	}
	if d.Service == "" {
		return errors.New("Descr.Service was empty string")
	}
	if d.Call == "" {
		return errors.New("Descr.Call was empty string")
	}
	if d.Type == RTUnknown {
		return errors.New("Descr.Type was RTUnknown")
	}
	return nil
}

// MarshalWriter marshals Descr onto io.Writer.
func (d *Descr) MarshalWriter(w io.Writer) error {
	if len(d.Package) > 500 {
		return fmt.Errorf("Descr.Package too long: %d bytes", len(d.Package))
	}
	if len(d.Service) > 500 {
		return fmt.Errorf("Descr.Service too long: %d bytes", len(d.Service))
	}
	if len(d.Call) > 500 {
		return fmt.Errorf("Descr.Call too long: %d bytes", len(d.Call))
	}
	if err := binary.PutBuffer[uint32](w, uint32(len(d.Package))); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint32](w, uint32(len(d.Service))); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint32](w, uint32(len(d.Call))); err != nil {
		return err
	}

	if _, err := w.Write(unsafe.Slice(unsafe.StringData(d.Package), len(d.Package))); err != nil {
		return err
	}
	if _, err := w.Write(unsafe.Slice(unsafe.StringData(d.Service), len(d.Service))); err != nil {
		return err
	}
	if _, err := w.Write(unsafe.Slice(unsafe.StringData(d.Call), len(d.Call))); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint8](w, uint8(d.Type)); err != nil {
		return err
	}
	return nil
}

func (d *Descr) UnmarshalReader(w io.Reader) error {
	b := [12]byte{}
	if _, err := io.ReadFull(w, b[:]); err != nil {
		return err
	}
	pkgLen := binary.Get[uint32](b[0:4])
	svcLen := binary.Get[uint32](b[4:8])
	callLen := binary.Get[uint32](b[8:12])

	s := make([]byte, pkgLen+svcLen+callLen+1)
	if _, err := io.ReadFull(w, b[:]); err != nil {
		return err
	}

	d.Package = unsafe.String(unsafe.SliceData(s[0:pkgLen]), int(pkgLen))
	d.Service = unsafe.String(unsafe.SliceData(s[pkgLen:pkgLen+svcLen]), int(svcLen))
	d.Call = unsafe.String(unsafe.SliceData(s[pkgLen+svcLen:pkgLen+svcLen+callLen]), int(callLen))
	tByte, err := binary.GetBuffer[uint8](w)
	if err != nil {
		return err
	}
	d.Type = RPCType(tByte)
	return nil
}

// Open is the open message for an RPC.
type Open struct {
	// OpenID is the ID of the Open message so the response can refer to it.
	OpenID uint32
	// Descr is the RPC descriptor.
	Descr Descr
}

// MarshalWriter marshals Open onto io.Writer.
func (o *Open) MarshalWriter(w io.Writer) error {
	if err := binary.PutBuffer[uint32](w, o.OpenID); err != nil {
		return err
	}
	return o.Descr.MarshalWriter(w)
}

// UnmarshalReader unmarshals an Open message from a io.Reader stream.
func (o *Open) UnmarshalReader(r io.Reader) error {
	var err error
	o.OpenID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}

	var descr Descr
	if err = descr.UnmarshalReader(r); err != nil {
		return err
	}
	o.Descr = descr
	return nil
}

// OpenAck is the response to an Open message.
type OpenAck struct {
	// OpenID is the ID of the Open message being acknowledged.
	OpenID uint32
	// StreamID is the ID of the stream.
	StreamID uint32
}

// MarshalWriter marshals OpenAck onto io.Writer.
func (o *OpenAck) MarshalWriter(w io.Writer) error {
	if err := binary.PutBuffer[uint32](w, o.OpenID); err != nil {
		return err
	}
	return binary.PutBuffer[uint32](w, o.StreamID)
}

// UnmarshalReader unmarshals an OpenAck message from a io.Reader stream.
func (o *OpenAck) UnmarshalReader(r io.Reader) error {
	var err error
	o.OpenID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}
	o.StreamID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}
	return nil
}
