package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

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
	Handle func(tea.KeyMsg) tea.Cmd
}

// keyAliases maps display-friendly key names to their bubbletea KeyMsg.String() values.
var keyAliases = map[string]string{
	"space": " ",
}

// matchKey checks if a keymap key matches a bubbletea key string, handling aliases.
func matchKey(keymapKey string, msgKey string) bool {
	if keymapKey == msgKey {
		return true
	}
	if alias, ok := keyAliases[keymapKey]; ok {
		return alias == msgKey
	}
	return false
}

type StatusData struct {
	DeviceConnected bool
	DeviceName      string
	DevicePath      string
	FreeSpace       uint64
	FreeSpaceStr    string
	EpisodeCount    int
	SourceFound     bool
	SourceName      string
	SourcePath      string
	StatusMsg       string
	StatusStyle     string // "info", "success", "error", "syncing"
}
