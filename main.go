package main

import (
	"fmt"
	"os"

	"github.com/synclet-io/synclet/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
