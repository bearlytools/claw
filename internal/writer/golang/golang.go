package golang

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/gopherfs/fs"

	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/render"
)

// Writer implements writer.WriteFiles for the Go language.
type Writer struct {
	fs               fs.Writer
	forceRegenerate  bool   // Force regeneration of all .go files
	vendorDir        string // Path to vendor directory for lazy loading check
}

func (w *Writer) SetFS(fs fs.Writer) {
	w.fs = fs
}

// SetForceRegenerate sets whether to force regeneration of all .go files.
// This is used by "clawc get" to ensure .go files are regenerated.
func (w *Writer) SetForceRegenerate(force bool) {
	w.forceRegenerate = force
}

// SetVendorDir sets the vendor directory path for lazy loading checks.
func (w *Writer) SetVendorDir(dir string) {
	w.vendorDir = dir
}

func (w *Writer) WriteFiles(ctx context.Context, config imports.ConfigProvider, renders []render.Rendered) error {
	if err := w.getImports(ctx, config); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errs := newErrs()
	for _, r := range renders {
		r := r
		log.Println("rendering: ", r.Path)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if config.InRootRepo(r.Path) {
				p, err := config.Abs(r.Path)
				if err != nil {
					errs.add(err)
					return
				}
				p = filepath.Join(p, r.Package+".go")

				// Lazy loading: check if we should skip this file
				if w.shouldSkipGeneration(p, r.Path) {
					log.Printf("Skipping generation for %s (already exists)\n", p)
					return
				}

				if err := w.fs.WriteFile(p, r.Native, 0o600); err != nil {
					errs.add(fmt.Errorf("problem writing package(%s) to local file(%s): %w", r.Package, p, err))
					return
				}
			}
		}()
	}
	wg.Wait()

	log.Println("done")
	return errs.get()
}

// shouldSkipGeneration determines if .go file generation should be skipped.
// Returns true if:
// - The file is in the vendor directory (not local)
// - The .go file already exists
// - We're not in force regenerate mode
func (w *Writer) shouldSkipGeneration(goFilePath, clawPath string) bool {
	// Force regenerate mode always generates (never skips)
	if w.forceRegenerate {
		return false
	}

	// If no vendor directory is set, never skip (not in lazy loading mode)
	if w.vendorDir == "" {
		return false
	}

	// Check if this is a vendored file
	isVendored := isInVendor(clawPath, w.vendorDir)
	if !isVendored {
		// Local files are always regenerated
		return false
	}

	// For vendored files, check if .go file exists
	if _, err := os.Stat(goFilePath); os.IsNotExist(err) {
		// .go file doesn't exist, need to generate
		return false
	}

	// .go file exists for vendored file, skip generation
	return true
}

// isInVendor checks if a path is within the vendor directory.
func isInVendor(path, vendorDir string) bool {
	if vendorDir == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absVendorDir, err := filepath.Abs(vendorDir)
	if err != nil {
		return false
	}
	// Check if path starts with vendor directory
	rel, err := filepath.Rel(absVendorDir, absPath)
	if err != nil {
		return false
	}
	// If the relative path doesn't start with "..", it's inside vendor
	return !filepath.IsAbs(rel) && !filepath.HasPrefix(rel, "..")
}

// getImports is going to grab any file that the Claw file imports and is not in the
// current Git repo via the "go get" command.
func (w *Writer) getImports(ctx context.Context, config imports.ConfigProvider) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := sync.WaitGroup{}
	errs := newErrs()
	for _, imp := range config.GetImports() {
		imp := imp
		wg.Add(1)

		go func() {
			defer wg.Done()

			if config.InRootRepo(imp.FullPath) {
				return
			}

			if err := w.goGet(ctx, imp.FullPath, imp.RepoVersion); err != nil {
				errs.add(err)
				cancel()
			}
		}()
	}
	wg.Wait()

	return errs.get()
}

// goGet uses the "go get" command to fetch a package at some version.
func (w *Writer) goGet(ctx context.Context, pkg, version string) error {
	p, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go command is not installed: %s", err)
	}
	var get string
	if version == "" {
		get = fmt.Sprintf("%s@latest", pkg)
	} else {
		get = fmt.Sprintf("%s@%s", pkg, version)
	}

	log.Println("get: ", get)
	cmd := exec.CommandContext(ctx, p, "get", "-d", get)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("error output of 'go get':\n", string(out))
		return fmt.Errorf("problem running %q:\n%s", cmd.String(), string(out))
	}
	return nil
}

// sameFile determines if the file at path has the size and sha256 hash as what is passed.
func sameFile(path string, size int64, sum256 string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if size != fi.Size() {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	sum := fmt.Sprintf("%x", h.Sum(nil))
	if sum != sum256 {
		return false, nil
	}
	return true, nil
}

type goErrors struct {
	ch chan error
}

func newErrs() goErrors {
	return goErrors{ch: make(chan error, 1)}
}

func (g goErrors) add(err error) {
	select {
	case g.ch <- err:
	default:
	}
}

func (g goErrors) get() error {
	select {
	case e := <-g.ch:
		return e
	default:
		return nil
	}
}
