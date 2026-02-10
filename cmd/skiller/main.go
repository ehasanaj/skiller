package main

import (
	"log"

	"skiller/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	model, err := ui.NewModel()
	if err != nil {
		log.Fatalf("failed to initialize skiller: %v", err)
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatalf("skiller exited with error: %v", err)
	}
}
