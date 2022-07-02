// Package render sets up the interface for rendering a .claw file to a language native representation.
// It also supports registering the handlers of those renderers (which are in other packages).
package render

import (
	"context"
	"fmt"
	"sync"

	"github.com/bearlytools/claw/internal/idl"
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
	Render(ctx context.Context, file *idl.File) ([]byte, error)
}

// Rendered represents rendered output for a language.
type Rendered struct {
	// Lang is the language this is for.
	Lang Lang
	// Native is the output for the language.
	Native []byte
}

// Render is used to render a set of languages from the .claw file.
func Render(ctx context.Context, file *idl.File, langs ...Lang) ([]Rendered, error) {
	out := make([]Rendered, len(langs))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)
	for i := 0; i < len(langs); i++ {
		lang := langs[i]
		i := i

		r, ok := Supported[lang]
		if !ok {
			cancel()
			wg.Wait()
			return nil, fmt.Errorf("language %v is not supported", langs[i])
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := r.Render(ctx, file)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
			out[i] = Rendered{Lang: lang, Native: b}
		}()
	}
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	return out, nil
}
