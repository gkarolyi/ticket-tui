package main

import (
	"fmt"
	"os"
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

	config, err := NewConfig(EnvMap())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := Run(config); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
