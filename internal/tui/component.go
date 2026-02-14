package tui

import tea "github.com/charmbracelet/bubbletea"

type KeyMap struct {
	Keys        []string
	Description string
	Condition   func() bool
	Handle      func(tea.KeyMsg) tea.Cmd
}

type Component interface {
	KeyMaps() []KeyMap
	BatchJobs() []tea.Cmd
	tea.Model
}

type StartInsertCmd struct {
	Handle func(tea.KeyMsg) error
}
