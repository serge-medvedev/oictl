package main

import (
	"context"
	"os"

	"oictl/internal/cli"
)

var version = "dev"

func main() {
	app := cli.New(os.Stdout, os.Stderr, version)
	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
