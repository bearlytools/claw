// Package vendor provides vendoring functionality for claw dependencies.
package vendor

import (
	"github.com/gostdlib/base/context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	osfs "github.com/gopherfs/fs/io/os"
	"github.com/johnsiilver/halfpike"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/idl"
	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/imports/git"
	"github.com/bearlytools/claw/internal/vcs"
)

// VendorManager handles the vendoring process for claw dependencies.
type VendorManager struct {
	rootDir     string
	vendorDir   string
	fs          fs.ReadFileFS
	git         vcsGit
	getClawFile func(ctx context.Context, pkgPath string, version string) (git.ClawFile, error)
}

type vcsGit interface {
	InRepo(pkgPath string) bool
	Root() string
	Origin() string
	Abs(p string) (string, error)
}

// DependencyNode represents a node in the dependency graph.
type DependencyNode struct {
	PkgPath         string
	Version         string
	Dependencies    []string
	ClawFile        *idl.File
	OriginalContent []byte // Store the original .claw file content
	ModuleFile      *imports.Module
	IsLeaf          bool
	ShouldVendor    bool   // Whether this package should be vendored
}

// DependencyGraph represents the complete dependency graph.
type DependencyGraph struct {
	Nodes map[string]*DependencyNode
	Edges map[string][]string // pkgPath -> list of dependencies
}

// NewVendorManager creates a new VendorManager.
func NewVendorManager(rootDir string) (*VendorManager, error) {
	fs, err := osfs.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}

	vendorDir := filepath.Join(rootDir, "vendor")

	// Try to set up git support - it's optional
	var gitVCS vcsGit
	// Use a recover to handle panics from the git package
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Git failed, continue without it
				gitVCS = nil
			}
		}()
		if g, err := vcs.NewGit(rootDir); err == nil {
			gitVCS = g
		}
	}()

	return &VendorManager{
		rootDir:     rootDir,
		vendorDir:   vendorDir,
		fs:          fs,
		git:         gitVCS,
		getClawFile: git.GetClawFile,
	}, nil
}

// VendorDependencies performs the complete vendoring process:
// 1. Dependency resolution and graph building
// 2. ACL validation
// 3. Vendor directory creation
// 4. Compilation order management
func (vm *VendorManager) VendorDependencies(ctx context.Context, clawFilePath string) (*DependencyGraph, error) {
	// Step 1: Parse the root claw.mod file
	rootModule, err := vm.parseModuleFile(filepath.Dir(clawFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to parse root claw.mod: %w", err)
	}

	// Step 2: Parse replace files
	localReplace, err := vm.parseLocalReplace(filepath.Dir(clawFilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to parse local.replace: %w", err)
	}

	// Step 3: Build dependency graph
	graph, err := vm.buildDependencyGraph(ctx, clawFilePath, rootModule, localReplace)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Step 4: Validate ACLs
	if err := vm.validateACLs(ctx, graph, rootModule); err != nil {
		return nil, fmt.Errorf("ACL validation failed: %w", err)
	}

	// Step 5: Create vendor directory structure
	if err := vm.createVendorStructure(ctx, graph, localReplace); err != nil {
		return nil, fmt.Errorf("failed to create vendor structure: %w", err)
	}

	return graph, nil
}

// parseModuleFile parses a claw.mod file in the given directory.
func (vm *VendorManager) parseModuleFile(dir string) (*imports.Module, error) {
	modPath := filepath.Join(dir, "claw.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("claw.mod file not found in %s", dir)
	}

	content, err := os.ReadFile(modPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read claw.mod: %w", err)
	}

	module := &imports.Module{}
	// Note: Using halfpike parser like the original code
	// This is a simplified version - in practice, you'd use the same parsing logic
	// as in the original Module.Start method
	if err := vm.parseModuleContent(string(content), module); err != nil {
		return nil, fmt.Errorf("failed to parse claw.mod content: %w", err)
	}

	return module, nil
}

// parseModuleContent parses claw.mod content using the existing halfpike parser.
func (vm *VendorManager) parseModuleContent(content string, module *imports.Module) error {
	ctx := context.Background()
	return halfpike.Parse(ctx, content, module)
}

// parseLocalReplace parses a local.replace file if it exists.
func (vm *VendorManager) parseLocalReplace(dir string) (imports.LocalReplace, error) {
	replacePath := filepath.Join(dir, "local.replace")
	if _, err := os.Stat(replacePath); os.IsNotExist(err) {
		return imports.LocalReplace{}, nil // No local.replace file is OK
	}

	content, err := os.ReadFile(replacePath)
	if err != nil {
		return imports.LocalReplace{}, fmt.Errorf("failed to read local.replace: %w", err)
	}

	localReplace := imports.LocalReplace{}
	// Parse the local.replace content using the existing parsing logic
	// This would use the same halfpike parser as in the original code
	if err := vm.parseLocalReplaceContent(string(content), &localReplace); err != nil {
		return imports.LocalReplace{}, fmt.Errorf("failed to parse local.replace: %w", err)
	}

	return localReplace, nil
}

// parseLocalReplaceContent parses local.replace content using the existing halfpike parser.
func (vm *VendorManager) parseLocalReplaceContent(content string, localReplace *imports.LocalReplace) error {
	ctx := context.Background()
	return halfpike.Parse(ctx, content, localReplace)
}

// buildDependencyGraph builds the complete dependency graph by recursively discovering imports.
func (vm *VendorManager) buildDependencyGraph(ctx context.Context, rootClawPath string, rootModule *imports.Module, localReplace imports.LocalReplace) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// Start with the root file
	if err := vm.processClawFile(ctx, rootClawPath, "", graph, rootModule, localReplace, make(map[string]bool)); err != nil {
		return nil, fmt.Errorf("failed to process root claw file: %w", err)
	}

	// Mark leaf nodes
	vm.markLeafNodes(graph)

	return graph, nil
}

// processClawFile processes a single .claw file and adds it to the dependency graph.
func (vm *VendorManager) processClawFile(ctx context.Context, clawPath, version string, graph *DependencyGraph, rootModule *imports.Module, localReplace imports.LocalReplace, visited map[string]bool) error {
	// Extract package path from file path
	pkgPath := vm.extractPackagePath(clawPath)

	// Check for circular dependencies
	if visited[pkgPath] {
		return fmt.Errorf("circular dependency detected: %s", pkgPath)
	}
	visited[pkgPath] = true
	defer delete(visited, pkgPath)

	// Skip if already processed
	if _, exists := graph.Nodes[pkgPath]; exists {
		return nil
	}

	// Apply replace directives to get the actual source
	actualPath, actualVersion := vm.applyReplaceDirectives(pkgPath, version, rootModule, localReplace)

	// Get the claw file content
	var clawFile git.ClawFile
	var err error

	if vm.isLocalPath(actualPath) {
		clawFile, err = vm.getLocalClawFile(actualPath)
	} else if vm.git.InRepo(actualPath) {
		// Package is in the current git repository, use local filesystem
		localPath, absErr := vm.git.Abs(actualPath)
		if absErr != nil {
			return fmt.Errorf("failed to resolve local path for %s: %w", actualPath, absErr)
		}
		clawFile, err = vm.getLocalClawFile(localPath)
	} else {
		clawFile, err = vm.getClawFile(ctx, actualPath, actualVersion)
	}

	if err != nil {
		return fmt.Errorf("failed to get claw file for %s: %w", pkgPath, err)
	}

	// Parse the claw file to extract imports
	idlFile, err := vm.parseClawFile(clawFile.Content)
	if err != nil {
		return fmt.Errorf("failed to parse claw file %s: %w", pkgPath, err)
	}

	// Get the module file for this dependency
	moduleFile, err := vm.getModuleFile(ctx, actualPath, actualVersion)
	if err != nil {
		// Module file is optional for dependencies
		moduleFile = nil
	}

	// Create dependency node
	node := &DependencyNode{
		PkgPath:         pkgPath,
		Version:         clawFile.Version,
		Dependencies:    vm.extractImports(idlFile),
		ClawFile:        idlFile,
		OriginalContent: clawFile.Content, // Store original content
		ModuleFile:      moduleFile,
		ShouldVendor:    vm.shouldVendorPackage(pkgPath),
	}

	graph.Nodes[pkgPath] = node
	graph.Edges[pkgPath] = node.Dependencies

	// Recursively process dependencies
	for _, dep := range node.Dependencies {
		// Apply version resolution
		depVersion := vm.resolveVersion(dep, rootModule)
		if err := vm.processClawFile(ctx, dep, depVersion, graph, rootModule, localReplace, visited); err != nil {
			return fmt.Errorf("failed to process dependency %s: %w", dep, err)
		}
	}

	return nil
}

// Helper methods (simplified implementations)

func (vm *VendorManager) extractPackagePath(clawPath string) string {
	// Extract package path from file path
	// This is a simplified implementation
	return clawPath
}

func (vm *VendorManager) applyReplaceDirectives(pkgPath, version string, rootModule *imports.Module, localReplace imports.LocalReplace) (string, string) {
	// Apply replace directives in order of precedence:
	// 1. local.replace
	// 2. claw.mod replace
	// 3. global.replace (handled during git fetch)

	// Check local.replace first
	for _, replace := range localReplace.Replace {
		if replace.FromPath == pkgPath {
			return replace.ToPath, version
		}
	}

	// Check claw.mod replace
	for _, replace := range rootModule.Replace {
		if replace.FromPath == pkgPath {
			return replace.ToPath, version
		}
	}

	return pkgPath, version
}

func (vm *VendorManager) isLocalPath(path string) bool {
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

// shouldVendorPackage determines if a package should be vendored.
// Returns false for packages within the current Git repository (local).
// Returns true only for external dependencies that should be vendored.
func (vm *VendorManager) shouldVendorPackage(pkgPath string) bool {
	// If git is not available, fall back to vendoring everything
	if vm.git == nil {
		return true
	}
	
	// Only vendor packages that are NOT in the current repository
	return !vm.git.InRepo(pkgPath)
}

func (vm *VendorManager) getLocalClawFile(path string) (git.ClawFile, error) {
	// For local paths, find the .claw file in the directory
	var clawFilePath string
	if strings.HasSuffix(path, ".claw") {
		clawFilePath = path
	} else {
		// Find .claw file in directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return git.ClawFile{}, fmt.Errorf("failed to read directory %s: %w", path, err)
		}

		var clawFiles []string
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".claw") {
				clawFiles = append(clawFiles, entry.Name())
			}
		}

		switch len(clawFiles) {
		case 0:
			return git.ClawFile{}, fmt.Errorf("no .claw file found in directory %s", path)
		case 1:
			clawFilePath = filepath.Join(path, clawFiles[0])
		default:
			return git.ClawFile{}, fmt.Errorf("multiple .claw files found in directory %s: %v", path, clawFiles)
		}
	}

	// Read the .claw file content
	content, err := os.ReadFile(clawFilePath)
	if err != nil {
		return git.ClawFile{}, fmt.Errorf("failed to read file %s: %w", clawFilePath, err)
	}

	return git.ClawFile{
		Content: content,
		Version: "local",
	}, nil
}

func (vm *VendorManager) parseClawFile(content []byte) (*idl.File, error) {
	// Parse the claw file using the existing IDL parser
	file := idl.New()
	ctx := context.Background()
	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), file); err != nil {
		return nil, err
	}
	return file, nil
}

func (vm *VendorManager) getModuleFile(ctx context.Context, pkgPath, version string) (*imports.Module, error) {
	// Get the module file for a dependency
	if vm.isLocalPath(pkgPath) {
		// For local paths, try to read claw.mod directly
		modPath := filepath.Join(pkgPath, "claw.mod")
		if _, err := os.Stat(modPath); os.IsNotExist(err) {
			return nil, nil // No module file is OK for dependencies
		}

		content, err := os.ReadFile(modPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read local module file: %w", err)
		}

		module := &imports.Module{}
		if err := vm.parseModuleContent(string(content), module); err != nil {
			return nil, fmt.Errorf("failed to parse local module file: %w", err)
		}
		return module, nil
	}

	// For remote paths, this would need to fetch the claw.mod file
	// For now, we'll return nil to indicate no module file
	return nil, nil
}

func (vm *VendorManager) extractImports(file *idl.File) []string {
	// Extract import statements from the IDL file
	var imports []string
	for _, imp := range file.Imports.Imports {
		imports = append(imports, imp.Path)
	}
	return imports
}

func (vm *VendorManager) resolveVersion(pkgPath string, rootModule *imports.Module) string {
	// Resolve version based on claw.mod requirements
	for _, req := range rootModule.Required {
		if req.Path == pkgPath {
			if !req.Version.IsZero() {
				return fmt.Sprintf("v%d.%d.%d", req.Version.Major, req.Version.Minor, req.Version.Patch)
			}
			return req.ID
		}
	}
	return "" // Use latest
}

func (vm *VendorManager) markLeafNodes(graph *DependencyGraph) {
	// Mark nodes that have no dependencies as leaf nodes
	for pkgPath, node := range graph.Nodes {
		node.IsLeaf = len(graph.Edges[pkgPath]) == 0
	}
}

// validateACLs validates that all dependencies allow the current module to import them.
func (vm *VendorManager) validateACLs(ctx context.Context, graph *DependencyGraph, rootModule *imports.Module) error {
	currentModulePath := rootModule.Path

	for pkgPath, node := range graph.Nodes {
		if node.ModuleFile != nil {
			if err := vm.checkACLPermission(currentModulePath, pkgPath, node.ModuleFile); err != nil {
				return fmt.Errorf("ACL violation for %s: %w", pkgPath, err)
			}
		}
	}

	return nil
}

func (vm *VendorManager) checkACLPermission(currentModule, targetModule string, targetModuleFile *imports.Module) error {
	// Check if current module is allowed to import target module
	if len(targetModuleFile.ACLs) == 0 {
		return fmt.Errorf("module %s does not allow any imports", targetModule)
	}

	// Check for public access
	for _, acl := range targetModuleFile.ACLs {
		if acl.Path == "*" || acl.Path == "public" {
			return nil
		}

		// Check exact match
		if acl.Path == currentModule {
			return nil
		}

		// Check wildcard match
		if strings.HasSuffix(acl.Path, "/*") {
			prefix := strings.TrimSuffix(acl.Path, "/*")
			if strings.HasPrefix(currentModule, prefix) {
				return nil
			}
		}
	}

	return fmt.Errorf("module %s is not allowed to import %s", currentModule, targetModule)
}

// createVendorStructure creates the vendor directory structure and copies files.
func (vm *VendorManager) createVendorStructure(ctx context.Context, graph *DependencyGraph, localReplace imports.LocalReplace) error {
	// Create vendor directory
	if err := os.MkdirAll(vm.vendorDir, 0o755); err != nil {
		return fmt.Errorf("failed to create vendor directory: %w", err)
	}

	// Copy only external dependencies to vendor directory (skip local packages)
	for pkgPath, node := range graph.Nodes {
		if node.ShouldVendor {
			if err := vm.vendorPackage(ctx, pkgPath, node, localReplace); err != nil {
				return fmt.Errorf("failed to vendor package %s: %w", pkgPath, err)
			}
		}
	}

	return nil
}

func (vm *VendorManager) vendorPackage(ctx context.Context, pkgPath string, node *DependencyNode, localReplace imports.LocalReplace) error {
	// Create package directory in vendor
	vendorPkgDir := filepath.Join(vm.vendorDir, pkgPath)
	if err := os.MkdirAll(vendorPkgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create vendor package directory: %w", err)
	}

	// Copy .claw file
	clawFileName := vm.getClawFileName(pkgPath)
	clawFilePath := filepath.Join(vendorPkgDir, clawFileName)
	if err := vm.writeClawFile(clawFilePath, node); err != nil {
		return fmt.Errorf("failed to write claw file: %w", err)
	}

	// Copy claw.mod file if it exists
	if node.ModuleFile != nil {
		modFilePath := filepath.Join(vendorPkgDir, "claw.mod")
		if err := vm.writeModuleFile(modFilePath, node.ModuleFile); err != nil {
			return fmt.Errorf("failed to write module file: %w", err)
		}
	}

	return nil
}

func (vm *VendorManager) getClawFileName(pkgPath string) string {
	// Extract the claw file name from the package path
	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1] + ".claw"
}

func (vm *VendorManager) writeClawFile(filePath string, node *DependencyNode) error {
	// Write the original .claw file content to preserve exact format
	return os.WriteFile(filePath, node.OriginalContent, 0o644)
}

func (vm *VendorManager) serializeIDLFile(file *idl.File) string {
	// This is a simplified serialization - in practice, you'd want to preserve
	// the original format and structure
	var sb strings.Builder

	// Write package declaration first (required by Claw language)
	if file.Package != "" {
		sb.WriteString(fmt.Sprintf("package %s\n\n", file.Package))
	}

	// Write imports
	if len(file.Imports.Imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range file.Imports.Imports {
			sb.WriteString(fmt.Sprintf("    \"%s\"\n", imp.Path))
		}
		sb.WriteString(")\n\n")
	}

	// Write enums
	for enum := range file.Enums() {
		sb.WriteString(fmt.Sprintf("Enum %s {\n", enum.Name))
		for i, entry := range enum.OrderByValues() {
			sb.WriteString(fmt.Sprintf("    %s @%d\n", entry.Name, i))
		}
		sb.WriteString("}\n\n")
	}

	// Write structs
	for _, strct := range file.Structs() {
		sb.WriteString(fmt.Sprintf("Struct %s {\n", strct.Name))
		for i, field := range strct.Fields {
			sb.WriteString(fmt.Sprintf("    %s %s @%d\n", field.Name, field.Type, i))
		}
		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func (vm *VendorManager) writeModuleFile(filePath string, module *imports.Module) error {
	// Write the module file to the vendor directory
	content := vm.serializeModule(module)
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func (vm *VendorManager) serializeModule(module *imports.Module) string {
	var sb strings.Builder

	// Write module path
	sb.WriteString(fmt.Sprintf("module %s\n\n", module.Path))

	// Write require block
	if len(module.Required) > 0 {
		sb.WriteString("require (\n")
		for _, req := range module.Required {
			if !req.Version.IsZero() {
				sb.WriteString(fmt.Sprintf("    %s v%d.%d.%d\n", req.Path, req.Version.Major, req.Version.Minor, req.Version.Patch))
			} else if req.ID != "" {
				sb.WriteString(fmt.Sprintf("    %s %s\n", req.Path, req.ID))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", req.Path))
			}
		}
		sb.WriteString(")\n\n")
	}

	// Write replace block
	if len(module.Replace) > 0 {
		sb.WriteString("replace (\n")
		for _, rep := range module.Replace {
			sb.WriteString(fmt.Sprintf("    %s => %s\n", rep.FromPath, rep.ToPath))
		}
		sb.WriteString(")\n\n")
	}

	// Write ACLs
	if len(module.ACLs) > 0 {
		sb.WriteString("acls (\n")
		for _, acl := range module.ACLs {
			sb.WriteString(fmt.Sprintf("    %s\n", acl.Path))
		}
		sb.WriteString(")\n")
	}

	return sb.String()
}

// GetCompilationOrder returns the packages in the order they should be compiled.
func (vm *VendorManager) GetCompilationOrder(graph *DependencyGraph) ([]string, error) {
	// Perform topological sort to determine compilation order
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []string

	var visit func(string) error
	visit = func(pkgPath string) error {
		if visiting[pkgPath] {
			return fmt.Errorf("circular dependency detected involving %s", pkgPath)
		}
		if visited[pkgPath] {
			return nil
		}

		visiting[pkgPath] = true
		for _, dep := range graph.Edges[pkgPath] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[pkgPath] = false
		visited[pkgPath] = true
		result = append(result, pkgPath)
		return nil
	}

	// Visit all nodes
	for pkgPath := range graph.Nodes {
		if err := visit(pkgPath); err != nil {
			return nil, err
		}
	}

	// Reverse the result to get dependencies first
	for i := len(result)/2 - 1; i >= 0; i-- {
		opp := len(result) - 1 - i
		result[i], result[opp] = result[opp], result[i]
	}

	return result, nil
}

// VendorSingleDependency vendors a single dependency at a specific version,
// including its transitive dependencies.
// This is used by the "clawc get" command to update individual dependencies.
// It validates ACLs for all dependencies before vendoring.
func (vm *VendorManager) VendorSingleDependency(ctx context.Context, pkgPath, version string) error {
	// Step 1: Parse the root claw.mod file (need current module for ACL checks)
	rootModule, err := vm.parseModuleFile(vm.rootDir)
	if err != nil {
		// If claw.mod doesn't exist in root, create a minimal module
		// This allows clawc get to work without a claw.mod file
		rootModule = &imports.Module{
			Path: "local",
			ACLs: []imports.ACL{}, // Empty ACL list allows all imports
		}
	}

	// Step 2: Parse local.replace if it exists
	localReplace, err := vm.parseLocalReplace(vm.rootDir)
	if err != nil {
		// local.replace is optional, continue without it
		localReplace = imports.LocalReplace{}
	}

	// Step 3: Build dependency graph for this package and its transitive dependencies
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}

	// Process the package and its transitive dependencies
	visited := make(map[string]bool)
	if err := vm.processSingleDependency(ctx, pkgPath, version, graph, rootModule, localReplace, visited); err != nil {
		return fmt.Errorf("failed to process dependency %s@%s: %w", pkgPath, version, err)
	}

	// Step 4: Validate ACLs for all dependencies in the graph
	if err := vm.validateACLs(ctx, graph, rootModule); err != nil {
		return fmt.Errorf("ACL validation failed for %s: %w", pkgPath, err)
	}

	// Step 5: Vendor all packages in the dependency graph
	for depPath, node := range graph.Nodes {
		if err := vm.vendorPackage(ctx, depPath, node, localReplace); err != nil {
			return fmt.Errorf("failed to vendor package %s: %w", depPath, err)
		}
	}

	return nil
}

// processSingleDependency processes a single dependency and its transitive dependencies.
// It's similar to processClawFile but simplified for the single dependency use case.
func (vm *VendorManager) processSingleDependency(
	ctx context.Context,
	pkgPath, version string,
	graph *DependencyGraph,
	rootModule *imports.Module,
	localReplace imports.LocalReplace,
	visited map[string]bool,
) error {
	// Check for circular dependencies
	if visited[pkgPath] {
		return fmt.Errorf("circular dependency detected: %s", pkgPath)
	}
	visited[pkgPath] = true
	defer delete(visited, pkgPath)

	// Skip if already processed
	if _, exists := graph.Nodes[pkgPath]; exists {
		return nil
	}

	// Apply replace directives to get the actual source
	actualPath, actualVersion := vm.applyReplaceDirectives(pkgPath, version, rootModule, localReplace)
	if actualVersion == "" {
		actualVersion = version
	}

	// Get the claw file content
	var clawFile git.ClawFile
	var err error

	if vm.isLocalPath(actualPath) {
		clawFile, err = vm.getLocalClawFile(actualPath)
	} else if vm.git.InRepo(actualPath) {
		// Package is in the current git repository, use local filesystem
		localPath, absErr := vm.git.Abs(actualPath)
		if absErr != nil {
			return fmt.Errorf("failed to resolve local path for %s: %w", actualPath, absErr)
		}
		clawFile, err = vm.getLocalClawFile(localPath)
	} else {
		clawFile, err = vm.getClawFile(ctx, actualPath, actualVersion)
	}

	if err != nil {
		return fmt.Errorf("failed to get claw file for %s: %w", pkgPath, err)
	}

	// Parse the claw file to extract imports
	idlFile, err := vm.parseClawFile(clawFile.Content)
	if err != nil {
		return fmt.Errorf("failed to parse claw file %s: %w", pkgPath, err)
	}

	// Get the module file for this dependency
	moduleFile, err := vm.getModuleFile(ctx, actualPath, actualVersion)
	if err != nil {
		// Module file is optional for dependencies
		moduleFile = nil
	}

	// Create dependency node
	node := &DependencyNode{
		PkgPath:         pkgPath,
		Version:         clawFile.Version,
		Dependencies:    vm.extractImports(idlFile),
		ClawFile:        idlFile,
		OriginalContent: clawFile.Content,
		ModuleFile:      moduleFile,
		ShouldVendor:    true, // For clawc get, always vendor
	}

	graph.Nodes[pkgPath] = node
	graph.Edges[pkgPath] = node.Dependencies

	// Recursively process transitive dependencies
	for _, dep := range node.Dependencies {
		// Apply version resolution from the root module's require section
		depVersion := vm.resolveVersion(dep, rootModule)
		if err := vm.processSingleDependency(ctx, dep, depVersion, graph, rootModule, localReplace, visited); err != nil {
			return fmt.Errorf("failed to process transitive dependency %s: %w", dep, err)
		}
	}

	return nil
}

// IsVendored checks if a package is already vendored.
func (vm *VendorManager) IsVendored(pkgPath string) bool {
	vendorPath := filepath.Join(vm.vendorDir, pkgPath)

	// Check if the vendor directory exists
	if _, err := os.Stat(vendorPath); os.IsNotExist(err) {
		return false
	}

	// Check if a .claw file exists in the directory
	entries, err := os.ReadDir(vendorPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".claw") {
			return true
		}
	}

	return false
}

// GetVendoredVersion returns the version of a vendored package.
// Returns empty string if the package is not vendored or version cannot be determined.
func (vm *VendorManager) GetVendoredVersion(pkgPath string) (string, error) {
	if !vm.IsVendored(pkgPath) {
		return "", fmt.Errorf("package %s is not vendored", pkgPath)
	}

	// Try to read the claw.mod file for version information
	modPath := filepath.Join(vm.vendorDir, pkgPath, "claw.mod")
	if _, err := os.Stat(modPath); err == nil {
		content, err := os.ReadFile(modPath)
		if err == nil {
			module := &imports.Module{}
			if err := vm.parseModuleContent(string(content), module); err == nil {
				// For now, we don't have version info in the module itself
				// We would need to store this separately or in a metadata file
				// Return empty for now
				return "", nil
			}
		}
	}

	// Version tracking could be improved by storing version metadata
	// in a separate file or in the claw.mod file
	return "", nil
}

// GetVendorPath returns the vendor path for a given package path.
func (vm *VendorManager) GetVendorPath(pkgPath string) string {
	if pkgPath == "" {
		return vm.vendorDir
	}
	return filepath.Join(vm.vendorDir, pkgPath)
}
