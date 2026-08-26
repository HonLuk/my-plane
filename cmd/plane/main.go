package main

import (
	"errors"
	"os"

	"github.com/HonLuk/my-plane/internal/commands"
	"github.com/HonLuk/my-plane/internal/output"
)

func main() {
	renderer := output.NewRenderer(os.Stdout, os.Stderr)
	if err := commands.Run(os.Args[1:], renderer); err != nil {
		var exitError *commands.ExitError
		if !errors.As(err, &exitError) || !exitError.Silent {
			renderer.Errorln(renderer.Red(err.Error()))
		}
		if exitError != nil && exitError.Code != 0 {
			os.Exit(exitError.Code)
		}
		os.Exit(1)
	}
}
