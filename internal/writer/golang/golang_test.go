package golang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldSkipGeneration(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	localDir := filepath.Join(tmpDir, "local")

	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}

	// Create test package paths
	vendoredPkgDir := filepath.Join(vendorDir, "github.com", "foo", "bar")
	if err := os.MkdirAll(vendoredPkgDir, 0755); err != nil {
		t.Fatalf("failed to create vendored package dir: %v", err)
	}

	// Create a .go file in the vendored package
	vendoredGoFile := filepath.Join(vendoredPkgDir, "test.go")
	if err := os.WriteFile(vendoredGoFile, []byte("package test"), 0644); err != nil {
		t.Fatalf("failed to create vendored .go file: %v", err)
	}

	// Create a .claw file path in vendor (for testing)
	vendoredClawPath := filepath.Join(vendoredPkgDir, "test.claw")

	// Create a non-existent .go file path in vendor
	vendoredMissingGoFile := filepath.Join(vendoredPkgDir, "missing.go")

	// Create a local .go file
	localGoFile := filepath.Join(localDir, "local.go")
	localClawPath := filepath.Join(localDir, "local.claw")

	tests := []struct {
		name        string
		writer      *Writer
		goFilePath  string
		clawPath    string
		wantSkip    bool
		description string
	}{
		{
			name: "forceRegenerate=true, vendored file exists - should NOT skip",
			writer: &Writer{
				forceRegenerate: true,
				vendorDir:       vendorDir,
			},
			goFilePath:  vendoredGoFile,
			clawPath:    vendoredClawPath,
			wantSkip:    false,
			description: "Force regenerate mode must always generate, never skip",
		},
		{
			name: "forceRegenerate=false, vendorDir empty - should NOT skip",
			writer: &Writer{
				forceRegenerate: false,
				vendorDir:       "",
			},
			goFilePath:  vendoredGoFile,
			clawPath:    vendoredClawPath,
			wantSkip:    false,
			description: "No vendor directory means not in lazy loading mode - always generate",
		},
		{
			name: "local file (not in vendor) - should NOT skip",
			writer: &Writer{
				forceRegenerate: false,
				vendorDir:       vendorDir,
			},
			goFilePath:  localGoFile,
			clawPath:    localClawPath,
			wantSkip:    false,
			description: "Local files are always regenerated, never skipped",
		},
		{
			name: "vendored file with existing .go - should SKIP",
			writer: &Writer{
				forceRegenerate: false,
				vendorDir:       vendorDir,
			},
			goFilePath:  vendoredGoFile,
			clawPath:    vendoredClawPath,
			wantSkip:    true,
			description: "Vendored file with existing .go should be skipped (lazy loading)",
		},
		{
			name: "vendored file without .go - should NOT skip",
			writer: &Writer{
				forceRegenerate: false,
				vendorDir:       vendorDir,
			},
			goFilePath:  vendoredMissingGoFile,
			clawPath:    vendoredClawPath,
			wantSkip:    false,
			description: "Vendored file without .go must be generated (first time)",
		},
		{
			name: "vendorDir set but forceRegenerate=true - should NOT skip",
			writer: &Writer{
				forceRegenerate: true,
				vendorDir:       vendorDir,
			},
			goFilePath:  vendoredGoFile,
			clawPath:    vendoredClawPath,
			wantSkip:    false,
			description: "Force regenerate overrides all other conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.writer.shouldSkipGeneration(tt.goFilePath, tt.clawPath)

			if got != tt.wantSkip {
				t.Errorf("shouldSkipGeneration() = %v, want %v\nDescription: %s",
					got, tt.wantSkip, tt.description)
			}
		})
	}
}

func TestIsInVendor(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	localDir := filepath.Join(tmpDir, "local")

	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("failed to create local dir: %v", err)
	}

	vendoredFile := filepath.Join(vendorDir, "github.com", "foo", "bar", "test.claw")
	localFile := filepath.Join(localDir, "test.claw")

	tests := []struct {
		name        string
		path        string
		vendorDir   string
		want        bool
		description string
	}{
		{
			name:        "file in vendor directory",
			path:        vendoredFile,
			vendorDir:   vendorDir,
			want:        true,
			description: "File path under vendor directory should be detected",
		},
		{
			name:        "file outside vendor directory",
			path:        localFile,
			vendorDir:   vendorDir,
			want:        false,
			description: "File not under vendor directory should not be detected as vendored",
		},
		{
			name:        "empty vendor directory",
			path:        vendoredFile,
			vendorDir:   "",
			want:        false,
			description: "Empty vendor directory means no lazy loading",
		},
		{
			name:        "vendor directory itself",
			path:        vendorDir,
			vendorDir:   vendorDir,
			want:        true,
			description: "The vendor directory itself should be considered 'in vendor'",
		},
		{
			name:        "parent of vendor directory",
			path:        tmpDir,
			vendorDir:   vendorDir,
			want:        false,
			description: "Parent of vendor directory should not be considered vendored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInVendor(tt.path, tt.vendorDir)

			if got != tt.want {
				t.Errorf("isInVendor() = %v, want %v\nDescription: %s\nPath: %s\nVendorDir: %s",
					got, tt.want, tt.description, tt.path, tt.vendorDir)
			}
		})
	}
}

// TestShouldSkipGenerationIntegration tests the integration with the intended workflow
func TestShouldSkipGenerationIntegration(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	pkgDir := filepath.Join(vendorDir, "github.com", "test", "pkg")

	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create package dir: %v", err)
	}

	clawPath := filepath.Join(pkgDir, "test.claw")
	goPath := filepath.Join(pkgDir, "test.go")

	// Scenario 1: First compilation (no .go file exists)
	t.Run("first compilation - generate .go file", func(t *testing.T) {
		w := &Writer{
			forceRegenerate: false,
			vendorDir:       vendorDir,
		}

		// .go file doesn't exist yet
		if _, err := os.Stat(goPath); !os.IsNotExist(err) {
			t.Fatalf("test setup error: .go file should not exist")
		}

		// Should NOT skip - need to generate
		if w.shouldSkipGeneration(goPath, clawPath) {
			t.Error("First compilation should NOT skip generation")
		}

		// Simulate generating the file
		if err := os.WriteFile(goPath, []byte("package test"), 0644); err != nil {
			t.Fatalf("failed to create .go file: %v", err)
		}
	})

	// Scenario 2: Second compilation (lazy loading - skip)
	t.Run("second compilation - skip generation (lazy load)", func(t *testing.T) {
		w := &Writer{
			forceRegenerate: false,
			vendorDir:       vendorDir,
		}

		// .go file now exists from previous step
		if _, err := os.Stat(goPath); os.IsNotExist(err) {
			t.Fatalf("test setup error: .go file should exist")
		}

		// Should skip - file already exists
		if !w.shouldSkipGeneration(goPath, clawPath) {
			t.Error("Second compilation should skip generation (lazy loading)")
		}
	})

	// Scenario 3: Explicit clawc get (force regenerate)
	t.Run("clawc get - force regenerate", func(t *testing.T) {
		w := &Writer{
			forceRegenerate: true,
			vendorDir:       vendorDir,
		}

		// .go file exists but force regenerate is true
		if _, err := os.Stat(goPath); os.IsNotExist(err) {
			t.Fatalf("test setup error: .go file should exist")
		}

		// Should NOT skip - force regenerate overrides
		if w.shouldSkipGeneration(goPath, clawPath) {
			t.Error("clawc get with force regenerate should NOT skip generation")
		}
	})
}
