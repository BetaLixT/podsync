package tui

import (
	"strings"

	"github.com/dcruza/podsync/internal/db"
)

type episodeList struct {
	episodes []db.Episode
	cursor   int
	height   int
	offset   int
}

func newEpisodeList() *episodeList {
	return &episodeList{
		episodes: []db.Episode{},
		cursor:   0,
		height:   10,
		offset:   0,
	}
}

func (e *episodeList) SetEpisodes(episodes []db.Episode) {
	e.episodes = episodes
	if e.cursor >= len(e.episodes) {
		e.cursor = max(0, len(e.episodes)-1)
	}
	e.updateOffset()
}

func (e *episodeList) SetHeight(h int) {
	e.height = h
	e.updateOffset()
}

func (e *episodeList) MoveUp() {
	if e.cursor > 0 {
		e.cursor--
		e.updateOffset()
	}
}

func (e *episodeList) MoveDown() {
	if e.cursor < len(e.episodes)-1 {
		e.cursor++
		e.updateOffset()
	}
}

func (e *episodeList) updateOffset() {
	if e.cursor < e.offset {
		e.offset = e.cursor
	} else if e.cursor >= e.offset+e.height {
		e.offset = e.cursor - e.height + 1
	}
}

func (e *episodeList) Selected() *db.Episode {
	if len(e.episodes) == 0 || e.cursor >= len(e.episodes) {
		return nil
	}
	return &e.episodes[e.cursor]
}

func (e *episodeList) RemoveSelected() {
	if len(e.episodes) == 0 {
		return
	}
	e.episodes = append(e.episodes[:e.cursor], e.episodes[e.cursor+1:]...)
	if e.cursor >= len(e.episodes) && e.cursor > 0 {
		e.cursor--
	}
	e.updateOffset()
}

func (e *episodeList) Count() int {
	return len(e.episodes)
}

func (e *episodeList) View(width int) string {
	if len(e.episodes) == 0 {
		return subtitleStyle.Render("  No episodes on iPod. Press 's' to sync.")
	}

	var b strings.Builder
	visibleEnd := min(e.offset+e.height, len(e.episodes))

	for i := e.offset; i < visibleEnd; i++ {
		ep := e.episodes[i]
		line := e.formatEpisodeLine(ep, width, i == e.cursor)
		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (e *episodeList) formatEpisodeLine(ep db.Episode, width int, selected bool) string {
	duration := formatDuration(ep.Duration)
	durationWidth := len(duration) + 1

	prefix := "  "
	if selected {
		prefix = "> "
	}

	availableWidth := width - len(prefix) - durationWidth - 3
	if availableWidth < 20 {
		availableWidth = 20
	}

	podcastWidth := min(20, availableWidth/3)
	titleWidth := availableWidth - podcastWidth - 3

	podcast := truncate(ep.PodcastName, podcastWidth)
	title := truncate(ep.EpisodeTitle, titleWidth)

	podcast = padRight(podcast, podcastWidth)
	title = padRight(title, titleWidth)

	line := prefix + podcast + " - " + title + " " + duration

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

func formatDuration(totalSeconds int) string {
	if totalSeconds <= 0 {
		return "--:--"
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return padInt(hours) + ":" + padInt(minutes) + ":" + padInt(seconds)
	}
	return padInt(minutes) + ":" + padInt(seconds)
}

func padInt(n int) string {
	if n < 10 {
		return "0" + intToStr(n)
	}
	return intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
