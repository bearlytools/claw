// Package validate provides validation interceptors for RPC calls.
// It allows validating request and response payloads using pluggable validators.
//
// Validators can be registered per-method to validate incoming requests before
// they reach handlers, and outgoing responses before they're sent to clients.
//
// Example usage:
//
//	// Create a validator registry
//	registry := validate.NewRegistry()
//
//	// Register a validator for a specific method
//	registry.RegisterRequest("myapp/UserService/CreateUser", &userCreateValidator{})
//
//	// Add the interceptor to the server
//	srv := server.New(
//	    server.WithUnaryInterceptor(validate.UnaryServerInterceptor(registry)),
//	)
package validate

import (
	"errors"
	"fmt"
	"iter"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
	"github.com/bearlytools/claw/rpc/internal/msgs"
)

// Common validation errors.
var (
	ErrValidation = errors.New("validation error")
)

// ValidationError wraps a validation failure with details about what failed.
type ValidationError struct {
	// Field is the name of the field that failed validation (optional).
	Field string
	// Message describes why validation failed.
	Message string
	// Cause is the underlying error if any.
	Cause error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrValidation
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// NewFieldError creates a validation error for a specific field.
func NewFieldError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// Validator validates a message payload.
// Implementations should be safe for concurrent use.
type Validator interface {
	// Validate checks the payload and returns an error if invalid.
	// The context can be used for timeout/cancellation.
	// The payload is the raw bytes of the request or response.
	Validate(ctx context.Context, payload []byte) error
}

// ValidatorFunc is a function adapter for Validator.
type ValidatorFunc func(ctx context.Context, payload []byte) error

func (f ValidatorFunc) Validate(ctx context.Context, payload []byte) error {
	return f(ctx, payload)
}

// Registry holds validators for different methods.
// It is safe for concurrent use after initial configuration.
type Registry struct {
	// requestValidators maps "pkg/service/method" to request validators.
	requestValidators map[string]Validator
	// responseValidators maps "pkg/service/method" to response validators.
	responseValidators map[string]Validator
}

// NewRegistry creates a new validator registry.
func NewRegistry() *Registry {
	return &Registry{
		requestValidators:  make(map[string]Validator),
		responseValidators: make(map[string]Validator),
	}
}

// RegisterRequest registers a validator for incoming requests to a method.
// Pattern format: "pkg/service/method"
func (r *Registry) RegisterRequest(pattern string, v Validator) *Registry {
	r.requestValidators[pattern] = v
	return r
}

// RegisterResponse registers a validator for outgoing responses from a method.
// Pattern format: "pkg/service/method"
func (r *Registry) RegisterResponse(pattern string, v Validator) *Registry {
	r.responseValidators[pattern] = v
	return r
}

// RegisterRequestFunc registers a function as a request validator.
func (r *Registry) RegisterRequestFunc(pattern string, f func(ctx context.Context, payload []byte) error) *Registry {
	return r.RegisterRequest(pattern, ValidatorFunc(f))
}

// RegisterResponseFunc registers a function as a response validator.
func (r *Registry) RegisterResponseFunc(pattern string, f func(ctx context.Context, payload []byte) error) *Registry {
	return r.RegisterResponse(pattern, ValidatorFunc(f))
}

// GetRequestValidator returns the request validator for a method, if any.
func (r *Registry) GetRequestValidator(pkg, service, method string) (Validator, bool) {
	pattern := pkg + "/" + service + "/" + method
	v, ok := r.requestValidators[pattern]
	return v, ok
}

// GetResponseValidator returns the response validator for a method, if any.
func (r *Registry) GetResponseValidator(pkg, service, method string) (Validator, bool) {
	pattern := pkg + "/" + service + "/" + method
	v, ok := r.responseValidators[pattern]
	return v, ok
}

// UnaryServerInterceptor returns a server interceptor that validates requests
// before they reach the handler and responses before they're sent.
func UnaryServerInterceptor(registry *Registry) interceptor.UnaryServerInterceptor {
	return func(ctx context.Context, req []byte, info *interceptor.UnaryServerInfo, handler interceptor.UnaryHandler) ([]byte, error) {
		// Validate request if a validator is registered.
		if v, ok := registry.GetRequestValidator(info.Package, info.Service, info.Method); ok {
			if err := v.Validate(ctx, req); err != nil {
				return nil, wrapValidationError(err)
			}
		}

		// Call the handler.
		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		// Validate response if a validator is registered.
		if v, ok := registry.GetResponseValidator(info.Package, info.Service, info.Method); ok {
			if err := v.Validate(ctx, resp); err != nil {
				return nil, wrapValidationError(err)
			}
		}

		return resp, nil
	}
}

// StreamServerInterceptor returns a server interceptor that validates
// streaming messages.
func StreamServerInterceptor(registry *Registry) interceptor.StreamServerInterceptor {
	return func(ctx context.Context, ss interceptor.ServerStream, info *interceptor.StreamServerInfo, handler interceptor.StreamHandler) error {
		// Wrap the stream to intercept messages.
		wrapped := &validatingServerStream{
			ServerStream:      ss,
			ctx:               ctx,
			requestValidator:  nil,
			responseValidator: nil,
		}

		// Get validators for this method.
		if v, ok := registry.GetRequestValidator(info.Package, info.Service, info.Method); ok {
			wrapped.requestValidator = v
		}
		if v, ok := registry.GetResponseValidator(info.Package, info.Service, info.Method); ok {
			wrapped.responseValidator = v
		}

		return handler(ctx, wrapped)
	}
}

// validatingServerStream wraps a ServerStream to validate messages.
type validatingServerStream struct {
	interceptor.ServerStream
	ctx               context.Context
	requestValidator  Validator
	responseValidator Validator
	validationErr     error
}

func (s *validatingServerStream) Recv() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for payload := range s.ServerStream.Recv() {
			// Validate incoming message.
			if s.requestValidator != nil {
				if err := s.requestValidator.Validate(s.ctx, payload); err != nil {
					s.validationErr = wrapValidationError(err)
					return
				}
			}
			if !yield(payload) {
				return
			}
		}
	}
}

// Err returns any validation error that occurred during Recv.
func (s *validatingServerStream) Err() error {
	return s.validationErr
}

func (s *validatingServerStream) Send(payload []byte) error {
	// Validate outgoing message.
	if s.responseValidator != nil {
		if err := s.responseValidator.Validate(s.ctx, payload); err != nil {
			return wrapValidationError(err)
		}
	}

	return s.ServerStream.Send(payload)
}

// UnaryClientInterceptor returns a client interceptor that validates requests
// before sending and responses after receiving.
func UnaryClientInterceptor(registry *Registry) interceptor.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req []byte, invoker interceptor.UnaryInvoker) ([]byte, error) {
		// Parse method into components.
		pkg, service, call := parseMethod(method)

		// Validate request if a validator is registered.
		if v, ok := registry.GetRequestValidator(pkg, service, call); ok {
			if err := v.Validate(ctx, req); err != nil {
				return nil, wrapValidationError(err)
			}
		}

		// Invoke the actual call.
		resp, err := invoker(ctx, req)
		if err != nil {
			return nil, err
		}

		// Validate response if a validator is registered.
		if v, ok := registry.GetResponseValidator(pkg, service, call); ok {
			if err := v.Validate(ctx, resp); err != nil {
				return nil, wrapValidationError(err)
			}
		}

		return resp, nil
	}
}

// parseMethod parses a method string "pkg/service/method" into components.
func parseMethod(method string) (pkg, service, call string) {
	// Simple split - assumes format "pkg/service/method"
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(method); i++ {
		if method[i] == '/' {
			parts = append(parts, method[start:i])
			start = i + 1
		}
	}
	parts = append(parts, method[start:])

	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2]
	case 2:
		return "", parts[0], parts[1]
	case 1:
		return "", "", parts[0]
	default:
		return "", "", ""
	}
}

// wrapValidationError ensures the error has the appropriate error code.
func wrapValidationError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a ValidationError, wrap it with the RPC error code.
	var ve *ValidationError
	if errors.As(err, &ve) {
		return fmt.Errorf("%s: %w", msgs.ErrInvalidArgument.String(), err)
	}

	// Wrap generic errors.
	return fmt.Errorf("%s: validation failed: %w", msgs.ErrInvalidArgument.String(), err)
}
