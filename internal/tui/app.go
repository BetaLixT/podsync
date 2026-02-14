package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetaLixT/podsync"
	"github.com/BetaLixT/podsync/internal/config"
	"github.com/BetaLixT/podsync/internal/db"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewType int

const (
	WaitingForIPod = ViewType(iota)
	DeviceEpisodes
	GpodderShows
	GpodderShowEpisodes
	Options
)

type syncResult struct {
	synced int
	err    error
}

type markCompleteResult struct {
	err error
}

type showEpisodesMsg []podsync.Episode

type ipodCheckMsg struct {
	connected bool
	freeSpace uint64
}

type Model struct {
	config             *config.Config
	gpodder            podsync.PodcastSource
	ipod               podsync.Device
	db                 *db.DB
	episodes           *episodeList
	markedShowEpisodes map[int64][]podsync.Episode
	shows              *showList
	showEpisodes       *showEpisodeList
	options            *optionsCtrl
	ipodConnected      bool
	gpodderFound       bool
	currentView        ViewType
	freeSpace          uint64
	width              int
	height             int
	statusMsg          string
	statusStyle        lipgloss.Style
	syncing            bool

	// General input handling
	insertMode  bool
	confirmQuit bool
}

func NewModel(cfg *config.Config, podcast podsync.PodcastSource, device podsync.Device, localDB *db.DB) *Model {
	return &Model{
		config:             cfg,
		gpodder:            podcast,
		ipod:               device,
		db:                 localDB,
		episodes:           newEpisodeList(),
		markedShowEpisodes: map[int64][]podsync.Episode{},
		shows:              newShowList(),
		gpodderFound:       podcast.DatabaseExists(),
		statusStyle:        statusInfoStyle,
		currentView:        WaitingForIPod,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.checkIPod(),
		tea.EnterAltScreen,
	)
}

func (m *Model) checkIPod() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		connected := m.ipod.IsConnected()
		var freeSpace uint64
		if connected {
			freeSpace, _ = m.ipod.GetFreeSpace()
		}
		return ipodCheckMsg{connected: connected, freeSpace: freeSpace}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.episodes.SetHeight(msg.Height - 10)
		m.shows.SetHeight(msg.Height - 10)
		if m.showEpisodes != nil {
			m.showEpisodes.SetHeight(msg.Height - 12)
		}
		return m, nil

	case ipodCheckMsg:
		wasConnected := m.ipodConnected
		m.ipodConnected = msg.connected
		m.freeSpace = msg.freeSpace

		var cmds []tea.Cmd
		cmds = append(cmds, m.checkIPod())

		if msg.connected && !wasConnected {
			m.statusMsg = "iPod connected"
			m.statusStyle = statusSuccessStyle
			cmds = append(cmds, m.loadEpisodes())
			if m.currentView == WaitingForIPod {
				m.currentView = DeviceEpisodes
			}
		} else if !msg.connected && wasConnected {
			m.statusMsg = "iPod disconnected"
			m.statusStyle = statusErrorStyle
			m.episodes.SetEpisodes(nil)
			if m.currentView == DeviceEpisodes {
				m.currentView = WaitingForIPod
			}
		}

		return m, tea.Batch(cmds...)

	case syncResult:
		m.syncing = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Sync failed: %v", msg.err)
			m.statusStyle = statusErrorStyle
		} else {
			m.statusMsg = fmt.Sprintf("Synced %d episodes", msg.synced)
			m.statusStyle = statusSuccessStyle
		}
		if m.showEpisodes != nil {
			m.showEpisodes.ClearMarked()
		}
		return m, m.loadEpisodes()

	case markCompleteResult:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusStyle = statusErrorStyle
		} else {
			m.statusMsg = "Episode marked complete"
			m.statusStyle = statusSuccessStyle
		}
		return m, m.loadEpisodes()

	case []db.Episode:
		m.episodes.SetEpisodes(msg)
		return m, nil

	case []podsync.Podcast:
		m.shows.SetShows(msg)
		return m, nil

	case showEpisodesMsg:
		if m.showEpisodes != nil {
			m.showEpisodes.SetEpisodes([]podsync.Episode(msg))
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	if m.insertMode {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			switch m.currentView {
			case Options:
				m.options.InsertModeSave()
			case GpodderShowEpisodes:
				m.showEpisodes.CommitFilter()
			}
			m.insertMode = false
			return m, nil
		case "esc":
			switch m.currentView {
			case Options:
				m.options.InsertModeCancel()
			case GpodderShowEpisodes:
				m.showEpisodes.CancelFilter()
			}
			m.insertMode = false
			return m, nil
		}

		switch m.currentView {
		case Options:
			_, cmd := m.options.InsertModeUpdate(msg)
			return m, cmd
		case GpodderShowEpisodes:
			cmd := m.showEpisodes.UpdateFilterInput(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.confirmQuit {
		switch msg.String() {
		case "q", "y", "ctrl+c":
			return m, tea.Quit
		default:
			m.confirmQuit = false
			m.statusMsg = ""
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "q":
		m.confirmQuit = true
		m.statusMsg = "Press q again to quit"
		m.statusStyle = syncingStyle
		return m, nil

	case "tab":
		switch m.currentView {
		case DeviceEpisodes, WaitingForIPod:
			m.currentView = GpodderShows
			return m, m.loadShows()
		case GpodderShows:
			if m.ipodConnected {
				m.currentView = DeviceEpisodes
			} else {
				m.currentView = WaitingForIPod
			}
		}
		return m, nil

	case "o":
		switch m.currentView {
		case DeviceEpisodes, WaitingForIPod, GpodderShows:
			m.options = newOptionsCtrl(*m.config)
			m.currentView = Options
		}
		return m, nil

	case "j", "down":
		switch m.currentView {
		case DeviceEpisodes:
			m.episodes.MoveDown()
		case GpodderShows:
			m.shows.MoveDown()
		case GpodderShowEpisodes:
			m.showEpisodes.MoveDown()
		case Options:
			m.options.MoveDown()
		}
		return m, nil

	case "k", "up":
		switch m.currentView {
		case DeviceEpisodes:
			m.episodes.MoveUp()
		case GpodderShows:
			m.shows.MoveUp()
		case GpodderShowEpisodes:
			m.showEpisodes.MoveUp()
		case Options:
			m.options.MoveUp()
		}
		return m, nil

	case "s":
		switch m.currentView {
		case DeviceEpisodes:

			if m.ipodConnected && !m.syncing {

			}
			if m.ipodConnected && !m.syncing {

				m.syncing = true
				marked := []podsync.Episode{}
				for s := range m.markedShowEpisodes {
					marked = append(marked, m.markedShowEpisodes[s]...)
				}
				m.statusStyle = syncingStyle

				if len(marked) > 0 {
					m.statusMsg = "Syncing marked episodes..."
					return m, m.syncMarkedEpisodes(marked)
				}
				m.statusMsg = "Syncing..."
				return m, m.syncEpisodes()
			}
		case GpodderShowEpisodes:
			m.showEpisodes.CycleSort()
		case Options:
			c, err := getConfig[*config.Config](m.options)
			if err == nil {
				err = c.Save()
				if err == nil {
					return m, tea.Quit
				}
			}
		}
		return m, nil

	case "m":
		if m.currentView == DeviceEpisodes && m.ipodConnected && !m.syncing {
			selected := m.episodes.Selected()
			if selected != nil {
				return m, m.markComplete(selected)
			}
		}
		return m, nil

	case "enter":
		switch m.currentView {
		case DeviceEpisodes:
			if m.ipodConnected && !m.syncing {
				selected := m.episodes.Selected()
				if selected != nil {
					return m, m.markComplete(selected)
				}
			}
		case GpodderShows:
			selected := m.shows.Selected()
			if selected != nil {
				m.showEpisodes = newShowEpisodeList(selected.Title)
				m.showEpisodes.SetHeight(m.height - 12)
				m.currentView = GpodderShowEpisodes
				return m, m.loadShowEpisodes(selected.ID)
			}
		}
		return m, nil

	case " ":
		if m.currentView == GpodderShowEpisodes {
			m.showEpisodes.ToggleMark()
		}
		return m, nil

	case "/":
		if m.currentView == GpodderShowEpisodes {
			m.showEpisodes.StartFilter()
			m.insertMode = true
		}
		return m, nil

	case "c":
		if m.currentView == GpodderShowEpisodes {
			m.showEpisodes.ClearFilter()
		}
		return m, nil

	case "i":
		switch m.currentView {
		case Options:
			m.insertMode = true
			m.options.InsertMode()
			return m, nil
		}
		return m, nil

	case "esc":
		switch m.currentView {
		case Options:
			m.options = nil
			if m.ipodConnected {
				m.currentView = DeviceEpisodes
			} else {
				m.currentView = WaitingForIPod
			}
		case GpodderShows:
			if m.ipodConnected {
				m.currentView = DeviceEpisodes
			} else {
				m.currentView = WaitingForIPod
			}
		case GpodderShowEpisodes:
			selected := m.shows.Selected()

			markedEpisodes := m.showEpisodes.GetMarked()
			if len(markedEpisodes) != 0 {
				m.markedShowEpisodes[selected.ID] = markedEpisodes
			} else {
				delete(m.markedShowEpisodes, selected.ID)
			}

			m.showEpisodes = nil
			m.currentView = GpodderShows
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) loadEpisodes() tea.Cmd {
	return func() tea.Msg {
		episodes, err := m.db.GetAllEpisodes()
		if err != nil {
			return []db.Episode{}
		}

		var valid []db.Episode
		for _, ep := range episodes {
			if m.ipod.FileExists(ep.IPodPath) {
				valid = append(valid, ep)
			} else {
				m.db.RemoveEpisode(ep.ID)
			}
		}

		return valid
	}
}

func (m *Model) loadShowEpisodes(podcastID int64) tea.Cmd {
	return func() tea.Msg {
		episodes, err := m.gpodder.GetEpisodesForPodcast(podcastID)
		if err != nil {
			return showEpisodesMsg([]podsync.Episode{})
		}
		return showEpisodesMsg(episodes)
	}
}

func (m *Model) loadShows() tea.Cmd {
	return func() tea.Msg {
		podcasts, err := m.gpodder.GetPodcasts()
		if err != nil {
			return []podsync.Podcast{}
		}
		return podcasts
	}
}

func (m *Model) syncMarkedEpisodes(episodes []podsync.Episode) tea.Cmd {
	return func() tea.Msg {
		var synced int
		for _, ep := range episodes {
			alreadySynced, err := m.db.IsEpisodeSynced(ep.ID)
			if err != nil {
				log.Printf("failed to check if episode synced: %s", err.Error())
				continue
			}
			if alreadySynced {
				log.Printf("episode synced already")
				continue
			}

			srcPath := m.gpodder.GetFullPath(ep)
			destFilename := formatEpisodeFilename(ep)
			destPath, err := m.ipod.CopyFile(srcPath, ep.PodcastTitle, destFilename)
			if err != nil {
				log.Printf("copy failed for %s: %s", destFilename, err.Error())
				continue
			}

			dbEpisode := db.Episode{
				GPodderEpisodeID: ep.ID,
				PodcastName:      ep.PodcastTitle,
				EpisodeTitle:     ep.Title,
				Filename:         destFilename,
				IPodPath:         destPath,
				Duration:         ep.TotalTime,
			}

			if err := m.db.AddEpisode(dbEpisode); err != nil {
				continue
			}

			synced++
		}

		return syncResult{synced: synced}
	}
}

func (m *Model) syncEpisodes() tea.Cmd {
	return func() tea.Msg {
		episodes, err := m.gpodder.GetLatestEpisodesPerPodcast(m.config.EpisodesPerPodcast)
		if err != nil {
			return syncResult{err: fmt.Errorf("failed to get episodes from gPodder: %w", err)}
		}

		var synced int
		for _, ep := range episodes {
			alreadySynced, err := m.db.IsEpisodeSynced(ep.ID)
			if err != nil {
				continue
			}
			if alreadySynced {
				continue
			}

			srcPath := m.gpodder.GetFullPath(ep)
			destFilename := formatEpisodeFilename(ep)
			destPath, err := m.ipod.CopyFile(srcPath, ep.PodcastTitle, destFilename)
			if err != nil {
				continue
			}

			dbEpisode := db.Episode{
				GPodderEpisodeID: ep.ID,
				PodcastName:      ep.PodcastTitle,
				EpisodeTitle:     ep.Title,
				Filename:         destFilename,
				IPodPath:         destPath,
				Duration:         ep.TotalTime,
			}

			if err := m.db.AddEpisode(dbEpisode); err != nil {
				continue
			}

			synced++
		}

		return syncResult{synced: synced}
	}
}

func (m *Model) markComplete(ep *db.Episode) tea.Cmd {
	return func() tea.Msg {
		if err := m.ipod.RemoveFile(ep.IPodPath); err != nil {
			return markCompleteResult{err: err}
		}

		if err := m.gpodder.MarkEpisodePlayed(ep.GPodderEpisodeID); err != nil {
			return markCompleteResult{err: fmt.Errorf("failed to mark as played in gPodder: %w", err)}
		}

		if err := m.db.RemoveEpisode(ep.ID); err != nil {
			return markCompleteResult{err: err}
		}

		return markCompleteResult{}
	}
}

func (m Model) View() string {
	var b strings.Builder

	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	switch m.currentView {
	case WaitingForIPod:
		waitingView := m.renderWaiting()
		b.WriteString(waitingView)
	case DeviceEpisodes:
		episodesView := m.renderEpisodes()
		b.WriteString(episodesView)
	case GpodderShows:
		showsView := m.renderShows()
		b.WriteString(showsView)
	case GpodderShowEpisodes:
		b.WriteString(m.renderShowEpisodes())
	case Options:
		optionsView := m.renderOptions()
		b.WriteString(optionsView)
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderHeader() string {
	var status string
	if m.ipodConnected {
		status = connectedStyle.Render("Connected") + " (" + m.config.IPodMount + ")"
	} else {
		status = disconnectedStyle.Render("Waiting for iPod...")
	}

	var gpodderStatus string
	if m.gpodderFound {
		gpodderStatus = connectedStyle.Render("Found") + " (" + m.gpodder.DatabasePath() + ")"
	} else {
		gpodderStatus = disconnectedStyle.Render("Not found") + " (" + m.gpodder.DatabasePath() + ")"
	}

	title := titleStyle.Render("Podsync")
	ipodStatus := "iPod: " + status
	gpodderLine := "gPodder: " + gpodderStatus

	var info string
	if m.ipodConnected {
		episodeCount := m.episodes.Count()
		freeSpaceStr := m.ipod.FormatBytes(m.freeSpace)
		info = fmt.Sprintf("Episodes: %d | Free: %s", episodeCount, freeSpaceStr)
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

func (m Model) renderWaiting() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := "\n\n" + subtitleStyle.Render("    Waiting for iPod to be connected...") + "\n\n"
	content += subtitleStyle.Render("    Mount path: "+m.config.IPodMount) + "\n\n"

	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderOptions() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := m.options.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderShowEpisodes() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := m.showEpisodes.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderShows() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := m.shows.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderEpisodes() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := m.episodes.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderHelp() string {

	if m.insertMode {
		return keyStyle.Render("Enter") + helpStyle.Render(" confirm") + helpStyle.Render("  ") +
			keyStyle.Render("Esc") + helpStyle.Render(" back") + helpStyle.Render("  ") +
			keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	}
	switch m.currentView {
	case WaitingForIPod:
		return keyStyle.Render("o") + helpStyle.Render(" options") + helpStyle.Render("  ") +
			keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	case DeviceEpisodes:
		syncType := "sync"
		if len(m.markedShowEpisodes) != 0 {
			syncType = "sync marked"
		}
		return keyStyle.Render("s") + helpStyle.Render(" ") + helpStyle.Render(syncType) +
			helpStyle.Render("  ") +
			keyStyle.Render("m") + helpStyle.Render("/") + keyStyle.Render("Enter") + helpStyle.Render(" mark complete") +
			helpStyle.Render("  ") +
			keyStyle.Render("j") + helpStyle.Render("/") + keyStyle.Render("k") + helpStyle.Render("/") + keyStyle.Render("arrows") + helpStyle.Render(" navigate") +
			helpStyle.Render("  ") +
			keyStyle.Render("Tab") + helpStyle.Render(" shows") +
			helpStyle.Render("  ") +
			keyStyle.Render("o") + helpStyle.Render(" options") +
			helpStyle.Render("  ") +
			keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	case GpodderShows:
		return keyStyle.Render("j") + helpStyle.Render("/") + keyStyle.Render("k") + helpStyle.Render("/") + keyStyle.Render("arrows") + helpStyle.Render(" navigate") +
			helpStyle.Render("  ") +
			keyStyle.Render("Enter") + helpStyle.Render(" open") +
			helpStyle.Render("  ") +
			keyStyle.Render("Tab") + helpStyle.Render("/") + keyStyle.Render("Esc") + helpStyle.Render(" back") +
			helpStyle.Render("  ") +
			keyStyle.Render("o") + helpStyle.Render(" options") +
			helpStyle.Render("  ") +
			keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	case GpodderShowEpisodes:
		return keyStyle.Render("j") + helpStyle.Render("/") + keyStyle.Render("k") + helpStyle.Render("/") + keyStyle.Render("arrows") + helpStyle.Render(" navigate") +
			helpStyle.Render("  ") +
			keyStyle.Render("Space") + helpStyle.Render(" mark") +
			helpStyle.Render("  ") +
			keyStyle.Render("s") + helpStyle.Render(" sort") +
			helpStyle.Render("  ") +
			keyStyle.Render("/") + helpStyle.Render(" filter") +
			helpStyle.Render("  ") +
			keyStyle.Render("c") + helpStyle.Render(" clear") +
			helpStyle.Render("  ") +
			keyStyle.Render("Esc") + helpStyle.Render(" back") +
			helpStyle.Render("  ") +
			keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	case Options:
		return keyStyle.Render("j") + helpStyle.Render("/") + keyStyle.Render("k") + helpStyle.Render("/") + keyStyle.Render("arrows") + helpStyle.Render(" navigate") +
			helpStyle.Render("  ") +
			keyStyle.Render("Esc") + helpStyle.Render(" back") + helpStyle.Render("  ") +
			keyStyle.Render("i") + helpStyle.Render(" edit") + helpStyle.Render("  ") +
			keyStyle.Render("s") + helpStyle.Render(" save") + helpStyle.Render("  ") +
			keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	}

	return keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
}

func formatEpisodeFilename(ep podsync.Episode) string {
	ext := filepath.Ext(ep.DownloadFilename)
	published := time.Unix(ep.Published, 0)
	date := fmt.Sprintf("%02d-%02d", published.Day(), published.Month())
	title := sanitizeFilename(ep.Title)
	return fmt.Sprintf("%02d - %s - %s%s", ep.EpisodeNumber, title, date, ext)
}

func sanitizeFilename(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch c {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			result = append(result, '_')
		default:
			result = append(result, c)
		}
	}
	return string(result)
}

func Run(cfg *config.Config, podcast podsync.PodcastSource, device podsync.Device, localDB *db.DB) error {
	model := NewModel(cfg, podcast, device, localDB)

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		return err
	}
	defer f.Close()

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
