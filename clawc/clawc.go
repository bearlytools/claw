package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	osfs "github.com/gopherfs/fs/io/os"
	// Registers the golang renderer.

	"github.com/bearlytools/claw/clawc/internal/imports"
	"github.com/bearlytools/claw/clawc/internal/imports/git"
	"github.com/bearlytools/claw/clawc/internal/render"
	_ "github.com/bearlytools/claw/clawc/internal/render/golang"
	"github.com/bearlytools/claw/clawc/internal/vendor"
	"github.com/bearlytools/claw/clawc/internal/writer"
)

func main() {
	ctx := context.Background()

	flag.Parse()
	args := flag.Args()

	// Command routing: check if first argument is a command
	if len(args) > 0 && args[0] == "get" {
		handleGet(ctx, args[1:])
		return
	}

	// Default: compilation mode
	handleCompile(ctx, args)
}

// handleGet implements the "clawc get package@version" command
func handleGet(ctx context.Context, args []string) {
	if len(args) == 0 {
		exitf("usage: clawc get <package>[@version]")
	}

	// Parse package@version syntax
	pkgSpec := args[0]
	pkg, version, err := parsePackageSpec(pkgSpec)
	if err != nil {
		exitf("invalid package specification: %s", err)
	}

	// Resolve @latest if needed
	if version == "latest" || version == "" {
		latestVersion, err := git.GetLatestVersion(ctx, pkg)
		if err != nil {
			exitf("failed to resolve latest version for %s: %s", pkg, err)
		}
		version = latestVersion
		fmt.Printf("Resolved %s to version %s\n", pkg, version)
	}

	// Create vendor manager
	vendorManager, err := vendor.NewVendorManager(".")
	if err != nil {
		exitf("failed to create vendor manager: %s", err)
	}

	// Vendor the single dependency
	fmt.Printf("Fetching %s@%s...\n", pkg, version)
	if err := vendorManager.VendorSingleDependency(ctx, pkg, version); err != nil {
		exitf("failed to vendor dependency %s@%s: %s", pkg, version, err)
	}

	// Find the vendored .claw file
	vendorPkgDir := vendorManager.GetVendorPath(pkg)
	vendorClawFile, err := findClawFileInDir(vendorPkgDir)
	if err != nil {
		exitf("failed to find .claw file in vendor directory %s: %s", vendorPkgDir, err)
	}

	// Compile the dependency to generate .go file (force regenerate)
	fmt.Printf("Generating .go file for %s...\n", pkg)
	rootVendorDir := vendorManager.GetVendorPath("")
	if err := compileDependencyWithOptions(ctx, vendorClawFile, rootVendorDir, vendorManager.GetWorkRepo(), vendorManager.GetWorkVendorDir(), true); err != nil {
		exitf("failed to compile dependency %s: %s", pkg, err)
	}

	// TODO: Update claw.mod with the new version requirement
	fmt.Printf("Successfully fetched and compiled %s@%s\n", pkg, version)
	fmt.Printf("Note: claw.mod not automatically updated yet - manual update required\n")
}

// parsePackageSpec parses a package specification in the format "package[@version]"
// and returns the package path and version separately.
// Returns an error if the package specification is invalid.
func parsePackageSpec(spec string) (pkg, version string, err error) {
	// Trim whitespace
	spec = strings.TrimSpace(spec)

	// Check for empty string
	if spec == "" {
		return "", "", fmt.Errorf("package specification cannot be empty")
	}

	// Split on @ to separate package and version
	parts := strings.SplitN(spec, "@", 2)
	pkg = strings.TrimSpace(parts[0])

	// Check if package path is empty (e.g., "@v1.2.3")
	if pkg == "" {
		return "", "", fmt.Errorf("package path is required (spec starts with @)")
	}

	// Extract version if present
	if len(parts) == 2 {
		version = strings.TrimSpace(parts[1])
		// Empty version after @ is treated as no version
		if version == "" {
			version = ""
		}
	}

	return pkg, version, nil
}

// handleCompile implements the default compilation behavior
func handleCompile(ctx context.Context, args []string) {
	path := ""
	if len(args) == 0 {
		path = "."
	} else {
		path = args[0]
	}

	// Mount our filesystem for reading.
	fs, err := osfs.New()
	if err != nil {
		panic(err)
	}

	var clawFile string
	// Check if the path is a file or directory
	if strings.HasSuffix(path, ".claw") {
		// It's a file path, use it directly and extract directory for vendoring
		clawFile = path
		path = filepath.Dir(path)
	} else {
		// It's a directory path, find the .claw file in it
		clawFile, err = imports.FindClawFile(fs, path)
		if err != nil {
			exitf("problem finding .claw file: %s", err)
		}
	}

	// Convert clawFile to absolute path to avoid path resolution issues
	clawFile, err = filepath.Abs(clawFile)
	if err != nil {
		exitf("failed to get absolute path for claw file: %s", err)
	}

	// Step 1: Vendor all dependencies
	vendorManager, err := vendor.NewVendorManager(path)
	if err != nil {
		exitf("failed to create vendor manager: %s", err)
	}

	dependencyGraph, err := vendorManager.VendorDependencies(ctx, clawFile)
	if err != nil {
		exitf("failed to vendor dependencies: %s", err)
	}

	// Step 2: Get compilation order
	compilationOrder, err := vendorManager.GetCompilationOrder(dependencyGraph)
	if err != nil {
		exitf("failed to determine compilation order: %s", err)
	}

	// Step 3: Compile dependencies in order
	originalClawFile := clawFile // Preserve the original file path
	for _, pkgPath := range compilationOrder {
		// Skip the root package for now - we'll compile it separately at the end
		if strings.Contains(pkgPath, originalClawFile) {
			continue
		}

		// Check if this is a local dependency (within current git repo) or external
		if isLocalDependency(vendorManager, pkgPath) {
			// Local dependency - compile from the actual repository location
			if compileErr := compileLocalDependency(ctx, vendorManager, pkgPath); compileErr != nil {
				exitf("failed to compile local dependency %s: %s", pkgPath, compileErr)
			}
		} else {
			// External dependency - compile from vendor directory
			rootVendorDir := vendorManager.GetVendorPath("")
			vendorPkgDir := vendorManager.GetVendorPath(pkgPath)
			vendorClawFile, err := findClawFileInDir(vendorPkgDir)
			if err != nil {
				exitf("failed to find .claw file in vendor directory %s: %s", vendorPkgDir, err)
			}

			if compileErr := compileDependency(ctx, vendorClawFile, rootVendorDir, vendorManager.GetWorkRepo(), vendorManager.GetWorkVendorDir()); compileErr != nil {
				exitf("failed to compile dependency %s: %s", pkgPath, compileErr)
			}
		}
	}

	// Step 4: Compile the main file using vendored dependencies
	vendoredConfig := imports.NewVendoredConfig(
		vendorManager.GetVendorPath(""),
		vendorManager.GetWorkRepo(),
		vendorManager.GetWorkVendorDir(),
	)
	if readErr := vendoredConfig.Read(ctx, originalClawFile); readErr != nil {
		exitf("error: %s\n", readErr)
	}

	rendered, err := render.Render(ctx, vendoredConfig.Config, render.Go)
	if err != nil {
		exit(err)
	}
	wr, err := writer.New(vendoredConfig, writer.WithVendorDir(vendorManager.GetVendorPath("")))
	if err != nil {
		exit(err)
	}
	if err := wr.Write(ctx, rendered); err != nil {
		exit(err)
	}
}

// compileDependency compiles a single dependency from the vendor directory.
func compileDependency(ctx context.Context, vendorClawFile, rootVendorDir, repoPath, vendorDirName string) error {
	return compileDependencyWithOptions(ctx, vendorClawFile, rootVendorDir, repoPath, vendorDirName, false)
}

// compileDependencyWithOptions compiles a single dependency with optional force regeneration.
func compileDependencyWithOptions(ctx context.Context, vendorClawFile, rootVendorDir, repoPath, vendorDirName string, forceRegenerate bool) error {

	// Create a vendored config for the dependency that points to the root vendor directory
	vendoredConfig := imports.NewVendoredConfig(rootVendorDir, repoPath, vendorDirName)
	if err := vendoredConfig.Read(ctx, vendorClawFile); err != nil {
		return fmt.Errorf("error reading dependency config: %w", err)
	}

	rendered, err := render.Render(ctx, vendoredConfig.Config, render.Go)
	if err != nil {
		return fmt.Errorf("error rendering dependency: %w", err)
	}

	var wr *writer.Writer
	if forceRegenerate {
		wr, err = writer.New(vendoredConfig, writer.WithVendorDir(rootVendorDir), writer.WithForceRegenerate(true))
	} else {
		wr, err = writer.New(vendoredConfig, writer.WithVendorDir(rootVendorDir))
	}
	if err != nil {
		return fmt.Errorf("error creating writer: %w", err)
	}

	if err := wr.Write(ctx, rendered); err != nil {
		return fmt.Errorf("error writing dependency: %w", err)
	}

	return nil
}

// compileLocalDependency compiles a local dependency that's within the claw.work boundary.
func compileLocalDependency(ctx context.Context, vendorManager *vendor.VendorManager, pkgPath string) error {
	// For local dependencies, we need to find the actual file system path
	// and compile it directly without using vendor directory

	// Get claw.work configuration
	work := vendorManager.GetWork()
	workDir := vendorManager.GetWorkDir()

	// Convert package path to file system path relative to claw.work
	// For github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers
	// We want testing/imports/vehicles/claw/manufacturers
	if !strings.HasPrefix(pkgPath, work.Repo) {
		return fmt.Errorf("package path %s does not start with claw.work repo %s", pkgPath, work.Repo)
	}

	relPath := strings.TrimPrefix(pkgPath, work.Repo+"/")
	localPath := filepath.Join(workDir, relPath)

	// Find the .claw file in the local directory
	clawFile, err := findClawFileInDir(localPath)
	if err != nil {
		return fmt.Errorf("failed to find .claw file in local directory %s: %w", localPath, err)
	}

	// Create a regular config (not vendored) for local dependencies
	config := imports.NewConfig()
	if err := config.Read(ctx, clawFile); err != nil {
		return fmt.Errorf("error reading local dependency config: %w", err)
	}

	rendered, err := render.Render(ctx, config, render.Go)
	if err != nil {
		return fmt.Errorf("error rendering local dependency: %w", err)
	}

	wr, err := writer.New(config)
	if err != nil {
		return fmt.Errorf("error creating writer for local dependency: %w", err)
	}

	if err := wr.Write(ctx, rendered); err != nil {
		return fmt.Errorf("error writing local dependency: %w", err)
	}

	return nil
}

// isLocalDependency checks if a package is within the claw.work repo boundary
func isLocalDependency(vendorManager *vendor.VendorManager, pkgPath string) bool {
	work := vendorManager.GetWork()
	if work == nil {
		return false
	}
	return strings.HasPrefix(pkgPath, work.Repo)
}

// findClawFileInDir finds the .claw file in a directory.
func findClawFileInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var clawFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".claw") {
			clawFiles = append(clawFiles, entry.Name())
		}
	}

	switch len(clawFiles) {
	case 0:
		return "", fmt.Errorf("no .claw file found in directory %s", dir)
	case 1:
		return filepath.Join(dir, clawFiles[0]), nil
	default:
		return "", fmt.Errorf("multiple .claw files found in directory %s: %v", dir, clawFiles)
	}
}

func exit(i ...any) {
	fmt.Println(i...)
	os.Exit(1)
}

func exitf(s string, i ...any) {
	fmt.Printf(s+"\n", i...)
	os.Exit(1)
}
