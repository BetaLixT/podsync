package tui

import (
	"fmt"
	"time"

	"github.com/BetaLixT/podsync"
	"github.com/BetaLixT/podsync/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type ipodCheckMsg struct {
	connected bool
	freeSpace uint64
}

type syncResult struct {
	synced int
	err    error
}

type markCompleteResult struct {
	err error
}

type DeviceComponent struct {
	width             int
	height            int
	device            podsync.Device
	source            podsync.PodcastSource
	db                podsync.PodcastDevice
	config            *config.Config
	episodes          *episodeList
	ipodConnected     bool
	freeSpace         uint64
	syncing           bool
	getMarkedEpisodes func() map[string][]podsync.SourceEpisode
	onStatusUpdate    func(StatusData)
}

func NewDeviceComponent(
	cfg *config.Config,
	device podsync.Device,
	source podsync.PodcastSource,
	db podsync.PodcastDevice,
	getMarkedEpisodes func() map[string][]podsync.SourceEpisode,
	onStatusUpdate func(StatusData),
) *DeviceComponent {
	return &DeviceComponent{
		0,                 // width
		0,                 // height
		device,            // device
		source,            // source
		db,                // db
		cfg,               // config
		newEpisodeList(),  // episodes
		false,             // ipodConnected
		0,                 // freeSpace
		false,             // syncing
		getMarkedEpisodes, // getMarkedEpisodes
		onStatusUpdate,    // onStatusUpdate
	}
}

func (d *DeviceComponent) Init() tea.Cmd {
	return nil
}

func (d *DeviceComponent) BatchJobs() []tea.Cmd {
	return []tea.Cmd{d.checkIPod()}
}

func (d *DeviceComponent) KeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"s"},
			Description: "sync",
			Condition:   func() bool { return d.ipodConnected && !d.syncing },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				d.syncing = true

				marked := d.getMarkedEpisodes()
				if len(marked) > 0 {
					episodes := []podsync.SourceEpisode{}
					for _, eps := range marked {
						episodes = append(episodes, eps...)
					}
					return d.syncMarkedEpisodes(episodes)
				}
				return d.syncEpisodes()
			},
		},
		{
			Keys:        []string{"m", "enter"},
			Description: "mark complete",
			Condition: func() bool {
				return d.ipodConnected && !d.syncing && d.episodes.Selected() != nil
			},
			Handle: func(km tea.KeyMsg) tea.Cmd {
				selected := d.episodes.Selected()
				if selected == nil {
					return nil
				}
				return d.markComplete(selected)
			},
		},
		{
			Keys:        []string{"j", "down"},
			Description: "navigate",
			Condition:   func() bool { return d.ipodConnected },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				d.episodes.MoveDown()
				return nil
			},
		},
		{
			Keys:        []string{"k", "up"},
			Description: "navigate",
			Condition:   func() bool { return d.ipodConnected },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				d.episodes.MoveUp()
				return nil
			},
		},
	}
}

func (d *DeviceComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.episodes.SetHeight(msg.Height - 10)
		return d, nil

	case ipodCheckMsg:
		wasConnected := d.ipodConnected
		d.ipodConnected = msg.connected
		d.freeSpace = msg.freeSpace

		var cmds []tea.Cmd
		cmds = append(cmds, d.checkIPod())

		if msg.connected && !wasConnected {
			cmds = append(cmds, d.loadEpisodes())
		} else if !msg.connected && wasConnected {
			d.episodes.SetEpisodes(nil)
		}

		d.pushStatus()
		return d, tea.Batch(cmds...)

	case syncResult:
		d.syncing = false
		d.pushStatus()
		if msg.err != nil {
			return d, nil
		}
		return d, d.loadEpisodes()

	case markCompleteResult:
		d.pushStatus()
		return d, d.loadEpisodes()

	case []podsync.Episode:
		d.episodes.SetEpisodes(msg)
		d.pushStatus()
		return d, nil
	}

	return d, nil
}

func (d *DeviceComponent) View() string {
	if !d.ipodConnected {
		width := d.width
		if width == 0 {
			width = 60
		}
		content := "\n\n" + subtitleStyle.Render("    Waiting for iPod to be connected...") + "\n\n"
		content += subtitleStyle.Render("    Mount path: "+d.config.IPodMount) + "\n\n"
		return boxStyle.Width(width - 2).Render(content)
	}

	width := d.width
	if width == 0 {
		width = 60
	}
	content := d.episodes.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (d *DeviceComponent) pushStatus() {
	if d.onStatusUpdate == nil {
		return
	}
	d.onStatusUpdate(StatusData{
		DeviceConnected: d.ipodConnected,
		DevicePath:      d.config.IPodMount,
		FreeSpace:       d.freeSpace,
		FreeSpaceStr:    d.device.FormatBytes(d.freeSpace),
		EpisodeCount:    d.episodes.Count(),
	})
}

func (d *DeviceComponent) checkIPod() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		connected := d.device.IsConnected()
		var freeSpace uint64
		if connected {
			freeSpace, _ = d.device.GetFreeSpace()
		}
		return ipodCheckMsg{connected: connected, freeSpace: freeSpace}
	})
}

func (d *DeviceComponent) loadEpisodes() tea.Cmd {
	return func() tea.Msg {
		episodes, err := d.db.GetAllEpisodes()
		if err != nil {
			return []podsync.Episode{}
		}

		var valid []podsync.Episode
		for _, ep := range episodes {
			if d.device.FileExists(ep.IPodPath) {
				valid = append(valid, ep)
			} else {
				d.db.RemoveEpisode(ep.ID)
			}
		}

		return valid
	}
}

func (d *DeviceComponent) syncMarkedEpisodes(episodes []podsync.SourceEpisode) tea.Cmd {
	return func() tea.Msg {
		var synced int
		for _, ep := range episodes {
			alreadySynced, err := d.db.IsEpisodeSynced(ep.ID)
			if err != nil {
				continue
			}
			if alreadySynced {
				continue
			}

			srcPath := d.source.GetFullPath(ep)
			destFilename := formatEpisodeFilename(ep)
			destPath, err := d.device.CopyFile(srcPath, ep.PodcastTitle, destFilename)
			if err != nil {
				continue
			}

			dbEpisode := podsync.Episode{
				SourcePodcastId: ep.ID,
				PodcastName:     ep.PodcastTitle,
				EpisodeTitle:    ep.Title,
				Filename:        destFilename,
				IPodPath:        destPath,
				Duration:        ep.TotalTime,
			}

			if err := d.db.AddEpisode(dbEpisode); err != nil {
				continue
			}

			synced++
		}

		return syncResult{synced: synced}
	}
}

func (d *DeviceComponent) syncEpisodes() tea.Cmd {
	return func() tea.Msg {
		episodes, err := d.source.GetLatestEpisodesPerPodcast(d.config.EpisodesPerPodcast)
		if err != nil {
			return syncResult{err: fmt.Errorf("failed to get episodes: %w", err)}
		}

		var synced int
		for _, ep := range episodes {
			alreadySynced, err := d.db.IsEpisodeSynced(ep.ID)
			if err != nil {
				continue
			}
			if alreadySynced {
				continue
			}

			srcPath := d.source.GetFullPath(ep)
			destFilename := formatEpisodeFilename(ep)
			destPath, err := d.device.CopyFile(srcPath, ep.PodcastTitle, destFilename)
			if err != nil {
				continue
			}

			dbEpisode := podsync.Episode{
				SourcePodcastId: ep.ID,
				PodcastName:     ep.PodcastTitle,
				EpisodeTitle:    ep.Title,
				Filename:        destFilename,
				IPodPath:        destPath,
				Duration:        ep.TotalTime,
			}

			if err := d.db.AddEpisode(dbEpisode); err != nil {
				continue
			}

			synced++
		}

		return syncResult{synced: synced}
	}
}

func (d *DeviceComponent) markComplete(ep *podsync.Episode) tea.Cmd {
	return func() tea.Msg {
		if err := d.device.RemoveFile(ep.IPodPath); err != nil {
			return markCompleteResult{err: err}
		}

		if err := d.source.MarkEpisodePlayed(ep.SourcePodcastId); err != nil {
			return markCompleteResult{err: fmt.Errorf("failed to mark as played: %w", err)}
		}

		if err := d.db.RemoveEpisode(ep.ID); err != nil {
			return markCompleteResult{err: err}
		}

		return markCompleteResult{}
	}
}
