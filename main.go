package main

import (
	"os"

	"github.com/garygentry/dotfiles/cmd/dotfiles"
)

func main() {
	if err := dotfiles.Execute(); err != nil {
		os.Exit(1)
	}
}
