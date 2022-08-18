package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/bearlytools/claw/internal/imports"
	"github.com/bearlytools/claw/internal/render"
	"github.com/bearlytools/claw/internal/writer"

	osfs "github.com/gopherfs/fs/io/os"

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

	// Mount our filesystem for reading.
	fs, err := osfs.New()
	if err != nil {
		panic(err)
	}

	clawFile, err := imports.FindClawFile(fs, path)
	if err != nil {
		exitf("problem finding .claw file: %s", err)
	}

	config := imports.NewConfig()
	if err := config.Read(ctx, clawFile); err != nil {
		exitf("error: %s\n", err)
	}

	rendered, err := render.Render(ctx, config, render.Go)
	if err != nil {
		exit(err)
	}
	wr, err := writer.New(config)
	if err != nil {
		exit(err)
	}
	if err := wr.Write(ctx, rendered); err != nil {
		exit(err)
	}
}

func exit(i ...any) {
	fmt.Println(i...)
	os.Exit(1)
}

func exitf(s string, i ...any) {
	fmt.Printf(s+"\n", i...)
	os.Exit(1)
}
