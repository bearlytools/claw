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

	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/render"
	_ "github.com/bearlytools/claw/internal/render/golang"
	"github.com/bearlytools/claw/internal/vendor"
	"github.com/bearlytools/claw/internal/writer"
)

func main() {
	ctx := context.Background()

	flag.Parse()
	args := flag.Args()
	path := ""
	switch len(args) {
	case 0:
		path = "."
	case 1:
		path = args[0]
	default:
		panic("you can 0 or 1 argument")
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
			vendorPkgDir := vendorManager.GetVendorPath(pkgPath)
			vendorClawFile, err := findClawFileInDir(vendorPkgDir)
			if err != nil {
				exitf("failed to find .claw file in vendor directory %s: %s", vendorPkgDir, err)
			}

			if compileErr := compileDependency(ctx, vendorClawFile); compileErr != nil {
				exitf("failed to compile dependency %s: %s", pkgPath, compileErr)
			}
		}
	}

	// Step 4: Compile the main file using vendored dependencies
	vendoredConfig := imports.NewVendoredConfig(vendorManager.GetVendorPath(""))
	if readErr := vendoredConfig.Read(ctx, originalClawFile); readErr != nil {
		exitf("error: %s\n", readErr)
	}

	rendered, err := render.Render(ctx, vendoredConfig.Config, render.Go)
	if err != nil {
		exit(err)
	}
	wr, err := writer.New(vendoredConfig)
	if err != nil {
		exit(err)
	}
	if err := wr.Write(ctx, rendered); err != nil {
		exit(err)
	}
}

// compileDependency compiles a single dependency from the vendor directory.
func compileDependency(ctx context.Context, vendorClawFile string) error {
	// Find the root vendor directory (not the dependency-specific one)
	rootVendorDir := findRootVendorDir(vendorClawFile)

	// Create a vendored config for the dependency that points to the root vendor directory
	vendoredConfig := imports.NewVendoredConfig(rootVendorDir)
	if err := vendoredConfig.Read(ctx, vendorClawFile); err != nil {
		return fmt.Errorf("error reading dependency config: %w", err)
	}

	rendered, err := render.Render(ctx, vendoredConfig.Config, render.Go)
	if err != nil {
		return fmt.Errorf("error rendering dependency: %w", err)
	}

	wr, err := writer.New(vendoredConfig)
	if err != nil {
		return fmt.Errorf("error creating writer: %w", err)
	}

	if err := wr.Write(ctx, rendered); err != nil {
		return fmt.Errorf("error writing dependency: %w", err)
	}

	return nil
}

// compileLocalDependency compiles a local dependency that's within the current git repository.
func compileLocalDependency(ctx context.Context, vendorManager *vendor.VendorManager, pkgPath string) error {
	// For local dependencies, we need to find the actual file system path
	// and compile it directly without using vendor directory

	// Get the git repo root
	repoRoot := vendorManager.GetRepoRoot()

	// Convert package path to file system path relative to repo root
	// For github.com/bearlytools/claw/testing/imports/vehicles/claw/manufacturers
	// We want testing/imports/vehicles/claw/manufacturers
	gitOrigin := vendorManager.GetGitOrigin()
	if !strings.HasPrefix(pkgPath, gitOrigin) {
		return fmt.Errorf("package path %s does not start with git origin %s", pkgPath, gitOrigin)
	}

	relPath := strings.TrimPrefix(pkgPath, gitOrigin+"/")
	localPath := filepath.Join(repoRoot, relPath)

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

// isLocalDependency checks if a package is within the current git repository
func isLocalDependency(vendorManager *vendor.VendorManager, pkgPath string) bool {
	origin := vendorManager.GetGitOrigin()
	if origin == "" {
		// No git info available, assume external
		return false
	}
	return strings.HasPrefix(pkgPath, origin)
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

// findRootVendorDir finds the root vendor directory from a vendored file path.
func findRootVendorDir(vendorClawFile string) string {
	// Walk up the directory tree to find the "vendor" directory
	dir := filepath.Dir(vendorClawFile)
	for {
		if filepath.Base(dir) == "vendor" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, fallback to current directory
			break
		}
		dir = parent
	}

	// Fallback: assume vendor is at the current directory
	return "vendor"
}

func exit(i ...any) {
	fmt.Println(i...)
	os.Exit(1)
}

func exitf(s string, i ...any) {
	fmt.Printf(s+"\n", i...)
	os.Exit(1)
}
