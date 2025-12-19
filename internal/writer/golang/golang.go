package golang

import (
	"github.com/gostdlib/base/context"
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
	fs              fs.Writer
	forceRegenerate bool   // Force regeneration of all .go files
	vendorDir       string // Path to vendor directory for lazy loading check
}

func (w *Writer) SetFS(fs fs.Writer) {
	w.fs = fs
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
	cmd := exec.CommandContext(ctx, p, "get", get)
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
