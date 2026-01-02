package validate

import (
	"errors"
	"iter"
	"testing"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/rpc/interceptor"
)

func TestValidationErrorFormat(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantMsg string
	}{
		{
			name:    "Success: message only",
			err:     NewValidationError("value must be positive"),
			wantMsg: "validation error: value must be positive",
		},
		{
			name:    "Success: field and message",
			err:     NewFieldError("age", "must be at least 18"),
			wantMsg: `validation error: field "age": must be at least 18`,
		},
	}

	for _, test := range tests {
		if got := test.err.Error(); got != test.wantMsg {
			t.Errorf("TestValidationErrorFormat(%s): got %q, want %q", test.name, got, test.wantMsg)
		}
	}
}

func TestValidationErrorUnwrap(t *testing.T) {
	tests := []struct {
		name      string
		err       *ValidationError
		wantCause error
	}{
		{
			name:      "Success: no cause returns ErrValidation",
			err:       NewValidationError("test"),
			wantCause: ErrValidation,
		},
		{
			name: "Success: with cause returns cause",
			err: &ValidationError{
				Message: "test",
				Cause:   errors.New("underlying"),
			},
			wantCause: errors.New("underlying"),
		},
	}

	for _, test := range tests {
		got := test.err.Unwrap()
		if got.Error() != test.wantCause.Error() {
			t.Errorf("TestValidationErrorUnwrap(%s): got %v, want %v", test.name, got, test.wantCause)
		}
	}
}

func TestRegistryRequestValidator(t *testing.T) {
	validator := ValidatorFunc(func(ctx context.Context, payload []byte) error {
		return nil
	})

	reg := NewRegistry().
		RegisterRequest("myapp/UserService/GetUser", validator)

	tests := []struct {
		name    string
		pkg     string
		service string
		method  string
		wantOK  bool
	}{
		{
			name:    "Success: exact match",
			pkg:     "myapp",
			service: "UserService",
			method:  "GetUser",
			wantOK:  true,
		},
		{
			name:    "Success: no match for different method",
			pkg:     "myapp",
			service: "UserService",
			method:  "DeleteUser",
			wantOK:  false,
		},
		{
			name:    "Success: no match for different service",
			pkg:     "myapp",
			service: "OrderService",
			method:  "GetUser",
			wantOK:  false,
		},
	}

	for _, test := range tests {
		_, ok := reg.GetRequestValidator(test.pkg, test.service, test.method)
		if ok != test.wantOK {
			t.Errorf("TestRegistryRequestValidator(%s): got ok=%v, want %v", test.name, ok, test.wantOK)
		}
	}
}

func TestRegistryResponseValidator(t *testing.T) {
	validator := ValidatorFunc(func(ctx context.Context, payload []byte) error {
		return nil
	})

	reg := NewRegistry().
		RegisterResponse("myapp/UserService/GetUser", validator)

	_, ok := reg.GetResponseValidator("myapp", "UserService", "GetUser")
	if !ok {
		t.Errorf("TestRegistryResponseValidator: expected match, got none")
	}

	_, ok = reg.GetResponseValidator("myapp", "UserService", "DeleteUser")
	if ok {
		t.Errorf("TestRegistryResponseValidator: expected no match for DeleteUser")
	}
}

func TestUnaryServerInterceptorValidatesRequest(t *testing.T) {
	tests := []struct {
		name       string
		validatorFn func(ctx context.Context, payload []byte) error
		wantErr    bool
	}{
		{
			name: "Success: valid request passes",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Error: invalid request fails",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return NewValidationError("invalid payload")
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		reg := NewRegistry().
			RegisterRequest("myapp/UserService/GetUser", ValidatorFunc(test.validatorFn))

		interceptorFn := UnaryServerInterceptor(reg)

		info := &interceptor.UnaryServerInfo{
			Package: "myapp",
			Service: "UserService",
			Method:  "GetUser",
		}

		handler := func(ctx context.Context, req []byte) ([]byte, error) {
			return []byte("response"), nil
		}

		ctx := t.Context()
		_, err := interceptorFn(ctx, []byte("request"), info, handler)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestUnaryServerInterceptorValidatesRequest(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestUnaryServerInterceptorValidatesRequest(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestUnaryServerInterceptorValidatesResponse(t *testing.T) {
	tests := []struct {
		name       string
		validatorFn func(ctx context.Context, payload []byte) error
		wantErr    bool
	}{
		{
			name: "Success: valid response passes",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Error: invalid response fails",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return NewValidationError("invalid response")
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		reg := NewRegistry().
			RegisterResponse("myapp/UserService/GetUser", ValidatorFunc(test.validatorFn))

		interceptorFn := UnaryServerInterceptor(reg)

		info := &interceptor.UnaryServerInfo{
			Package: "myapp",
			Service: "UserService",
			Method:  "GetUser",
		}

		handler := func(ctx context.Context, req []byte) ([]byte, error) {
			return []byte("response"), nil
		}

		ctx := t.Context()
		_, err := interceptorFn(ctx, []byte("request"), info, handler)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestUnaryServerInterceptorValidatesResponse(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestUnaryServerInterceptorValidatesResponse(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestUnaryServerInterceptorNoValidator(t *testing.T) {
	reg := NewRegistry()
	interceptorFn := UnaryServerInterceptor(reg)

	info := &interceptor.UnaryServerInfo{
		Package: "myapp",
		Service: "UserService",
		Method:  "GetUser",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req []byte) ([]byte, error) {
		handlerCalled = true
		return []byte("response"), nil
	}

	ctx := t.Context()
	resp, err := interceptorFn(ctx, []byte("request"), info, handler)
	if err != nil {
		t.Errorf("TestUnaryServerInterceptorNoValidator: unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Errorf("TestUnaryServerInterceptorNoValidator: handler was not called")
	}
	if string(resp) != "response" {
		t.Errorf("TestUnaryServerInterceptorNoValidator: got response %q, want %q", string(resp), "response")
	}
}

func TestUnaryClientInterceptorValidatesRequest(t *testing.T) {
	tests := []struct {
		name       string
		validatorFn func(ctx context.Context, payload []byte) error
		wantErr    bool
	}{
		{
			name: "Success: valid request passes",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "Error: invalid request fails",
			validatorFn: func(ctx context.Context, payload []byte) error {
				return NewValidationError("invalid payload")
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		reg := NewRegistry().
			RegisterRequest("myapp/UserService/GetUser", ValidatorFunc(test.validatorFn))

		interceptorFn := UnaryClientInterceptor(reg)

		invoker := func(ctx context.Context, req []byte) ([]byte, error) {
			return []byte("response"), nil
		}

		ctx := t.Context()
		_, err := interceptorFn(ctx, "myapp/UserService/GetUser", []byte("request"), invoker)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestUnaryClientInterceptorValidatesRequest(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestUnaryClientInterceptorValidatesRequest(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestUnaryClientInterceptorValidatesResponse(t *testing.T) {
	reg := NewRegistry().
		RegisterResponse("myapp/UserService/GetUser", ValidatorFunc(func(ctx context.Context, payload []byte) error {
			if string(payload) == "bad" {
				return NewValidationError("bad response")
			}
			return nil
		}))

	interceptorFn := UnaryClientInterceptor(reg)

	tests := []struct {
		name     string
		response []byte
		wantErr  bool
	}{
		{
			name:     "Success: valid response passes",
			response: []byte("good"),
			wantErr:  false,
		},
		{
			name:     "Error: invalid response fails",
			response: []byte("bad"),
			wantErr:  true,
		},
	}

	for _, test := range tests {
		invoker := func(ctx context.Context, req []byte) ([]byte, error) {
			return test.response, nil
		}

		ctx := t.Context()
		_, err := interceptorFn(ctx, "myapp/UserService/GetUser", []byte("request"), invoker)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestUnaryClientInterceptorValidatesResponse(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestUnaryClientInterceptorValidatesResponse(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestParseMethod(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		wantPkg     string
		wantService string
		wantCall    string
	}{
		{
			name:        "Success: full method path",
			method:      "myapp/UserService/GetUser",
			wantPkg:     "myapp",
			wantService: "UserService",
			wantCall:    "GetUser",
		},
		{
			name:        "Success: two parts",
			method:      "UserService/GetUser",
			wantPkg:     "",
			wantService: "UserService",
			wantCall:    "GetUser",
		},
		{
			name:        "Success: one part",
			method:      "GetUser",
			wantPkg:     "",
			wantService: "",
			wantCall:    "GetUser",
		},
	}

	for _, test := range tests {
		pkg, svc, call := parseMethod(test.method)
		if pkg != test.wantPkg || svc != test.wantService || call != test.wantCall {
			t.Errorf("TestParseMethod(%s): got (%q, %q, %q), want (%q, %q, %q)",
				test.name, pkg, svc, call, test.wantPkg, test.wantService, test.wantCall)
		}
	}
}

// fakeServerStream implements interceptor.ServerStream for testing.
type fakeServerStream struct {
	messages [][]byte
	sent     [][]byte
	ctx      context.Context
}

func (f *fakeServerStream) Recv() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for _, msg := range f.messages {
			if !yield(msg) {
				return
			}
		}
	}
}

func (f *fakeServerStream) Send(payload []byte) error {
	f.sent = append(f.sent, payload)
	return nil
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}

func TestStreamServerInterceptorValidatesRecv(t *testing.T) {
	reg := NewRegistry().
		RegisterRequest("myapp/UserService/StreamUsers", ValidatorFunc(func(ctx context.Context, payload []byte) error {
			if string(payload) == "invalid" {
				return NewValidationError("invalid message")
			}
			return nil
		}))

	interceptorFn := StreamServerInterceptor(reg)

	tests := []struct {
		name     string
		messages [][]byte
		wantErr  bool
	}{
		{
			name:     "Success: all valid messages pass",
			messages: [][]byte{[]byte("msg1"), []byte("msg2")},
			wantErr:  false,
		},
		{
			name:     "Error: invalid message fails",
			messages: [][]byte{[]byte("msg1"), []byte("invalid"), []byte("msg2")},
			wantErr:  true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		stream := &fakeServerStream{
			messages: test.messages,
			ctx:      ctx,
		}

		info := &interceptor.StreamServerInfo{
			Package: "myapp",
			Service: "UserService",
			Method:  "StreamUsers",
		}

		var gotErr error
		handler := func(ctx context.Context, ss interceptor.ServerStream) error {
			for range ss.Recv() {
				// Consume all messages
			}
			// Check if the wrapped stream has a validation error
			if vs, ok := ss.(*validatingServerStream); ok {
				gotErr = vs.Err()
			}
			return gotErr
		}

		err := interceptorFn(ctx, stream, info, handler)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestStreamServerInterceptorValidatesRecv(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestStreamServerInterceptorValidatesRecv(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestStreamServerInterceptorValidatesSend(t *testing.T) {
	reg := NewRegistry().
		RegisterResponse("myapp/UserService/StreamUsers", ValidatorFunc(func(ctx context.Context, payload []byte) error {
			if string(payload) == "invalid" {
				return NewValidationError("invalid response")
			}
			return nil
		}))

	interceptorFn := StreamServerInterceptor(reg)

	tests := []struct {
		name       string
		sendPayload []byte
		wantErr    bool
	}{
		{
			name:       "Success: valid response passes",
			sendPayload: []byte("valid"),
			wantErr:    false,
		},
		{
			name:       "Error: invalid response fails",
			sendPayload: []byte("invalid"),
			wantErr:    true,
		},
	}

	for _, test := range tests {
		ctx := t.Context()
		stream := &fakeServerStream{
			messages: nil,
			ctx:      ctx,
		}

		info := &interceptor.StreamServerInfo{
			Package: "myapp",
			Service: "UserService",
			Method:  "StreamUsers",
		}

		var sendErr error
		handler := func(ctx context.Context, ss interceptor.ServerStream) error {
			sendErr = ss.Send(test.sendPayload)
			return sendErr
		}

		err := interceptorFn(ctx, stream, info, handler)

		switch {
		case err == nil && test.wantErr:
			t.Errorf("TestStreamServerInterceptorValidatesSend(%s): got err == nil, want err != nil", test.name)
		case err != nil && !test.wantErr:
			t.Errorf("TestStreamServerInterceptorValidatesSend(%s): got err == %s, want err == nil", test.name, err)
		}
	}
}

func TestValidatorFunc(t *testing.T) {
	called := false
	fn := ValidatorFunc(func(ctx context.Context, payload []byte) error {
		called = true
		return nil
	})

	ctx := t.Context()
	err := fn.Validate(ctx, []byte("test"))
	if err != nil {
		t.Errorf("TestValidatorFunc: unexpected error: %v", err)
	}
	if !called {
		t.Errorf("TestValidatorFunc: function was not called")
	}
}

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantContains string
	}{
		{
			name:        "Success: nil returns nil",
			err:         nil,
			wantContains: "",
		},
		{
			name:        "Success: ValidationError is wrapped with code",
			err:         NewValidationError("test error"),
			wantContains: "InvalidArgument",
		},
		{
			name:        "Success: generic error is wrapped",
			err:         errors.New("some error"),
			wantContains: "validation failed",
		},
	}

	for _, test := range tests {
		result := wrapValidationError(test.err)
		if test.err == nil {
			if result != nil {
				t.Errorf("TestWrapValidationError(%s): got %v, want nil", test.name, result)
			}
			continue
		}
		if result == nil {
			t.Errorf("TestWrapValidationError(%s): got nil, want non-nil", test.name)
			continue
		}
		if test.wantContains != "" {
			errStr := result.Error()
			found := false
			for i := 0; i <= len(errStr)-len(test.wantContains); i++ {
				if errStr[i:i+len(test.wantContains)] == test.wantContains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("TestWrapValidationError(%s): error %q does not contain %q", test.name, errStr, test.wantContains)
			}
		}
	}
}

func TestRegistryChaining(t *testing.T) {
	v1 := ValidatorFunc(func(ctx context.Context, payload []byte) error { return nil })
	v2 := ValidatorFunc(func(ctx context.Context, payload []byte) error { return nil })

	reg := NewRegistry().
		RegisterRequest("myapp/UserService/GetUser", v1).
		RegisterResponse("myapp/UserService/GetUser", v2).
		RegisterRequestFunc("myapp/UserService/CreateUser", func(ctx context.Context, payload []byte) error {
			return nil
		}).
		RegisterResponseFunc("myapp/UserService/CreateUser", func(ctx context.Context, payload []byte) error {
			return nil
		})

	_, ok := reg.GetRequestValidator("myapp", "UserService", "GetUser")
	if !ok {
		t.Errorf("TestRegistryChaining: expected request validator for GetUser")
	}

	_, ok = reg.GetResponseValidator("myapp", "UserService", "GetUser")
	if !ok {
		t.Errorf("TestRegistryChaining: expected response validator for GetUser")
	}

	_, ok = reg.GetRequestValidator("myapp", "UserService", "CreateUser")
	if !ok {
		t.Errorf("TestRegistryChaining: expected request validator for CreateUser")
	}

	_, ok = reg.GetResponseValidator("myapp", "UserService", "CreateUser")
	if !ok {
		t.Errorf("TestRegistryChaining: expected response validator for CreateUser")
	}
}
