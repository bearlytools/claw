// Package errors provides an errors package for this service. It includes all of the stdlib's
// functions and types.
package errors

import (
	"github.com/gostdlib/base/context"
	"github.com/gostdlib/base/errors"
)

//go:generate stringer -type=Category -linecomment

// Category represents the category of the error.
type Category uint32

func (c Category) Category() string {
	return c.String()
}

const (
	// CatUnknown represents an unknown category. This should not be used.
	CatUnknown Category = Category(0) // Unknown
	// CatUser represents an error that is caused by bad user input.
	CatUser Category = Category(1) // User
	// CatInternal represents an internal error.
	CatInternal Category = Category(2) // Internal
)

//go:generate stringer -type=Type -linecomment

// Type represents the type of the error.
type Type uint16

func (t Type) Type() string {
	return t.String()
}

const (
	// TypeUnknown represents an unknown type.
	TypeUnknown Type = Type(0) // Unknown
	// TypeBug represents a bug in the calling code. This is only bugs that are known bugs and
	// not because of bad user input. An example would be a switch statement that doesn't cover
	// all cases. The default case should return an error of this type.
	TypeBug Type = Type(1) // Bug
	// TypeParameter represents an error with a parameter that didn't pass validation.
	TypeParameter Type = Type(2) // Parameter
	// TypeConn represents an error with a connection.
	TypeConn Type = Type(3) // Conn
	// TypeTimeout represents a timeout error or cancelation.
	TypeTimeout Type = Type(4) // TimeoutOrCancel
	// TypeFS represents an error with the file system.
	TypeFS Type = Type(5) // FS

	// TypeStorageCreate represents an error with creating storage tables, containers, etc.
	TypeStorageCreate Type = Type(1000) // StorageCreate
	// TypeStorageDelete represents an error with deleting something from storage.
	TypeStorageDelete Type = Type(1001) // StorageDelete
	// TypeStorageGet represents an error with getting something from storage.
	TypeStorageGet Type = Type(1002) // StorageGet
	// TypeStorageList represents an error with listing something from storage.
	TypeStorageList Type = Type(1003) // StorageList
	// TypeStorageUpdate represents an error with updating something in storage.
	TypeStorageUpdate Type = Type(1004) // StorageUpdate
	// TypeStoragePut represents an error with putting something in storage.
	TypeStoragePut Type = Type(1005) // StoragePut
	// TypeStorageClose represents an error with closing storage.
	TypeStorageClose Type = Type(1006) // StorageClose
)

// LogAttrer is an interface that can be implemented by an error to return a list of attributes
// used in logging.
type LogAttrer = errors.LogAttrer

// Error is the error type for this service. Error implements github.com/gostdlib/base/errors.E .
type Error = errors.Error

// EOption is an optional argument for E().
type EOption = errors.EOption

// WithSuppressTraceErr will prevent the trace as being recorded with an error status.
// The trace will still receive the error message. This is useful for errors that are
// retried and you only want to get a status of error if the error is not resolved.
func WithSuppressTraceErr() EOption {
	return errors.WithSuppressTraceErr()
}

// WithCallNum is used if you need to set the runtime.CallNum() in order to get the correct filename and line.
// This can happen if you create a call wrapper around E(), because you would then need to look up one more stack frame
// for every wrapper. This defaults to 1 which sets to the frame of the caller of E().
func WithCallNum(i int) EOption {
	return errors.WithCallNum(i)
}

// WithStackTrace will add a stack trace to the error. This is useful for debugging in certain rare
// cases. This is not recommended for general use as it can cause performance issues when errors
// are created frequently.
func WithStackTrace() EOption {
	return errors.WithStackTrace()
}

// E creates a new Error with the given parameters.
func E(ctx context.Context, c errors.Category, t errors.Type, msg error, options ...errors.EOption) Error {
	// This makes sure we do the correct call number since we are a wrapper. Now, if they set the
	// call number, this will not override it.
	opts := make([]errors.EOption, 0, len(options)+1)
	opts = append(opts, WithCallNum(2))
	opts = append(opts, options...)

	return errors.E(ctx, c, t, msg, opts...)
}
