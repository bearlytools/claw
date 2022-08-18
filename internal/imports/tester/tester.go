package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bearlytools/claw/internal/imports"
)

func main() {
	ctx := context.Background()

	clawPath, err := imports.FindClawFile("../../../testing/imports/vehicles/claw")
	if err != nil {
		exitln(err)
	}

	config := imports.NewConfig()

	log.Println("clawPath is: ", clawPath)
	if err := config.Read(ctx, clawPath); err != nil {
		exitf("error on Read(): %s\n", err)
	}

	for k, imp := range config.Imports {
		log.Println("Package Path: ", k)
		log.Println("\tPath: ", imp.FullPath)
		log.Println("\tExternal: ", imp.External)
		log.Println("\tImports: ")
		for k := range imp.Imports.Imports {
			log.Printf("\t\t%s", k)
		}
	}
	log.Println("success!?")
}

func exitf(s string, i ...any) {
	fmt.Printf(s, i...)
	os.Exit(1)
}

func exitln(i ...any) {
	fmt.Println(i...)
	os.Exit(1)
}
