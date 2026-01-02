package resolver

import (
	"testing"

	"github.com/gostdlib/base/context"
)

// fakeBuilder is a test builder.
type fakeBuilder struct {
	scheme string
}

func (b *fakeBuilder) Scheme() string {
	return b.scheme
}

func (b *fakeBuilder) Build(target Target, opts BuildOptions) (Resolver, error) {
	return &fakeResolver{}, nil
}

// fakeResolver is a test resolver.
type fakeResolver struct{}

func (r *fakeResolver) Resolve(ctx context.Context) ([]Address, error) {
	return nil, nil
}

func (r *fakeResolver) Close() error {
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	// Save original state and restore after test
	mu.Lock()
	originalBuilders := make(map[string]Builder)
	for k, v := range builders {
		originalBuilders[k] = v
	}
	mu.Unlock()

	defer func() {
		mu.Lock()
		builders = originalBuilders
		mu.Unlock()
	}()

	// Clear for test
	mu.Lock()
	builders = make(map[string]Builder)
	mu.Unlock()

	tests := []struct {
		name       string
		scheme     string
		wantFound  bool
	}{
		{
			name:      "Success: register and get",
			scheme:    "test",
			wantFound: true,
		},
		{
			name:      "Success: not found",
			scheme:    "unknown",
			wantFound: false,
		},
	}

	// Register test builder
	Register(&fakeBuilder{scheme: "test"})

	for _, test := range tests {
		got, ok := Get(test.scheme)
		if ok != test.wantFound {
			t.Errorf("[TestRegisterAndGet](%s): found = %v, want %v", test.name, ok, test.wantFound)
			continue
		}

		if test.wantFound && got == nil {
			t.Errorf("[TestRegisterAndGet](%s): got nil builder, want non-nil", test.name)
		}
	}
}

func TestRegisterReplaces(t *testing.T) {
	// Save original state and restore after test
	mu.Lock()
	originalBuilders := make(map[string]Builder)
	for k, v := range builders {
		originalBuilders[k] = v
	}
	mu.Unlock()

	defer func() {
		mu.Lock()
		builders = originalBuilders
		mu.Unlock()
	}()

	// Clear for test
	mu.Lock()
	builders = make(map[string]Builder)
	mu.Unlock()

	builder1 := &fakeBuilder{scheme: "test"}
	builder2 := &fakeBuilder{scheme: "test"}

	Register(builder1)
	got1, _ := Get("test")
	if got1 != builder1 {
		t.Error("[TestRegisterReplaces]: first registration failed")
	}

	Register(builder2)
	got2, _ := Get("test")
	if got2 != builder2 {
		t.Error("[TestRegisterReplaces]: second registration should replace first")
	}
}

func TestSchemes(t *testing.T) {
	// Save original state and restore after test
	mu.Lock()
	originalBuilders := make(map[string]Builder)
	for k, v := range builders {
		originalBuilders[k] = v
	}
	mu.Unlock()

	defer func() {
		mu.Lock()
		builders = originalBuilders
		mu.Unlock()
	}()

	// Clear for test
	mu.Lock()
	builders = make(map[string]Builder)
	mu.Unlock()

	// Register some builders
	Register(&fakeBuilder{scheme: "aaa"})
	Register(&fakeBuilder{scheme: "bbb"})
	Register(&fakeBuilder{scheme: "ccc"})

	schemes := Schemes()
	if len(schemes) != 3 {
		t.Errorf("[TestSchemes]: got %d schemes, want 3", len(schemes))
	}

	// Check all schemes are present
	schemeSet := make(map[string]bool)
	for _, s := range schemes {
		schemeSet[s] = true
	}

	for _, want := range []string{"aaa", "bbb", "ccc"} {
		if !schemeSet[want] {
			t.Errorf("[TestSchemes]: scheme %q not found", want)
		}
	}
}
