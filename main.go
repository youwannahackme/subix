package main

import (
	"os"

	"github.com/youwannahackme/subix/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
