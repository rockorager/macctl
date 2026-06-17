package main

import (
	"fmt"
	"os"

	"go.rockorager.dev/macctl/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
