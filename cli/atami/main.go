package main

import (
	"os"

	"github.com/atami-ai/atami-ai/cli/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:], cmd.Options{}))
}
