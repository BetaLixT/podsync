package tui

import (
	"fmt"
	"strings"

	"github.com/BetaLixT/podsync"
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
	width       int
	height      int
	statusMsg   string
	statusStyle lipgloss.Style

	deviceConnected bool
	sourceFound     bool

	currentDevice    Device
	currentSource    PodcastSource
	childComponents  map[RootContentComponent]Component
	currentComponent RootContentComponent
	keyMaps          []KeyMap
	insertMode       *InsertMode

	config *config.Config
	source podsync.PodcastSource
	device podsync.Device
}

func (m RootComponent) Init() tea.Cmd {

	m.childComponents = map[RootContentComponent]Component{
		AppRootView:     newComponent(),
		OptionsRootView: newComponent(),
	}
	m.keyMaps = []KeyMap{
		{
			[]string{"o"},
			"options",
			func() bool {
				return m.currentComponent != OptionsRootView
			},
			func(km tea.KeyMsg) tea.Cmd {
				m.currentComponent = OptionsRootView
				return nil
			},
		},
		{
			[]string{"ctrl+c", "q"},
			"quit",
			func() bool {
				return true
			},
			func(km tea.KeyMsg) tea.Cmd {
				return tea.Quit
			},
		},
	}

	batchJobs := []tea.Cmd{}
	for k := range m.childComponents {
		batchJobs = append(batchJobs, m.childComponents[k].BatchJobs()...)
	}
	batchJobs = append(batchJobs, tea.EnterAltScreen)
	return tea.Batch(
		batchJobs...,
	)
}

func (m RootComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:

		// The rune length check is to skip things like ctrl+
		// not sure if this is the best way to do it
		if m.insertMode != nil && len(msg.Runes) == 1 {
			err := m.insertMode.Handle(msg)
			if err != nil {
				m.statusMsg = err.Error()
				m.statusStyle = statusErrorStyle
			}
		}
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if comp, ok := m.childComponents[m.currentComponent]; ok {
			comp.Update(msg.Height - 10)
		}
		return m, nil
	case StartInsertCmd:
		m.insertMode = &InsertMode{&msg}
	}

	return m, nil
}

func (m RootComponent) View() string {
	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Main Content
	if comp, ok := m.childComponents[m.currentComponent]; ok {
		b.WriteString(comp.View())
	}

	// Hints
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	// Logs

	return b.String()
}

func (m RootComponent) GetKeyMaps() KeyMap {}

func (m RootComponent) renderHeader() string {

	// TODO: eventually redo this
	var status string
	if m.deviceConnected {
		status = connectedStyle.Render("Connected") + " (" + m.config.IPodMount + ")"
	} else {
		status = disconnectedStyle.Render("Waiting for iPod...")
	}

	var gpodderStatus string
	if m.sourceFound {
		gpodderStatus = connectedStyle.Render("Found") + " (" + m.source.DatabasePath() + ")"
	} else {
		gpodderStatus = disconnectedStyle.Render("Not found") + " (" + m.source.DatabasePath() + ")"
	}

	title := titleStyle.Render("Podsync")
	ipodStatus := "iPod: " + status
	gpodderLine := "gPodder: " + gpodderStatus

	var info string
	if m.deviceConnected {
		// episodeCount := m.episodes.Count()
		freeSpaceStr := m.device.FormatBytes(m.freeSpace)
		info = fmt.Sprintf("Episodes: %d | Free: %s", "-", freeSpaceStr)
	}

	width := m.width
	if width == 0 {
		width = 60
	}

	headerContent := title + "\n" + ipodStatus + "\n" + gpodderLine
	if info != "" {
		headerContent += "\n" + subtitleStyle.Render(info)
	}
	if m.statusMsg != "" {
		headerContent += "\n" + m.statusStyle.Render(m.statusMsg)
	}

	return boxStyle.Width(width - 2).Render(headerContent)
}

// -- Key map
func (r *RootComponent) handleKeyPress(
	k tea.KeyMsg,
) (tea.Model, tea.Cmd) {
	comp, ok := r.childComponents[r.currentComponent]
	km := []KeyMap{}
	if ok {
		km = r.childComponents[comp].KeyMaps()
	}

}

type InsertMode struct {
	*StartInsertCmd
}
