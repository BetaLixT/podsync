package tui

import (
	"github.com/BetaLixT/podsync/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type OptionsComponent struct {
	width   int
	height  int
	options *optionsCtrl
	config  *config.Config
}

func NewOptionsComponent(cfg *config.Config) *OptionsComponent {
	return &OptionsComponent{
		0,                    // width
		0,                    // height
		newOptionsCtrl(*cfg), // options
		cfg,                  // config
	}
}

func (o *OptionsComponent) Init() tea.Cmd {
	return nil
}

func (o *OptionsComponent) BatchJobs() []tea.Cmd {
	return nil
}

func (o *OptionsComponent) KeyMaps() []KeyMap {
	return []KeyMap{
		{
			Keys:        []string{"i"},
			Description: "edit",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				o.options.InsertMode()
				return func() tea.Msg {
					return StartInsertCmd{
						Handle: func(k tea.KeyMsg) tea.Cmd {
							_, cmd := o.options.InsertModeUpdate(k)
							return cmd
						},
					}
				}
			},
		},
		{
			Keys:        []string{"s"},
			Description: "save",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				c, err := getConfig[*config.Config](o.options)
				if err != nil {
					return nil
				}
				if err := c.Save(); err != nil {
					return nil
				}
				return tea.Quit
			},
		},
		{
			Keys:        []string{"j", "down"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				o.options.MoveDown()
				return nil
			},
		},
		{
			Keys:        []string{"k", "up"},
			Description: "navigate",
			Condition:   func() bool { return true },
			Handle: func(km tea.KeyMsg) tea.Cmd {
				o.options.MoveUp()
				return nil
			},
		},
	}
}

func (o *OptionsComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height
		o.options.SetHeight(msg.Height - 10)
		return o, nil
	}
	return o, nil
}

func (o *OptionsComponent) View() string {
	width := o.width
	if width == 0 {
		width = 60
	}
	content := o.options.View(width - 4)
	return boxStyle.Width(width - 2).Render(content)
}

func (o *OptionsComponent) CommitInsert() {
	o.options.InsertModeSave()
}

func (o *OptionsComponent) CancelInsert() {
	o.options.InsertModeCancel()
}
