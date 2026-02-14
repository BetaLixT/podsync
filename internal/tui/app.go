package tui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/BetaLixT/podsync"
	"github.com/BetaLixT/podsync/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func formatEpisodeFilename(ep podsync.SourceEpisode) string {
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

func Run(
	cfg *config.Config,
	podcast podsync.PodcastSource,
	device podsync.Device,
	localDB podsync.PodcastDevice,
	lgr podsync.Logger,
) error {
	app := NewAppComponent(cfg, podcast, device, localDB, nil, lgr)
	options := NewOptionsComponent(cfg)

	children := map[RootContentComponent]Component{
		AppRootView:     app,
		OptionsRootView: options,
	}
	root := NewRoot(cfg, IPodRockBoxDevice, GPodderSource, children, lgr)

	// Wire status callback now that root exists
	app.onStatusUpdate = func(data StatusData) {
		root.statusData = data
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		return err
	}
	defer f.Close()

	p := tea.NewProgram(root, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
