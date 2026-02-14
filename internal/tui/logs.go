package tui

import (
	"fmt"
	"strings"

	"github.com/BetaLixT/podsync"
	"github.com/charmbracelet/lipgloss"
)

type logList struct {
	lgr    podsync.Logger
	cursor int
	height int
	offset int
}

func newLogList(lgr podsync.Logger) *logList {
	return &logList{
		lgr,
		0, 0, 0,
	}
}

// func (s *logList) PushLog(l Log) {
// 	s.logs = append(s.logs, Log{})
// 	copy(s.logs[1:], s.logs)
// 	s.logs[0] = l
// }

func (s *logList) SetHeight(h int) {
	s.height = h
	s.updateOffset()
}

func (s *logList) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
		s.updateOffset()
	}
}

func (s *logList) ScrollToBottom() {
	s.cursor = len(s.lgr.GetAll()) - 1
	s.updateOffset()
}

func (s *logList) ScrollToTop() {
	s.cursor = 0
	s.updateOffset()
}

func (s *logList) View(width int) string {
	logs := s.lgr.GetAll()
	if len(logs) == 0 {
		return subtitleStyle.Render("  No logs.")
	}

	var b strings.Builder
	visibleEnd := min(s.offset+s.height, len(logs))

	for i := s.offset; i < visibleEnd; i++ {
		log := (logs)[len(logs)-1-i]
		line := s.formatLogLine(log, width, i == s.cursor)
		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (s *logList) MoveDown() {
	if s.cursor < len(s.lgr.GetAll())-1 {
		s.cursor++
		s.updateOffset()
	}
}

func (s *logList) updateOffset() {
	if s.cursor < s.offset {
		s.offset = s.cursor
	} else if s.cursor >= s.offset+s.height {
		s.offset = s.cursor - s.height + 1
	}
}

func (s *logList) formatLogLine(
	log podsync.Log,
	width int,
	selected bool,
) string {
	// prefix := "  "
	// if selected {
	// 	prefix = "> "
	// }

	level := "[" + string(log.Level) + "] "
	var logStyle lipgloss.Style
	switch log.Level {
	case podsync.LogErr:
		logStyle = logErrStyle
	case podsync.LogWrn:
		logStyle = logWrnStyle
	default:
		logStyle = logInfStyle
	}

	countWidth := len(level) + 1

	availableWidth := width - len(level) - countWidth
	if availableWidth < 20 {
		availableWidth = 20
	}

	msg := truncate(fmt.Sprintf(log.Message, log.Args...), availableWidth)

	if selected {
		return selectedStyle.Render(logStyle.Render(level + msg))
	}
	return normalStyle.Render(logStyle.Render(level + msg))
}
