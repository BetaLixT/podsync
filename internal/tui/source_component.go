package tui

import (
	"github.com/BetaLixT/podsync"
	tea "github.com/charmbracelet/bubbletea"
)

type SourceView int

const (
	ShowListView SourceView = iota
	EpisodeListView
)

type showEpisodesMsg []podsync.SourceEpisode

type SourceComponent struct {
	width               int
	height              int
	currentView         SourceView
	source              podsync.PodcastSource
	sourceFound         bool
	shows               *showList
	showEpisodes        *showEpisodeList
	onMarkedUpdate      func(podcastID string, episodes []podsync.SourceEpisode)
	getMarkedForPodcast func(podcastID string) []podsync.SourceEpisode
	onStatusUpdate      func(StatusData)
}

func NewSourceComponent(
	source podsync.PodcastSource,
	onMarkedUpdate func(podcastID string, episodes []podsync.SourceEpisode),
	getMarkedForPodcast func(podcastID string) []podsync.SourceEpisode,
	onStatusUpdate func(StatusData),
) *SourceComponent {
	sc := &SourceComponent{
		0,                       // width
		0,                       // height
		ShowListView,            // currentView
		source,                  // source
		source.DatabaseExists(), // sourceFound
		newShowList(),           // shows
		nil,                     // showEpisodes
		onMarkedUpdate,          // onMarkedUpdate
		getMarkedForPodcast,     // getMarkedForPodcast
		onStatusUpdate,          // onStatusUpdate
	}
	sc.pushStatus()
	return sc
}

func (s *SourceComponent) Init() tea.Cmd {
	return nil
}

func (s *SourceComponent) BatchJobs() []tea.Cmd {
	return []tea.Cmd{s.loadShows()}
}

func (s *SourceComponent) KeyMaps() []KeyMap {
	switch s.currentView {
	case ShowListView:
		return s.showListKeyMaps()
	case EpisodeListView:
		return s.episodeListKeyMaps()
	}
	return nil
}

func (s *SourceComponent) showListKeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"enter"},
			Description: "open",
			Condition:   func() bool { return s.shows.Selected() != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				selected := s.shows.Selected()
				if selected == nil {
					return nil
				}
				s.showEpisodes = newShowEpisodeList(selected.Title)
				s.showEpisodes.SetHeight(s.height - 12)
				if s.getMarkedForPodcast != nil {
					s.showEpisodes.RestoreMarked(s.getMarkedForPodcast(selected.ID))
				}
				s.currentView = EpisodeListView
				return s.loadShowEpisodes(selected.ID)
			},
		},
		{
			Keys:        []string{"j", "down"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.shows.MoveDown()
				return nil
			},
		},
		{
			Keys:        []string{"k", "up"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.shows.MoveUp()
				return nil
			},
		},
	}
}

func (s *SourceComponent) episodeListKeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"space"},
			Description: "mark",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.ToggleMark()
				return nil
			},
		},
		{
			Keys:        []string{"s"},
			Description: "sort",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.CycleSort()
				return nil
			},
		},
		{
			Keys:        []string{"/"},
			Description: "filter",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.StartFilter()
				return func() tea.Msg {
					return StartInsertCmd{
						Handle: func(k tea.KeyMsg) tea.Cmd {
							return s.showEpisodes.UpdateFilterInput(k)
						},
					}
				}
			},
		},
		{
			Keys:        []string{"c"},
			Description: "clear",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.ClearFilter()
				return nil
			},
		},
		{
			Keys:        []string{"esc"},
			Description: "back",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				if s.showEpisodes != nil {
					selected := s.shows.Selected()
					if selected != nil && s.onMarkedUpdate != nil {
						markedEpisodes := s.showEpisodes.GetMarked()
						s.onMarkedUpdate(selected.ID, markedEpisodes)
					}
					s.showEpisodes = nil
				}
				s.currentView = ShowListView
				return nil
			},
		},
		{
			Keys:        []string{"j", "down"},
			Description: "navigate",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.MoveDown()
				return nil
			},
		},
		{
			Keys:        []string{"k", "up"},
			Description: "navigate",
			Condition:   func() bool { return s.showEpisodes != nil },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.showEpisodes.MoveUp()
				return nil
			},
		},
	}
}

func (s *SourceComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.shows.SetHeight(msg.Height - 10)
		if s.showEpisodes != nil {
			s.showEpisodes.SetHeight(msg.Height - 12)
		}
		return s, nil

	case []podsync.SourcePodcast:
		s.shows.SetShows(msg)
		return s, nil

	case showEpisodesMsg:
		if s.showEpisodes != nil {
			s.showEpisodes.SetEpisodes([]podsync.SourceEpisode(msg))
		}
		return s, nil
	}

	return s, nil
}

func (s *SourceComponent) View() string {
	width := s.width
	if width == 0 {
		width = 60
	}

	var content string
	switch s.currentView {
	case ShowListView:
		content = s.shows.View(width - 4)
	case EpisodeListView:
		if s.showEpisodes != nil {
			content = s.showEpisodes.View(width - 4)
		}
	}

	return boxStyle.Width(width - 2).Render(content)
}

func (s *SourceComponent) CommitFilter() {
	if s.showEpisodes != nil {
		s.showEpisodes.CommitFilter()
	}
}

func (s *SourceComponent) CancelFilter() {
	if s.showEpisodes != nil {
		s.showEpisodes.CancelFilter()
	}
}

func (s *SourceComponent) pushStatus() {
	if s.onStatusUpdate == nil {
		return
	}
	s.onStatusUpdate(StatusData{
		SourceFound: s.sourceFound,
		SourcePath:  s.source.DatabasePath(),
	})
}

func (s *SourceComponent) loadShows() tea.Cmd {
	return func() tea.Msg {
		podcasts, err := s.source.GetPodcasts()
		if err != nil {
			return []podsync.SourcePodcast{}
		}
		return podcasts
	}
}

func (s *SourceComponent) loadShowEpisodes(podcastID string) tea.Cmd {
	return func() tea.Msg {
		episodes, err := s.source.GetEpisodesForPodcast(podcastID)
		if err != nil {
			return showEpisodesMsg([]podsync.SourceEpisode{})
		}
		return showEpisodesMsg(episodes)
	}
}
