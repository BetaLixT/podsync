package tui

import (
	"strings"

	"github.com/BetaLixT/podsync"
	"github.com/BetaLixT/podsync/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type appTab struct {
	name      string
	component Component
}

type AppComponent struct {
	width          int
	height         int
	tabs           []appTab
	activeTab      int
	markedEpisodes map[string][]podsync.SourceEpisode
	onStatusUpdate func(StatusData)
	deviceStatus   StatusData
	sourceStatus   StatusData
	lgr            podsync.Logger
}

func NewAppComponent(
	cfg *config.Config,
	podcast podsync.PodcastSource,
	device podsync.Device,
	db podsync.PodcastDevice,
	onStatusUpdate func(StatusData),
	lgr podsync.Logger,
) *AppComponent {
	app := &AppComponent{
		0,                                        // width
		0,                                        // height
		nil,                                      // tabs
		0,                                        // activeTab
		make(map[string][]podsync.SourceEpisode), // markedEpisodes
		onStatusUpdate,                           // onStatusUpdate
		StatusData{},                             // deviceStatus
		StatusData{},                             // sourceStatus
		lgr,                                      // logs
	}

	sourceComp := NewSourceComponent(
		podcast,
		app.handleMarkedUpdate,
		app.getMarkedForPodcast,
		app.handleSourceStatus,
	)

	deviceComp := NewDeviceComponent(
		cfg,
		device,
		podcast,
		db,
		app.getMarkedEpisodes,
		app.handleDeviceStatus,
		app.lgr,
	)

	logComp := NewLogComponent(
		app.lgr,
	)

	app.tabs = []appTab{
		{"Source", sourceComp},
		{"Device", deviceComp},
		{"Logs", logComp},
	}

	return app
}

func (a *AppComponent) Init() tea.Cmd {
	return nil
}

func (a *AppComponent) BatchJobs() []tea.Cmd {
	var cmds []tea.Cmd
	for _, tab := range a.tabs {
		cmds = append(cmds, tab.component.BatchJobs()...)
	}
	return cmds
}

func (a *AppComponent) KeyMaps() []KeyMap {
	var kms []KeyMap

	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		kms = append(kms, a.tabs[a.activeTab].component.KeyMaps()...)
	}

	tabCount := len(a.tabs)
	kms = append(kms, KeyMap{
		Keys:        []string{"tab", "shift+tab"},
		Description: "switch tab",
		Condition:   func() bool { return tabCount > 1 },
		Handle: func(km tea.KeyMsg) tea.Cmd {
			if km.String() == "shift+tab" {
				a.activeTab = (a.activeTab - 1 + tabCount) % tabCount
			} else {
				a.activeTab = (a.activeTab + 1) % tabCount
			}
			return nil
		},
	})

	return kms
}

func (a *AppComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		var cmds []tea.Cmd
		for _, tab := range a.tabs {
			_, cmd := tab.component.Update(msg)
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	// Route messages to the appropriate child
	switch msg.(type) {
	case ipodCheckMsg, syncResult, markCompleteResult, []podsync.Episode:
		return a.updateTab("Device", msg)
	case []podsync.SourcePodcast, showEpisodesMsg:
		return a.updateTab("Source", msg)
	}

	return a, nil
}

func (a *AppComponent) updateTab(name string, msg tea.Msg) (tea.Model, tea.Cmd) {
	for _, tab := range a.tabs {
		if tab.name == name {
			_, cmd := tab.component.Update(msg)
			return a, cmd
		}
	}
	return a, nil
}

func (a *AppComponent) View() string {
	var b strings.Builder

	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		b.WriteString(a.tabs[a.activeTab].component.View())
	}

	return b.String()
}

func (a *AppComponent) renderTabBar() string {
	var parts []string
	for i, tab := range a.tabs {
		if i == a.activeTab {
			parts = append(parts, activeTabStyle.Render(" "+tab.name+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+tab.name+" "))
		}
	}
	return strings.Join(parts, tabGapStyle.Render(" "))
}

func (a *AppComponent) CommitFilter() {
	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		if sc, ok := a.tabs[a.activeTab].component.(*SourceComponent); ok {
			sc.CommitFilter()
		}
	}
}

func (a *AppComponent) CancelFilter() {
	if a.activeTab >= 0 && a.activeTab < len(a.tabs) {
		if sc, ok := a.tabs[a.activeTab].component.(*SourceComponent); ok {
			sc.CancelFilter()
		}
	}
}

func (a *AppComponent) handleMarkedUpdate(podcastID string, episodes []podsync.SourceEpisode) {
	if len(episodes) == 0 {
		delete(a.markedEpisodes, podcastID)
	} else {
		a.markedEpisodes[podcastID] = episodes
	}
}

func (a *AppComponent) getMarkedEpisodes() map[string][]podsync.SourceEpisode {
	return a.markedEpisodes
}

func (a *AppComponent) getMarkedForPodcast(podcastID string) []podsync.SourceEpisode {
	return a.markedEpisodes[podcastID]
}

func (a *AppComponent) handleDeviceStatus(data StatusData) {
	a.deviceStatus = data
	a.pushStatus()
}

func (a *AppComponent) handleSourceStatus(data StatusData) {
	a.sourceStatus = data
	a.pushStatus()
}

func (a *AppComponent) pushStatus() {
	if a.onStatusUpdate == nil {
		return
	}
	a.onStatusUpdate(StatusData{
		DeviceConnected: a.deviceStatus.DeviceConnected,
		DevicePath:      a.deviceStatus.DevicePath,
		FreeSpace:       a.deviceStatus.FreeSpace,
		FreeSpaceStr:    a.deviceStatus.FreeSpaceStr,
		EpisodeCount:    a.deviceStatus.EpisodeCount,
		SourceFound:     a.sourceStatus.SourceFound,
		SourcePath:      a.sourceStatus.SourcePath,
		StatusMsg:       a.deviceStatus.StatusMsg,
		StatusStyle:     a.deviceStatus.StatusStyle,
	})
}
