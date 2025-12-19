package msgs

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
)

var (
	ErrUnknownStream = fmt.Errorf("unknown stream")
)

//go:generate go tool github.com/johnsiilver/stringer -type=CloseErrCode
type CloseErrCode uint8

const (
	// CERUnset indicates there was no error.
	CERUnset = 0
	// Internal indicates there was an internal error.
	Internal CloseErrCode = 1
)

// Close is a message that closes the stream. If ErrSize is set then the stream close is due to some type
// of error.
type Close struct {
	// StreamID is the ID of the stream that is closing.
	StreamID uint32
	// ErrSize is the size of the error message. 0 indicates no error.
	ErrSize uint16
	// ErrCode is a code describing the error type. 0 indicates no error.
	ErrCode CloseErrCode
	// Error is an error message.
	Error string
}

// NewClose creates a new Close message.
func NewClose(streamID uint32, errCode CloseErrCode, errMsg string) Msg {
	return Msg{
		Type: TClose,
		Close: Close{
			StreamID: streamID,
			ErrSize:  uint16(len(errMsg)),
			ErrCode:  errCode,
			Error:    errMsg,
		},
	}
}

// MarshalWriter marshals Close onto io.Writer.
func (c *Close) MarshalWriter(w io.Writer) error {
	if err := binary.PutBuffer[uint32](w, c.StreamID); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint16](w, c.ErrSize); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint8](w, uint8(c.ErrCode)); err != nil {
		return err
	}
	if c.ErrSize > 0 {
		if _, err := w.Write(unsafe.Slice(unsafe.StringData(c.Error), len(c.Error))); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalReader unmarshals an Close message from a io.Reader stream.
func (c *Close) UnmarshalReader(r io.Reader) error {
	var err error
	c.StreamID, err = binary.GetBuffer[uint32](r)
	if err != nil {
		return err
	}
	c.ErrSize, err = binary.GetBuffer[uint16](r)
	if err != nil {
		return err
	}
	errCode, err := binary.GetBuffer[uint8](r)
	if err != nil {
		return err
	}
	c.ErrCode = CloseErrCode(errCode)
	var n int
	if c.ErrSize > 0 {
		b := make([]byte, int(c.ErrSize))
		n, err = r.Read(b)
		if err != nil {
			return err
		}
		if n != int(c.ErrSize) {
			return fmt.Errorf("Close message message string was too small")
		}
		c.Error = unsafe.String(unsafe.SliceData(b), len(b))
	}
	return nil
}

// CloseAck is a message that acknowledges the closing of a stream. If ErrSize is set then the stream close is due to
// some type of error.
type CloseAck struct {
	// StreamID is the ID of the stream that is closing.
	StreamID uint16
	// ErrSize is the size of the error message. 0 indicates no error.
	ErrSize uint16
	// ErrCode is a code describing the error type. 0 indicates no error.
	ErrCode CloseErrCode
	// Error is an error message.
	Error string
}

// MarshalWriter marshals CloseAck onto io.Writer.
func (c *CloseAck) MarshalWriter(w io.Writer) error {
	if err := binary.PutBuffer[uint16](w, c.StreamID); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint16](w, c.ErrSize); err != nil {
		return err
	}
	if err := binary.PutBuffer[uint8](w, uint8(c.ErrCode)); err != nil {
		return err
	}
	if c.ErrSize > 0 {
		if _, err := w.Write(unsafe.Slice(unsafe.StringData(c.Error), len(c.Error))); err != nil {
			return err
		}
	}
	return nil

}

// UnmarshalReader unmarshals a CloseAck message from a io.Reader stream.
func (c *CloseAck) UnmarshalReader(r io.Reader) error {
	var err error
	c.StreamID, err = binary.GetBuffer[uint16](r)
	if err != nil {
		return err
	}
	c.ErrSize, err = binary.GetBuffer[uint16](r)
	if err != nil {
		return err
	}
	errCode, err := binary.GetBuffer[uint8](r)
	if err != nil {
		return err
	}
	c.ErrCode = CloseErrCode(errCode)
	if c.ErrSize > 0 {
		b := make([]byte, int(c.ErrSize))
		n, err := r.Read(b)
		if err != nil {
			return err
		}
		if n != int(c.ErrSize) {
			return fmt.Errorf("CloseAck message message string was too small")
		}
		c.Error = unsafe.String(unsafe.SliceData(b), len(b))
	}
	return nil
}
