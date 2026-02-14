package tui

import (
	"fmt"
	"strings"

	"github.com/BetaLixT/podsync/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Device string

const (
	IPodRockBoxDevice = Device("iPod Rockbox")
)

type PodcastSource string

const (
	GPodderSource = PodcastSource("gPodder")
)

type RootContentComponent int

const (
	AppRootView = RootContentComponent(iota)
	OptionsRootView
)

type RootComponent struct {
	width            int
	height           int
	statusMsg        string
	statusStyle      lipgloss.Style
	statusData       StatusData
	confirmQuit      bool
	insertMode       *InsertMode
	config           *config.Config
	currentDevice    Device
	currentSource    PodcastSource
	childComponents  map[RootContentComponent]Component
	currentComponent RootContentComponent
}

type InsertMode struct {
	*StartInsertCmd
}

func NewRoot(
	cfg *config.Config,
	deviceType Device,
	sourceType PodcastSource,
	children map[RootContentComponent]Component,
) *RootComponent {
	return &RootComponent{
		0,               // width
		0,               // height
		"",              // statusMsg
		statusInfoStyle, // statusStyle
		StatusData{},    // statusData
		false,           // confirmQuit
		nil,             // insertMode
		cfg,             // config
		deviceType,      // currentDevice
		sourceType,      // currentSource
		children,        // childComponents
		AppRootView,     // currentComponent
	}
}

func (r *RootComponent) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, comp := range r.childComponents {
		cmds = append(cmds, comp.BatchJobs()...)
	}
	cmds = append(cmds, tea.EnterAltScreen)
	return tea.Batch(cmds...)
}

func (r *RootComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return r.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		// Propagate to all children
		for _, comp := range r.childComponents {
			comp.Update(msg)
		}
		return r, nil

	case StartInsertCmd:
		r.insertMode = &InsertMode{&msg}
		return r, nil

	case StatusData:
		r.statusData = msg
		return r, nil
	}

	// Forward non-key messages to active child
	comp, ok := r.childComponents[r.currentComponent]
	if ok {
		_, cmd := comp.Update(msg)
		return r, cmd
	}

	return r, nil
}

func (r *RootComponent) KeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"q", "ctrl+c"},
			Description: "quit",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				if km.String() == "ctrl+c" {
					return tea.Quit
				}
				r.confirmQuit = true
				r.statusMsg = "Press q again to quit"
				r.statusStyle = syncingStyle
				return nil
			},
		},
		{
			Keys:        []string{"o"},
			Description: "options",
			Condition:   func() bool { return r.currentComponent != OptionsRootView },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				r.currentComponent = OptionsRootView
				return nil
			},
		},
		{
			Keys:        []string{"esc"},
			Description: "back",
			Condition:   func() bool { return r.currentComponent == OptionsRootView },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				r.currentComponent = AppRootView
				return nil
			},
		},
	}
}

func (r *RootComponent) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Insert mode: delegate to handler, except control keys
	if r.insertMode != nil {
		switch key {
		case "ctrl+c":
			return r, tea.Quit
		case "enter":
			comp := r.childComponents[r.currentComponent]
			if app, ok := comp.(*AppComponent); ok {
				app.CommitFilter()
			} else if opts, ok := comp.(*OptionsComponent); ok {
				opts.CommitInsert()
			}
			r.insertMode = nil
			return r, nil
		case "esc":
			comp := r.childComponents[r.currentComponent]
			if app, ok := comp.(*AppComponent); ok {
				app.CancelFilter()
			} else if opts, ok := comp.(*OptionsComponent); ok {
				opts.CancelInsert()
			}
			r.insertMode = nil
			return r, nil
		default:
			var cmd tea.Cmd
			if r.insertMode.Handle != nil {
				cmd = r.insertMode.Handle(msg)
			}
			return r, cmd
		}
	}

	// Confirm quit
	if r.confirmQuit {
		switch key {
		case "q", "y", "ctrl+c":
			return r, tea.Quit
		default:
			r.confirmQuit = false
			r.statusMsg = ""
			return r, nil
		}
	}

	// Collect all keymaps: active child first, then root
	var allKeyMaps []KeyMap
	if comp, ok := r.childComponents[r.currentComponent]; ok {
		allKeyMaps = append(allKeyMaps, comp.KeyMaps()...)
	}
	allKeyMaps = append(allKeyMaps, r.KeyMaps()...)

	for _, km := range allKeyMaps {
		if km.Condition != nil && !km.Condition() {
			continue
		}
		for _, k := range km.Keys {
			if matchKey(k, key) {
				cmd := km.Handle(msg)
				return r, cmd
			}
		}
	}

	return r, nil
}

func (r *RootComponent) View() string {
	var b strings.Builder

	b.WriteString(r.renderHeader())
	b.WriteString("\n")

	if comp, ok := r.childComponents[r.currentComponent]; ok {
		b.WriteString(comp.View())
	}

	b.WriteString("\n")
	b.WriteString(r.renderHelp())

	return b.String()
}

func (r *RootComponent) renderHeader() string {
	deviceName := string(r.currentDevice)
	sourceName := string(r.currentSource)

	var deviceStatus string
	if r.statusData.DeviceConnected {
		deviceStatus = connectedStyle.Render("Connected") + " (" + r.statusData.DevicePath + ")"
	} else {
		path := r.config.IPodMount
		if r.statusData.DevicePath != "" {
			path = r.statusData.DevicePath
		}
		deviceStatus = disconnectedStyle.Render("Waiting...") + " (" + path + ")"
	}

	var sourceStatus string
	if r.statusData.SourceFound {
		sourceStatus = connectedStyle.Render("Found") + " (" + r.statusData.SourcePath + ")"
	} else {
		sourceStatus = disconnectedStyle.Render("Not found") + " (" + r.statusData.SourcePath + ")"
	}

	title := titleStyle.Render("Podsync")
	deviceLine := deviceName + ": " + deviceStatus
	sourceLine := sourceName + ": " + sourceStatus

	var info string
	if r.statusData.DeviceConnected {
		info = fmt.Sprintf("Episodes: %d | Free: %s", r.statusData.EpisodeCount, r.statusData.FreeSpaceStr)
	}

	width := r.width
	if width == 0 {
		width = 60
	}

	headerContent := title + "\n" + deviceLine + "\n" + sourceLine
	if info != "" {
		headerContent += "\n" + subtitleStyle.Render(info)
	}
	if r.statusMsg != "" {
		headerContent += "\n" + r.statusStyle.Render(r.statusMsg)
	}

	return boxStyle.Width(width - 2).Render(headerContent)
}

func (r *RootComponent) renderHelp() string {
	if r.insertMode != nil {
		return keyStyle.Render("Enter") + helpStyle.Render(" confirm") + helpStyle.Render("  ") +
			keyStyle.Render("Esc") + helpStyle.Render(" back") + helpStyle.Render("  ") +
			keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	}

	// Collect all keymaps: active child first, then root
	var allKeyMaps []KeyMap
	if comp, ok := r.childComponents[r.currentComponent]; ok {
		allKeyMaps = append(allKeyMaps, comp.KeyMaps()...)
	}
	allKeyMaps = append(allKeyMaps, r.KeyMaps()...)

	var parts []string
	seenDesc := make(map[string]bool)
	seenKeys := make(map[string]bool)
	for _, km := range allKeyMaps {
		if km.Condition != nil && !km.Condition() {
			continue
		}
		if seenDesc[km.Description] {
			// Still claim the keys even if we skip rendering
			for _, k := range km.Keys {
				seenKeys[k] = true
			}
			continue
		}
		// Skip if all keys are already claimed by a higher-priority keymap
		allClaimed := true
		for _, k := range km.Keys {
			if !seenKeys[k] {
				allClaimed = false
				break
			}
		}
		if allClaimed {
			continue
		}

		seenDesc[km.Description] = true
		for _, k := range km.Keys {
			seenKeys[k] = true
		}

		keyParts := make([]string, len(km.Keys))
		for i, k := range km.Keys {
			keyParts[i] = keyStyle.Render(k)
		}
		parts = append(parts, strings.Join(keyParts, helpStyle.Render("/"))+helpStyle.Render(" "+km.Description))
	}

	return strings.Join(parts, helpStyle.Render("  "))
}
