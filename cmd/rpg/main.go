package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/uinjad/AzureNights2/internal/app"
	"github.com/uinjad/AzureNights2/internal/content"
	"github.com/uinjad/AzureNights2/internal/storage"
	"github.com/uinjad/AzureNights2/internal/tui"
)

// version is injected at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("AzureNights", version)
		return
	}

	reg, err := content.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading content:", err)
		os.Exit(1)
	}

	session := app.New(reg, storage.NewFileRepo(savePath()))

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
