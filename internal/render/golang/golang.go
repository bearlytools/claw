// Package golang implements the Go language renderer.
package golang

import (
	"bytes"
	"context"
	"embed"
	"text/template"

	"github.com/bearlytools/claw/internal/idl"
	"github.com/bearlytools/claw/internal/render"
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

// Renderer implements render.Renderer for the Go language.
type Renderer struct{}

// Render implements render.Renderer.Render().
func (r Renderer) Render(ctx context.Context, file *idl.File) ([]byte, error) {
	buff := bytes.Buffer{}
	if err := templates.ExecuteTemplate(&buff, "claw.tmpl", file); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}
