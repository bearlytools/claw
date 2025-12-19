package main

import (
	"testing"
)

func TestParsePackageSpec(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPkg     string
		wantVersion string
		wantErr     bool
		description string
	}{
		{
			name:        "valid package with semantic version",
			input:       "github.com/foo/bar@v1.2.3",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "v1.2.3",
			wantErr:     false,
			description: "Standard format with package and semantic version",
		},
		{
			name:        "valid package without version",
			input:       "github.com/foo/bar",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "",
			wantErr:     false,
			description: "Package only, no version specified (should default to latest)",
		},
		{
			name:        "valid package with @latest",
			input:       "github.com/foo/bar@latest",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "latest",
			wantErr:     false,
			description: "Explicit @latest syntax",
		},
		{
			name:        "valid package with commit hash",
			input:       "github.com/foo/bar@abc123def456",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "abc123def456",
			wantErr:     false,
			description: "Package with commit hash as version",
		},
		{
			name:        "invalid - empty string",
			input:       "",
			wantPkg:     "",
			wantVersion: "",
			wantErr:     true,
			description: "Empty input should error - user must provide package",
		},
		{
			name:        "invalid - only version no package",
			input:       "@v1.2.3",
			wantPkg:     "",
			wantVersion: "",
			wantErr:     true,
			description: "Version without package is invalid",
		},
		{
			name:        "invalid - only @ symbol",
			input:       "@",
			wantPkg:     "",
			wantVersion: "",
			wantErr:     true,
			description: "Just @ with no package or version should error",
		},
		{
			name:        "edge case - whitespace around input",
			input:       "  github.com/foo/bar@v1.2.3  ",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "v1.2.3",
			wantErr:     false,
			description: "Whitespace should be trimmed from input",
		},
		{
			name:        "edge case - whitespace around components",
			input:       "github.com/foo/bar  @  v1.2.3",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "v1.2.3",
			wantErr:     false,
			description: "Whitespace around @ and version should be trimmed",
		},
		{
			name:        "edge case - empty version after @",
			input:       "github.com/foo/bar@",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "",
			wantErr:     false,
			description: "Empty version after @ is treated as no version (same as latest)",
		},
		{
			name:        "edge case - only whitespace",
			input:       "   ",
			wantPkg:     "",
			wantVersion: "",
			wantErr:     true,
			description: "Whitespace-only input should error after trimming",
		},
		{
			name:        "valid - deep nested package path",
			input:       "github.com/org/repo/internal/pkg/subpkg@v2.0.0",
			wantPkg:     "github.com/org/repo/internal/pkg/subpkg",
			wantVersion: "v2.0.0",
			wantErr:     false,
			description: "Deep package paths should be handled correctly",
		},
		{
			name:        "valid - version with multiple parts",
			input:       "github.com/foo/bar@v1.2.3-beta.1+build.456",
			wantPkg:     "github.com/foo/bar",
			wantVersion: "v1.2.3-beta.1+build.456",
			wantErr:     false,
			description: "Semantic version with pre-release and build metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, version, err := parsePackageSpec(tt.input)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("parsePackageSpec() expected error for input %q, but got none", tt.input)
				}
				return
			}

			// Check no unexpected error
			if err != nil {
				t.Errorf("parsePackageSpec() unexpected error: %v", err)
				return
			}

			// Check package
			if pkg != tt.wantPkg {
				t.Errorf("parsePackageSpec() package = %q, want %q", pkg, tt.wantPkg)
			}

			// Check version
			if version != tt.wantVersion {
				t.Errorf("parsePackageSpec() version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}
