package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/idl"
	"github.com/bearlytools/claw/internal/render"
	"github.com/johnsiilver/halfpike"

	// Registers the golang renderer.
	_ "github.com/bearlytools/claw/internal/render/golang"
)

func main() {
	ctx := context.Background()

	flag.Parse()
	args := flag.Args()
	path := ""
	if len(args) == 0 {
		path = "."
	} else {
		path = args[0]
	}

	clawFile := ""
	files, err := os.ReadDir(path)
	if err != nil {
		exitf("could not read the directory: %s", err)
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) == ".claw" {
			if clawFile != "" {
				exitf("error: there is more than one .claw file in the directory")
			}
			clawFile = f.Name()
		}
	}
	if clawFile == "" {
		exitf("error: did not find a .claw file in the path")
	}
	fp := filepath.Join(path, clawFile)
	content, err := os.ReadFile(fp)
	if err != nil {
		exitf("error: problem reading file %s: %s", fp, err)
	}
	file := idl.New()

	if err := halfpike.Parse(ctx, conversions.ByteSlice2String(content), file); err != nil {
		exit(err)
	}

	results, err := render.Render(context.Background(), file, render.Go)
	if err != nil {
		exit(err)
	}
	os.Stdout.Write(results[0].Native)
}

func exit(i ...any) {
	fmt.Println(i...)
	os.Exit(1)
}

func exitf(s string, i ...any) {
	fmt.Printf(s+"\n", i...)
	os.Exit(1)
}
