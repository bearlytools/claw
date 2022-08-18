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

	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/render"
	"github.com/gopherfs/fs"
)

// Writer implements writer.WriteFiles for the Go language.
type Writer struct {
	fs fs.Writer
}

func (w *Writer) SetFS(fs fs.Writer) {
	w.fs = fs
}

func (w *Writer) WriteFiles(ctx context.Context, config *imports.Config, renders []render.Rendered) error {
	if err := w.getImports(ctx, config); err != nil {
		return err
	}

	for _, r := range renders {
		log.Println("rendering: ", r.Path)
		if config.InRootRepo(r.Path) {
			p, err := config.Abs(r.Path)
			if err != nil {
				return err
			}
			p = filepath.Join(p, r.Package+".go")
			if err := w.fs.WriteFile(p, r.Native, 0600); err != nil {
				return fmt.Errorf("problem writing package(%s) to local file(%s): %w", r.Package, p, err)
			}
		}
	}
	log.Println("done")
	return nil
}

func (w *Writer) getImports(ctx context.Context, config *imports.Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)
	for _, imp := range config.Imports {
		imp := imp
		wg.Add(1)

		go func() {
			defer wg.Done()

			if config.InRootRepo(imp.FullPath) {
				return
			}

			if err := w.goGet(ctx, imp.FullPath, imp.RepoVersion); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}()
	}
	wg.Wait()
	close(errCh)

	return <-errCh
}

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
