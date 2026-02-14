package tui

import (
	"strings"

	"github.com/BetaLixT/podsync"
)

type showList struct {
	shows  []podsync.Podcast
	cursor int
	height int
	offset int
}

func newShowList() *showList {
	return &showList{
		shows:  []podsync.Podcast{},
		cursor: 0,
		height: 10,
		offset: 0,
	}
}

func (s *showList) SetShows(shows []podsync.Podcast) {
	s.shows = shows
	if s.cursor >= len(s.shows) {
		s.cursor = max(0, len(s.shows)-1)
	}
	s.updateOffset()
}

func (s *showList) SetHeight(h int) {
	s.height = h
	s.updateOffset()
}

func (s *showList) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
		s.updateOffset()
	}
}

func (s *showList) MoveDown() {
	if s.cursor < len(s.shows)-1 {
		s.cursor++
		s.updateOffset()
	}
}

func (s *showList) updateOffset() {
	if s.cursor < s.offset {
		s.offset = s.cursor
	} else if s.cursor >= s.offset+s.height {
		s.offset = s.cursor - s.height + 1
	}
}

func (s *showList) Selected() *podsync.Podcast {
	if len(s.shows) == 0 || s.cursor >= len(s.shows) {
		return nil
	}
	return &s.shows[s.cursor]
}

func (s *showList) Count() int {
	return len(s.shows)
}

func (s *showList) View(width int) string {
	if len(s.shows) == 0 {
		return subtitleStyle.Render("  No shows found.")
	}

	var b strings.Builder
	visibleEnd := min(s.offset+s.height, len(s.shows))

	for i := s.offset; i < visibleEnd; i++ {
		show := s.shows[i]
		line := s.formatShowLine(show, width, i == s.cursor)
		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (s *showList) formatShowLine(show podsync.Podcast, width int, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	episodeCount := intToStr(show.EpisodeCount) + " ep"
	if show.EpisodeCount != 1 {
		episodeCount += "s"
	}
	countWidth := len(episodeCount) + 1

	availableWidth := width - len(prefix) - countWidth
	if availableWidth < 20 {
		availableWidth = 20
	}

	title := truncate(show.Title, availableWidth)
	title = padRight(title, availableWidth)

	line := prefix + title + " " + episodeCount

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}
