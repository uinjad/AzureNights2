package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/storage"
	"github.com/uinjad/AzureNights2/internal/tui"
)

// main is the composition root: it wires the data, logic, persistence, and UI
// adapters together. This is the one place that knows about all the layers.
func main() {
	reg, err := content.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading content:", err)
		os.Exit(1)
	}

	session := app.New(reg, storage.NewFileRepo(savePath()))
	if err := session.NewGame("Aria"); err != nil {
		fmt.Fprintln(os.Stderr, "starting game:", err)
		os.Exit(1)
	}

	program := tea.NewProgram(tui.New(session), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}

func savePath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "azurenights", "save.json")
	}
	return "azurenights-save.json"
}
