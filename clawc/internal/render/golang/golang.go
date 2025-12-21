// Package golang implements the Go language renderer.
package golang

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/clawc/internal/idl"
	"github.com/bearlytools/claw/clawc/internal/imports"
	"github.com/bearlytools/claw/clawc/internal/render"
)

//go:embed templates/*
var f embed.FS
var templates *template.Template

func init() {
	t, err := template.ParseFS(f, "templates/*.tmpl")
	if err != nil {
		panic(err)
	}
	templates = t

	if _, ok := render.Supported[render.Go]; ok {
		panic("someone alread registered the Go language renderer")
	}
	render.Supported[render.Go] = Renderer{}
}

type templateData struct {
	Path   string
	Config *imports.Config
	File   *idl.File
}

// Renderer implements render.Renderer for the Go language.
type Renderer struct{}

// Render implements render.Renderer.Render().
func (r Renderer) Render(ctx context.Context, config *imports.Config, path string) ([]byte, error) {
	buff := bytes.Buffer{}

	f, ok := config.Imports[path]
	if !ok {
		return nil, fmt.Errorf("could not find import path %q in config.Imports", path)
	}

	data := templateData{
		Path:   path,
		Config: config,
		File:   f,
	}

	if err := templates.ExecuteTemplate(&buff, "claw.tmpl", data); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// RenderAdditionalFiles implements render.MultiFileRenderer.
func (r Renderer) RenderAdditionalFiles(ctx context.Context, config *imports.Config, path string) (map[string][]byte, error) {
	f, ok := config.Imports[path]
	if !ok {
		return nil, fmt.Errorf("could not find import path %q in config.Imports", path)
	}

	// Only generate clawiter.go if there are structs
	if len(f.Structs()) == 0 {
		return nil, nil
	}

	data := templateData{
		Path:   path,
		Config: config,
		File:   f,
	}

	buff := bytes.Buffer{}
	if err := templates.ExecuteTemplate(&buff, "clawiter.tmpl", data); err != nil {
		return nil, fmt.Errorf("error rendering clawiter.tmpl: %w", err)
	}

	return map[string][]byte{
		"clawiter.go": buff.Bytes(),
	}, nil
}
