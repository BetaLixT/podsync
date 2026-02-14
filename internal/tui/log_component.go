package tui

import (
	"github.com/BetaLixT/podsync"
	tea "github.com/charmbracelet/bubbletea"
)

type LogComponent struct {
	width, height int
	logs          *logList
}

func NewLogComponent(lgr podsync.Logger) *LogComponent {
	return &LogComponent{
		0, 0,
		newLogList(lgr),
	}
}

func (s *LogComponent) Init() tea.Cmd {
	return nil
}

func (s *LogComponent) BatchJobs() []tea.Cmd {
	return []tea.Cmd{}
}

func (s *LogComponent) KeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"j", "down"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.logs.MoveDown()
				return nil
			},
		},
		{
			Keys:        []string{"k", "up"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.logs.MoveUp()
				return nil
			},
		},
		{
			Keys:        []string{"G"},
			Description: "scroll to bottom",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.logs.ScrollToBottom()
				return nil
			},
		},

		{
			Keys:        []string{"g"},
			Description: "scroll to top",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				s.logs.ScrollToTop()
				return nil
			},
		},
	}
}

func (s *LogComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.logs.SetHeight(msg.Height - 10)

		return s, nil
	}

	return s, nil
}

func (s *LogComponent) View() string {
	width := s.width
	if width == 0 {
		width = 60
	}

	var content string
	content = s.logs.View(width - 4)

	return boxStyle.Width(width - 2).Render(content)
}
