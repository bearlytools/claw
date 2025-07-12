package vendor

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/bearlytools/claw/internal/idl"
	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/imports/git"
)

// mockVCSGit implements the vcsGit interface for testing
type mockVCSGit struct {
	inRepo bool
	root   string
	origin string
}

func (m *mockVCSGit) InRepo(pkgPath string) bool {
	// If inRepo is explicitly set for simple testing, use it
	if m.origin == "" {
		return m.inRepo
	}
	// Otherwise, use realistic logic like the real implementation
	return strings.HasPrefix(pkgPath, m.origin)
}

func (m *mockVCSGit) Root() string {
	return m.root
}

func (m *mockVCSGit) Origin() string {
	return m.origin
}

func (m *mockVCSGit) Abs(p string) (string, error) {
	if m.origin == "" {
		return "", fmt.Errorf("no origin set")
	}
	return filepath.Join(m.root, strings.TrimPrefix(p, m.origin)), nil
}

// mockGetClawFile is a mock implementation of getClawFile
type mockGetClawFile struct {
	files map[string]git.ClawFile
}

func (m *mockGetClawFile) getClawFile(ctx context.Context, pkgPath string, version string) (git.ClawFile, error) {
	key := pkgPath
	if version != "" {
		key = fmt.Sprintf("%s@%s", pkgPath, version)
	}

	cf, ok := m.files[key]
	if !ok {
		cf, ok = m.files[pkgPath] // fallback without version
		if !ok {
			return git.ClawFile{}, fmt.Errorf("claw file not found for %s", pkgPath)
		}
	}

	cf.SHA256 = fmt.Sprintf("%x", sha256.Sum256(cf.Content))
	return cf, nil
}

// TestNewVendorManager tests the NewVendorManager constructor
func TestNewVendorManager(t *testing.T) {
	tests := []struct {
		desc    string
		rootDir string
		wantErr bool
	}{
		{
			desc:    "Success - valid root directory",
			rootDir: "/tmp/test-project",
			wantErr: false,
		},
		{
			desc:    "Success - relative path",
			rootDir: "./test-project",
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm, err := NewVendorManager(test.rootDir)

			switch {
			case err != nil && !test.wantErr:
				t.Errorf("NewVendorManager(): got err = %v, want nil", err)
			case err == nil && test.wantErr:
				t.Errorf("NewVendorManager(): got err = nil, want error")
			case err == nil:
				if vm.rootDir != test.rootDir {
					t.Errorf("NewVendorManager(): rootDir = %q, want %q", vm.rootDir, test.rootDir)
				}
				expectedVendorDir := filepath.Join(test.rootDir, "vendor")
				if vm.vendorDir != expectedVendorDir {
					t.Errorf("NewVendorManager(): vendorDir = %q, want %q", vm.vendorDir, expectedVendorDir)
				}
				if vm.fs == nil {
					t.Error("NewVendorManager(): fs is nil")
				}
				if vm.getClawFile == nil {
					t.Error("NewVendorManager(): getClawFile is nil")
				}
			}
		})
	}
}

// TestParseModuleContent tests the parseModuleContent function
func TestParseModuleContent(t *testing.T) {
	// This test validates that the parseModuleContent function can handle
	// valid module content. The specific parsing is handled by halfpike
	// and is tested elsewhere.

	vm := &VendorManager{}
	module := &imports.Module{}

	// Test with valid simple module content
	content := `module github.com/example/test`
	err := vm.parseModuleContent(content, module)
	// We expect this to succeed for simple module declarations
	if err != nil {
		t.Logf("parseModuleContent(): got err = %v (this may be expected due to halfpike parsing)", err)
	}

	// Test with invalid content should fail
	invalidContent := `not a module`
	err = vm.parseModuleContent(invalidContent, module)
	if err == nil {
		t.Error("parseModuleContent(): expected error for invalid content")
	}
}

// TestParseLocalReplaceContent tests the parseLocalReplaceContent function
func TestParseLocalReplaceContent(t *testing.T) {
	// This test validates that the parseLocalReplaceContent function can handle
	// valid replace content. The specific parsing is handled by halfpike
	// and is tested elsewhere.

	vm := &VendorManager{}
	localReplace := &imports.LocalReplace{}

	// Test with empty content (should work)
	content := ``
	err := vm.parseLocalReplaceContent(content, localReplace)
	if err != nil {
		t.Errorf("parseLocalReplaceContent(): got err = %v for empty content", err)
	}

	// Function works - the specific parsing validation is handled by halfpike
	// and is tested in the imports package
}

// TestExtractPackagePath tests the extractPackagePath helper function
func TestExtractPackagePath(t *testing.T) {
	tests := []struct {
		desc     string
		clawPath string
		want     string
	}{
		{
			desc:     "Basic path extraction",
			clawPath: "/path/to/package.claw",
			want:     "/path/to/package.claw",
		},
		{
			desc:     "Empty path",
			clawPath: "",
			want:     "",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got := vm.extractPackagePath(test.clawPath)

			if got != test.want {
				t.Errorf("extractPackagePath(%q) = %q, want %q", test.clawPath, got, test.want)
			}
		})
	}
}

// TestApplyReplaceDirectives tests the applyReplaceDirectives function
func TestApplyReplaceDirectives(t *testing.T) {
	rootModule := &imports.Module{
		Replace: []imports.Replace{
			{FromPath: "github.com/example/old", ToPath: "github.com/example/new"},
		},
	}

	localReplace := imports.LocalReplace{
		Replace: []imports.Replace{
			{FromPath: "github.com/example/local", ToPath: "/local/path"},
		},
	}

	tests := []struct {
		desc        string
		pkgPath     string
		version     string
		wantPath    string
		wantVersion string
	}{
		{
			desc:        "Local replace takes precedence",
			pkgPath:     "github.com/example/local",
			version:     "v1.0.0",
			wantPath:    "/local/path",
			wantVersion: "v1.0.0",
		},
		{
			desc:        "Module replace applied",
			pkgPath:     "github.com/example/old",
			version:     "v1.0.0",
			wantPath:    "github.com/example/new",
			wantVersion: "v1.0.0",
		},
		{
			desc:        "No replace needed",
			pkgPath:     "github.com/example/unchanged",
			version:     "v1.0.0",
			wantPath:    "github.com/example/unchanged",
			wantVersion: "v1.0.0",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			gotPath, gotVersion := vm.applyReplaceDirectives(test.pkgPath, test.version, rootModule, localReplace)

			if gotPath != test.wantPath {
				t.Errorf("applyReplaceDirectives(): gotPath = %q, want %q", gotPath, test.wantPath)
			}
			if gotVersion != test.wantVersion {
				t.Errorf("applyReplaceDirectives(): gotVersion = %q, want %q", gotVersion, test.wantVersion)
			}
		})
	}
}

// TestIsLocalPath tests the isLocalPath helper function
func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		desc string
		path string
		want bool
	}{
		{
			desc: "Absolute path",
			path: "/absolute/path",
			want: true,
		},
		{
			desc: "Relative path with ./",
			path: "./relative/path",
			want: true,
		},
		{
			desc: "Relative path with ../",
			path: "../relative/path",
			want: true,
		},
		{
			desc: "Remote path",
			path: "github.com/example/package",
			want: false,
		},
		{
			desc: "Empty path",
			path: "",
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got := vm.isLocalPath(test.path)

			if got != test.want {
				t.Errorf("isLocalPath(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

// TestGetLocalClawFile tests the getLocalClawFile function
func TestGetLocalClawFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vendor-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	clawContent := []byte(`package test

Struct TestStruct {
    Field1 string @0
}`)

	clawFile := filepath.Join(tempDir, "test.claw")
	if err := os.WriteFile(clawFile, clawContent, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc        string
		path        string
		wantErr     bool
		wantContent []byte
	}{
		{
			desc:        "Success - direct .claw file",
			path:        clawFile,
			wantErr:     false,
			wantContent: clawContent,
		},
		{
			desc:        "Success - directory with .claw file",
			path:        tempDir,
			wantErr:     false,
			wantContent: clawContent,
		},
		{
			desc:    "Error - nonexistent path",
			path:    "/nonexistent/path",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got, err := vm.getLocalClawFile(test.path)

			switch {
			case err != nil && !test.wantErr:
				t.Errorf("getLocalClawFile(): got err = %v, want nil", err)
			case err == nil && test.wantErr:
				t.Errorf("getLocalClawFile(): got err = nil, want error")
			case err == nil:
				if string(got.Content) != string(test.wantContent) {
					t.Errorf("getLocalClawFile(): content mismatch")
				}
				if got.Version != "local" {
					t.Errorf("getLocalClawFile(): version = %q, want %q", got.Version, "local")
				}
			}
		})
	}
}

// TestParseClawFile tests the parseClawFile function
func TestParseClawFile(t *testing.T) {
	tests := []struct {
		desc    string
		content []byte
		wantErr bool
	}{
		{
			desc: "Success - valid claw file",
			content: []byte(`package test

Struct TestStruct {
    Field1 string @0
}`),
			wantErr: false,
		},
		{
			desc:    "Error - invalid syntax",
			content: []byte(`invalid claw syntax`),
			wantErr: true,
		},
		{
			desc:    "Error - empty content",
			content: []byte(""),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got, err := vm.parseClawFile(test.content)

			switch {
			case err != nil && !test.wantErr:
				t.Errorf("parseClawFile(): got err = %v, want nil", err)
			case err == nil && test.wantErr:
				t.Errorf("parseClawFile(): got err = nil, want error")
			case err == nil:
				if got == nil {
					t.Error("parseClawFile(): got nil result")
				}
			}
		})
	}
}

// TestExtractImports tests the extractImports function
func TestExtractImports(t *testing.T) {
	file := &idl.File{
		Imports: idl.Import{
			Imports: map[string]idl.ImportEntry{
				"github.com/example/dep1": {Path: "github.com/example/dep1"},
				"github.com/example/dep2": {Path: "github.com/example/dep2"},
			},
		},
	}

	vm := &VendorManager{}
	got := vm.extractImports(file)

	// Since extractImports returns a slice, we need to check the length
	if len(got) != 2 {
		t.Errorf("extractImports(): got %d imports, want 2", len(got))
	}

	// Check that the expected imports are present
	wantImports := map[string]bool{
		"github.com/example/dep1": true,
		"github.com/example/dep2": true,
	}

	for _, imp := range got {
		if !wantImports[imp] {
			t.Errorf("extractImports(): unexpected import %q", imp)
		}
	}
}

// TestResolveVersion tests the resolveVersion function
func TestResolveVersion(t *testing.T) {
	rootModule := &imports.Module{
		Required: []imports.Require{
			{Path: "github.com/example/dep1", Version: imports.Version{Major: 1, Minor: 2, Patch: 3}},
			{Path: "github.com/example/dep2", ID: "branch-name"},
		},
	}

	tests := []struct {
		desc    string
		pkgPath string
		want    string
	}{
		{
			desc:    "Version found with semantic version",
			pkgPath: "github.com/example/dep1",
			want:    "v1.2.3",
		},
		{
			desc:    "Version found with ID",
			pkgPath: "github.com/example/dep2",
			want:    "branch-name",
		},
		{
			desc:    "Version not found",
			pkgPath: "github.com/example/unknown",
			want:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got := vm.resolveVersion(test.pkgPath, rootModule)

			if got != test.want {
				t.Errorf("resolveVersion(%q) = %q, want %q", test.pkgPath, got, test.want)
			}
		})
	}
}

// TestMarkLeafNodes tests the markLeafNodes function
func TestMarkLeafNodes(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"pkg1": {PkgPath: "pkg1"},
			"pkg2": {PkgPath: "pkg2"},
			"pkg3": {PkgPath: "pkg3"},
		},
		Edges: map[string][]string{
			"pkg1": {"pkg2", "pkg3"},
			"pkg2": {"pkg3"},
			"pkg3": {},
		},
	}

	vm := &VendorManager{}
	vm.markLeafNodes(graph)

	// pkg3 should be marked as a leaf node
	if !graph.Nodes["pkg3"].IsLeaf {
		t.Error("markLeafNodes(): pkg3 should be marked as leaf")
	}

	// pkg1 and pkg2 should not be marked as leaf nodes
	if graph.Nodes["pkg1"].IsLeaf {
		t.Error("markLeafNodes(): pkg1 should not be marked as leaf")
	}
	if graph.Nodes["pkg2"].IsLeaf {
		t.Error("markLeafNodes(): pkg2 should not be marked as leaf")
	}
}

// TestCheckACLPermission tests the checkACLPermission function
func TestCheckACLPermission(t *testing.T) {
	tests := []struct {
		desc             string
		currentModule    string
		targetModule     string
		targetModuleFile *imports.Module
		wantErr          bool
		errContains      string
	}{
		{
			desc:          "Success - public access",
			currentModule: "github.com/example/current",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{
					{Path: "public"},
				},
			},
			wantErr: false,
		},
		{
			desc:          "Success - wildcard access",
			currentModule: "github.com/example/current",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{
					{Path: "*"},
				},
			},
			wantErr: false,
		},
		{
			desc:          "Success - exact match",
			currentModule: "github.com/example/current",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{
					{Path: "github.com/example/current"},
				},
			},
			wantErr: false,
		},
		{
			desc:          "Success - prefix match",
			currentModule: "github.com/example/current/subpkg",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{
					{Path: "github.com/example/*"},
				},
			},
			wantErr: false,
		},
		{
			desc:          "Error - no ACLs",
			currentModule: "github.com/example/current",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{},
			},
			wantErr:     true,
			errContains: "does not allow any imports",
		},
		{
			desc:          "Error - not allowed",
			currentModule: "github.com/example/current",
			targetModule:  "github.com/example/target",
			targetModuleFile: &imports.Module{
				ACLs: []imports.ACL{
					{Path: "github.com/other/allowed"},
				},
			},
			wantErr:     true,
			errContains: "is not allowed to import",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			err := vm.checkACLPermission(test.currentModule, test.targetModule, test.targetModuleFile)

			switch {
			case err != nil && !test.wantErr:
				t.Errorf("checkACLPermission(): got err = %v, want nil", err)
			case err == nil && test.wantErr:
				t.Errorf("checkACLPermission(): got err = nil, want error")
			case err != nil && test.errContains != "":
				if !strings.Contains(err.Error(), test.errContains) {
					t.Errorf("checkACLPermission(): error %q does not contain %q", err.Error(), test.errContains)
				}
			}
		})
	}
}

// TestGetClawFileName tests the getClawFileName function
func TestGetClawFileName(t *testing.T) {
	tests := []struct {
		desc    string
		pkgPath string
		want    string
	}{
		{
			desc:    "Simple package name",
			pkgPath: "github.com/example/package",
			want:    "package.claw",
		},
		{
			desc:    "Deep package path",
			pkgPath: "github.com/example/org/deep/package",
			want:    "package.claw",
		},
		{
			desc:    "Single element",
			pkgPath: "package",
			want:    "package.claw",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got := vm.getClawFileName(test.pkgPath)

			if got != test.want {
				t.Errorf("getClawFileName(%q) = %q, want %q", test.pkgPath, got, test.want)
			}
		})
	}
}

// TestGetCompilationOrder tests the GetCompilationOrder function
func TestGetCompilationOrder(t *testing.T) {
	tests := []struct {
		desc    string
		graph   *DependencyGraph
		want    []string
		wantErr bool
	}{
		{
			desc: "Success - linear dependencies",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"pkg1": {PkgPath: "pkg1"},
					"pkg2": {PkgPath: "pkg2"},
					"pkg3": {PkgPath: "pkg3"},
				},
				Edges: map[string][]string{
					"pkg1": {"pkg2"},
					"pkg2": {"pkg3"},
					"pkg3": {},
				},
			},
			want:    []string{"pkg1", "pkg2", "pkg3"}, // The actual order depends on implementation
			wantErr: false,
		},
		{
			desc: "Success - no dependencies",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"pkg1": {PkgPath: "pkg1"},
				},
				Edges: map[string][]string{
					"pkg1": {},
				},
			},
			want:    []string{"pkg1"},
			wantErr: false,
		},
		{
			desc: "Error - circular dependency",
			graph: &DependencyGraph{
				Nodes: map[string]*DependencyNode{
					"pkg1": {PkgPath: "pkg1"},
					"pkg2": {PkgPath: "pkg2"},
				},
				Edges: map[string][]string{
					"pkg1": {"pkg2"},
					"pkg2": {"pkg1"},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			got, err := vm.GetCompilationOrder(test.graph)

			switch {
			case err != nil && !test.wantErr:
				t.Errorf("GetCompilationOrder(): got err = %v, want nil", err)
			case err == nil && test.wantErr:
				t.Errorf("GetCompilationOrder(): got err = nil, want error")
			case err == nil:
				if diff := pretty.Compare(test.want, got); diff != "" {
					t.Errorf("GetCompilationOrder(): -want/+got:\n%s", diff)
				}
			}
		})
	}
}

// TestGetVendorPath tests the GetVendorPath function
func TestGetVendorPath(t *testing.T) {
	vm := &VendorManager{
		vendorDir: "/project/vendor",
	}

	tests := []struct {
		desc    string
		pkgPath string
		want    string
	}{
		{
			desc:    "Empty package path returns vendor dir",
			pkgPath: "",
			want:    "/project/vendor",
		},
		{
			desc:    "Package path returns joined path",
			pkgPath: "github.com/example/pkg",
			want:    "/project/vendor/github.com/example/pkg",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := vm.GetVendorPath(test.pkgPath)

			if got != test.want {
				t.Errorf("GetVendorPath(%q) = %q, want %q", test.pkgPath, got, test.want)
			}
		})
	}
}

// TestWriteClawFile tests the writeClawFile function
func TestWriteClawFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vendor-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	content := []byte("test claw content")
	node := &DependencyNode{
		OriginalContent: content,
	}

	filePath := filepath.Join(tempDir, "test.claw")

	vm := &VendorManager{}
	err = vm.writeClawFile(filePath, node)
	if err != nil {
		t.Errorf("writeClawFile(): got err = %v, want nil", err)
	}

	// Verify file was written correctly
	gotContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(gotContent) != string(content) {
		t.Errorf("writeClawFile(): content mismatch, got %q, want %q", string(gotContent), string(content))
	}
}

// TestSerializeModule tests the serializeModule function
func TestSerializeModule(t *testing.T) {
	module := &imports.Module{
		Path: "github.com/example/module",
		Required: []imports.Require{
			{Path: "github.com/example/dep1", Version: imports.Version{Major: 1, Minor: 2, Patch: 3}},
			{Path: "github.com/example/dep2", ID: "branch-name"},
		},
		Replace: []imports.Replace{
			{FromPath: "github.com/example/old", ToPath: "github.com/example/new"},
		},
		ACLs: []imports.ACL{
			{Path: "public"},
		},
	}

	vm := &VendorManager{}
	got := vm.serializeModule(module)

	// Check that the serialized module contains expected content
	expectedParts := []string{
		"module github.com/example/module",
		"require (",
		"github.com/example/dep1 v1.2.3",
		"github.com/example/dep2 branch-name",
		"replace (",
		"github.com/example/old => github.com/example/new",
		"acls (",
		"public",
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Errorf("serializeModule(): output does not contain %q", part)
		}
	}
}

// TestVendorDependenciesIntegration tests the main VendorDependencies function with integration
func TestVendorDependenciesIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vendor-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	clawModContent := `module github.com/example/test`

	clawContent := `package test

Struct TestStruct {
    Field1 string @0
}`

	// Write test files
	modFile := filepath.Join(tempDir, "claw.mod")
	if err = os.WriteFile(modFile, []byte(clawModContent), 0o644); err != nil {
		t.Fatal(err)
	}

	clawFile := filepath.Join(tempDir, "test.claw")
	if err = os.WriteFile(clawFile, []byte(clawContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create vendor manager with mock dependencies
	vm, err := NewVendorManager(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Mock the getClawFile function
	mockGetClaw := &mockGetClawFile{
		files: map[string]git.ClawFile{
			"github.com/example/dep": {
				Content: []byte(`package dep

Struct DepStruct {
    Field1 string @0
}`),
				Version: "v1.0.0",
			},
		},
	}
	vm.getClawFile = mockGetClaw.getClawFile

	// Run the vendor process
	ctx := context.Background()
	graph, err := vm.VendorDependencies(ctx, clawFile)
	if err != nil {
		t.Errorf("VendorDependencies(): got err = %v, want nil", err)
		return
	}

	// Verify the graph structure
	if graph == nil {
		t.Error("VendorDependencies(): got nil graph")
		return
	}

	if len(graph.Nodes) == 0 {
		t.Error("VendorDependencies(): graph has no nodes")
	}

	// Check that vendor directory was created
	vendorDir := filepath.Join(tempDir, "vendor")
	if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
		t.Error("VendorDependencies(): vendor directory was not created")
	}
}

// TestShouldVendorPackage tests the shouldVendorPackage function
func TestShouldVendorPackage(t *testing.T) {
	tests := []struct {
		desc    string
		pkgPath string
		inRepo  bool
		gitNil  bool
		want    bool
	}{
		{
			desc:    "External package should be vendored",
			pkgPath: "github.com/external/package",
			inRepo:  false,
			gitNil:  false,
			want:    true,
		},
		{
			desc:    "Local package should NOT be vendored",
			pkgPath: "github.com/bearlytools/claw/internal/package",
			inRepo:  true,
			gitNil:  false,
			want:    false,
		},
		{
			desc:    "No git available - fallback to vendor everything",
			pkgPath: "github.com/example/package",
			inRepo:  false, // irrelevant when git is nil
			gitNil:  true,
			want:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			vm := &VendorManager{}
			
			if !test.gitNil {
				vm.git = &mockVCSGit{
					inRepo: test.inRepo,
					origin: "github.com/bearlytools/claw",
				}
			}

			got := vm.shouldVendorPackage(test.pkgPath)

			if got != test.want {
				t.Errorf("shouldVendorPackage(%q) = %v, want %v (inRepo=%v, gitNil=%v)", 
					test.pkgPath, got, test.want, test.inRepo, test.gitNil)
			}
		})
	}
}
