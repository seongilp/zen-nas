package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"zenfs/ui"
	"zenfs/zfs"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("zenfs", version)
		return
	}
	app := ui.NewApp(zfs.ExecRunner{})
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "zenfs:", err)
		os.Exit(1)
	}
}
