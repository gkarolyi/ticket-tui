package main

import (
	"fmt"
	"os"

	"github.com/wedow/ticket/internal/tui"
)

const description = "tk-plugin: Interactive terminal UI"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--tk-describe":
			fmt.Println(description)
			return
		case "--help", "-h":
			fmt.Println(description)
			fmt.Println()
			fmt.Println("Usage: tk tui")
			fmt.Println()
			fmt.Println("Browse, view, edit, and update tickets in an interactive terminal UI.")
			return
		}
	}

	config, err := tui.NewConfig(tui.EnvMap())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := tui.Run(config); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
