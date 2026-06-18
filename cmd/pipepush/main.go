package main

import (
	"fmt"
	"os"

	"github.com/Gerry3010/pipepush/internal/cli"
)

// version is overridden at build time: -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
