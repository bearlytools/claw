// Package writer contains interfaces that can be implemented to render files for
// a language implementation and a type that can be used to call those implementations
// and write out those files for all languages that were asked to be rendered.
package writer

import (
	"context"
	"fmt"
	"sync"

	"github.com/gopherfs/fs"
	osfs "github.com/gopherfs/fs/io/os"

	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/render"
	"github.com/bearlytools/claw/internal/writer/golang"
)

var supported = map[render.Lang]WriteFiles{
	render.Go: &golang.Writer{},
}

// WriteFiles writes a file to some location based on the language.
type WriteFiles interface {
	SetFS(fs.Writer)
	WriteFiles(context.Context, imports.ConfigProvider, []render.Rendered) error
}

// LazyWriter is an optional interface for writers that support lazy loading.
type LazyWriter interface {
	SetVendorDir(string)
	SetForceRegenerate(bool)
}

// Runtime init check that both render and writer both support the same languages and
// all vcs types.
func init() {
	for lang := range render.Supported {
		_, ok := supported[lang]
		if !ok {
			panic(fmt.Sprintf("bug: we sup[ort lang %q, but writer does not", lang))
		}
	}
}

type Writer struct {
	config          imports.ConfigProvider
	fs              fs.Writer
	vendorDir       string
	forceRegenerate bool
}

type writerOption func(w *Writer)

// WithFS uses the fs passed to write files to.
func WithFS(fs fs.Writer) writerOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

// WithVendorDir sets the vendor directory for lazy loading.
func WithVendorDir(dir string) writerOption {
	return func(w *Writer) {
		w.vendorDir = dir
	}
}

// WithForceRegenerate forces regeneration of all .go files.
func WithForceRegenerate(force bool) writerOption {
	return func(w *Writer) {
		w.forceRegenerate = force
	}
}

// New creates a new Writer.
func New(config imports.ConfigProvider, options ...writerOption) (*Writer, error) {
	fs, err := osfs.New()
	if err != nil {
		return nil, fmt.Errorf("could not create an osfs: %s", err)
	}
	w := &Writer{config: config, fs: fs}
	for _, o := range options {
		o(w)
	}
	return w, nil
}

// Write wrties all rendered content to the appropriate locations for their language.
func (w *Writer) Write(ctx context.Context, rendered []render.Rendered) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, r := range rendered {
		if supported[r.Lang] == nil {
			return fmt.Errorf("bug: writer.Writer does not support language: %v", r.Lang)
		}
	}

	// Organize all renders by language.
	m := map[render.Lang][]render.Rendered{}
	for _, r := range rendered {
		v := m[r.Lang]
		v = append(v, r)
		m[r.Lang] = v
	}

	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)

	for k, v := range m {
		k := k
		v := v

		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			wr := supported[k]
			wr.SetFS(w.fs)

			// If the writer supports lazy loading, configure it
			if lw, ok := wr.(LazyWriter); ok {
				if w.vendorDir != "" {
					lw.SetVendorDir(w.vendorDir)
				}
				if w.forceRegenerate {
					lw.SetForceRegenerate(w.forceRegenerate)
				}
			}

			if err := wr.WriteFiles(ctx, w.config, v); err != nil {
				errCh <- err
				cancel()
			}
		}()
	}
	wg.Wait()
	close(errCh)

	return <-errCh
}
