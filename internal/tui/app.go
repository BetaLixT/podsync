package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/BetaLixT/podsync/internal/config"
	"github.com/BetaLixT/podsync/internal/db"
	"github.com/BetaLixT/podsync/internal/gpodder"
	"github.com/BetaLixT/podsync/internal/ipod"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type syncResult struct {
	synced int
	err    error
}

type markCompleteResult struct {
	err error
}

type ipodCheckMsg struct {
	connected bool
	freeSpace uint64
}

type Model struct {
	config        *config.Config
	gpodder       *gpodder.Client
	ipod          *ipod.IPod
	db            *db.DB
	episodes      *episodeList
	ipodConnected bool
	freeSpace     uint64
	width         int
	height        int
	statusMsg     string
	statusStyle   lipgloss.Style
	syncing       bool
}

func NewModel(cfg *config.Config) (*Model, error) {
	localDB := db.New(cfg.PodsyncDatabase())
	if err := localDB.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	m := &Model{
		config:      cfg,
		gpodder:     gpodder.New(cfg.GPodderHome),
		ipod:        ipod.New(cfg.IPodMount, cfg.PodcastFolder),
		db:          localDB,
		episodes:    newEpisodeList(),
		statusStyle: statusInfoStyle,
	}

	return m, nil
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
		} else if !msg.connected && wasConnected {
			m.statusMsg = "iPod disconnected"
			m.statusStyle = statusErrorStyle
			m.episodes.SetEpisodes(nil)
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
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.ipodConnected && !m.syncing {
			m.episodes.MoveDown()
		}
		return m, nil

	case "k", "up":
		if m.ipodConnected && !m.syncing {
			m.episodes.MoveUp()
		}
		return m, nil

	case "s":
		if m.ipodConnected && !m.syncing {
			m.syncing = true
			m.statusMsg = "Syncing..."
			m.statusStyle = syncingStyle
			return m, m.syncEpisodes()
		}
		return m, nil

	case "m", "enter":
		if m.ipodConnected && !m.syncing {
			selected := m.episodes.Selected()
			if selected != nil {
				return m, m.markComplete(selected)
			}
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

			srcPath := m.gpodder.GetFullPath(ep.DownloadFilename)
			destPath, err := m.ipod.CopyFile(srcPath, ep.PodcastTitle, ep.DownloadFilename)
			if err != nil {
				continue
			}

			dbEpisode := db.Episode{
				GPodderEpisodeID: ep.ID,
				PodcastName:      ep.PodcastTitle,
				EpisodeTitle:     ep.Title,
				Filename:         ep.DownloadFilename,
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

	if !m.ipodConnected {
		waitingView := m.renderWaiting()
		b.WriteString(waitingView)
	} else {
		episodesView := m.renderEpisodes()
		b.WriteString(episodesView)
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

	title := titleStyle.Render("Podsync")
	ipodStatus := "iPod: " + status

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

	headerContent := title + "\n" + ipodStatus
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

func (m Model) renderEpisodes() string {
	width := m.width
	if width == 0 {
		width = 60
	}

	content := m.episodes.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (m Model) renderHelp() string {
	if !m.ipodConnected {
		return keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")
	}

	help := keyStyle.Render("s") + helpStyle.Render(" sync") +
		helpStyle.Render("  ") +
		keyStyle.Render("m") + helpStyle.Render("/") + keyStyle.Render("Enter") + helpStyle.Render(" mark complete") +
		helpStyle.Render("  ") +
		keyStyle.Render("j") + helpStyle.Render("/") + keyStyle.Render("k") + helpStyle.Render("/") + keyStyle.Render("arrows") + helpStyle.Render(" navigate") +
		helpStyle.Render("  ") +
		keyStyle.Render("q") + helpStyle.Render("/") + keyStyle.Render("Ctrl+C") + helpStyle.Render(" quit")

	return help
}

func Run(cfg *config.Config) error {
	model, err := NewModel(cfg)
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
