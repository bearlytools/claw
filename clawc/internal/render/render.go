// Package render sets up the interface for rendering a .claw file to a language native representation.
// It also supports registering the handlers of those renderers (which are in other packages).
package render

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/clawc/internal/imports"
	"github.com/bearlytools/claw/clawc/languages/go/conversions"
)

// Lang represents a programming language we can render a from a .claw file.
type Lang uint8

const (
	Unknown Lang = 0
	Go      Lang = 1
)

// Supported is langauges that we have registered support for.
var Supported = map[Lang]Renderer{}

// Renderer renders a language native file from a .claw file.
type Renderer interface {
	Render(ctx context.Context, config *imports.Config, path string) ([]byte, error)
}

// MultiFileRenderer is an optional interface that renderers can implement
// to generate additional files beyond the main output.
type MultiFileRenderer interface {
	Renderer
	// RenderAdditionalFiles returns additional files to generate.
	// The map keys are filenames (e.g., "clawiter.go"), values are file contents.
	RenderAdditionalFiles(ctx context.Context, config *imports.Config, path string) (map[string][]byte, error)
}

// Rendered represents rendered output for a language.
type Rendered struct {
	// Package is the Claw package this represents.
	Package string
	// RepoVersion is the version the repo is at.
	RepoVersion string
	// Path is the path in the local filesystem that source .claw file can be found at.
	Path string
	// Lang is the language this is for.
	Lang Lang
	// Native is the output for the language (main file).
	Native []byte
	// Files contains additional generated files (filename -> content).
	// The main file is in Native, additional files like clawiter.go are here.
	Files map[string][]byte
}

// Render is used to render a set of languages from the .claw file.
func Render(ctx context.Context, config *imports.Config, langs ...Lang) ([]Rendered, error) {
	out := make([]Rendered, 0, len(langs)*len(config.Imports))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, l := range langs {
		_, ok := Supported[l]
		if !ok {
			return nil, fmt.Errorf("language %v is not supported", langs[i])
		}
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	errCh := make(chan error, 1)

	for i := 0; i < len(langs); i++ {
		lang := langs[i]
		renderer := Supported[lang]

		for _, f := range config.Imports {
			pkg := f.Package
			path := f.FullPath
			repoVersion := f.RepoVersion

			wg.Add(1)
			go func() {
				defer wg.Done()
				b, err := renderer.Render(ctx, config, path)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
				b = cleanImports(b)

				result := Rendered{
					Package:     pkg,
					RepoVersion: repoVersion,
					Path:        path,
					Lang:        lang,
					Native:      b,
				}

				// Check if renderer supports additional files
				if mfr, ok := renderer.(MultiFileRenderer); ok {
					files, err := mfr.RenderAdditionalFiles(ctx, config, path)
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
					// Clean imports in additional files too
					for filename, content := range files {
						files[filename] = cleanImports(content)
					}
					result.Files = files
				}

				mu.Lock()
				out = append(out, result)
				mu.Unlock()
			}()
		}
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	return out, nil
}

type importCheck struct {
	path string
	find string
}

var findImports = []importCheck{
	{"github.com/bearlytools/claw/clawc/languages/go/mapping", "mapping."},
	{"github.com/bearlytools/claw/languages/go/reflect", "reflect."},
	{"github.com/bearlytools/claw/languages/go/reflect/runtime", "runtime."},
	{"github.com/bearlytools/claw/clawc/languages/go/conversions", "conversions."},
	{"github.com/bearlytools/claw/clawc/languages/go/field", "field."},
	{"github.com/bearlytools/claw/clawc/languages/go/clawiter", "clawiter."},
}

// cleanImports is a crap way to do this, but it does work and I'm being lazy.
// So now we are going to do a two pass cleaning and completely brute force.
// I SHOULD FEEL BAD ABOUT THIS!
// If you come across this code, certainly don't copy it.
func cleanImports(b []byte) []byte {
	lines := bytes.SplitAfter(b, []byte("\n"))
	remove := map[string]bool{}
	for _, ic := range findImports {
		remove[ic.path] = true
	}

	for _, line := range lines {
		for _, ic := range findImports {
			if bytes.Contains(line, conversions.UnsafeGetBytes(ic.find)) {
				delete(remove, ic.path)
			}
		}
	}

	out := &bytes.Buffer{}
	for _, line := range lines {
		without := bytes.TrimSpace(line)
		without = bytes.TrimPrefix(without, []byte(`"`))
		without = bytes.TrimSuffix(without, []byte(`"`))
		if !remove[conversions.ByteSlice2String(without)] {
			out.Write(line)
		}
	}
	return out.Bytes()
}
