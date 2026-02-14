package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BetaLixT/podsync"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type showEpisodeSortOrder int

const (
	sortByPublished showEpisodeSortOrder = iota
	sortByTitle
	sortByDuration
)

func (s showEpisodeSortOrder) String() string {
	switch s {
	case sortByPublished:
		return "Date"
	case sortByTitle:
		return "Title"
	case sortByDuration:
		return "Duration"
	default:
		return "Date"
	}
}

func (s showEpisodeSortOrder) Next() showEpisodeSortOrder {
	return (s + 1) % 3
}

type showEpisodeList struct {
	podcastTitle string
	allEpisodes  []podsync.SourceEpisode
	episodes     []podsync.SourceEpisode
	marked       map[string]bool
	cursor       int
	height       int
	offset       int
	sortOrder    showEpisodeSortOrder
	filterInput  textinput.Model
	filterActive bool
	filterText   string
}

func newShowEpisodeList(podcastTitle string) *showEpisodeList {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 100
	ti.Width = 30

	return &showEpisodeList{
		podcastTitle: podcastTitle,
		allEpisodes:  []podsync.SourceEpisode{},
		episodes:     []podsync.SourceEpisode{},
		marked:       make(map[string]bool),
		cursor:       0,
		height:       10,
		offset:       0,
		sortOrder:    sortByPublished,
		filterInput:  ti,
	}
}

func (s *showEpisodeList) SetEpisodes(episodes []podsync.SourceEpisode) {
	s.allEpisodes = episodes
	s.applyFilterAndSort()
}

func (s *showEpisodeList) SetHeight(h int) {
	s.height = h
	s.updateOffset()
}

func (s *showEpisodeList) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
		s.updateOffset()
	}
}

func (s *showEpisodeList) MoveDown() {
	if s.cursor < len(s.episodes)-1 {
		s.cursor++
		s.updateOffset()
	}
}

func (s *showEpisodeList) updateOffset() {
	if s.cursor < s.offset {
		s.offset = s.cursor
	} else if s.cursor >= s.offset+s.height {
		s.offset = s.cursor - s.height + 1
	}
}

func (s *showEpisodeList) Selected() *podsync.SourceEpisode {
	if len(s.episodes) == 0 || s.cursor >= len(s.episodes) {
		return nil
	}
	return &s.episodes[s.cursor]
}

func (s *showEpisodeList) Count() int {
	return len(s.episodes)
}

func (s *showEpisodeList) TotalCount() int {
	return len(s.allEpisodes)
}

func (s *showEpisodeList) ToggleMark() {
	sel := s.Selected()
	if sel == nil {
		return
	}
	if s.marked[sel.ID] {
		delete(s.marked, sel.ID)
	} else {
		s.marked[sel.ID] = true
	}
}

func (s *showEpisodeList) MarkedCount() int {
	return len(s.marked)
}

func (s *showEpisodeList) HasMarked() bool {
	return len(s.marked) != 0
}

func (s *showEpisodeList) GetMarked() []podsync.SourceEpisode {
	var result []podsync.SourceEpisode
	for _, ep := range s.allEpisodes {
		if s.marked[ep.ID] {
			result = append(result, ep)
		}
	}
	return result
}

func (s *showEpisodeList) RestoreMarked(episodes []podsync.SourceEpisode) {
	for _, ep := range episodes {
		s.marked[ep.ID] = true
	}
}

func (s *showEpisodeList) ClearMarked() {
	s.marked = make(map[string]bool)
}

func (s *showEpisodeList) CycleSort() {
	s.sortOrder = s.sortOrder.Next()
	s.applyFilterAndSort()
}

func (s *showEpisodeList) StartFilter() {
	s.filterInput.SetValue(s.filterText)
	s.filterInput.Focus()
	s.filterActive = true
}

func (s *showEpisodeList) CommitFilter() {
	s.filterText = s.filterInput.Value()
	s.filterInput.Blur()
	s.filterActive = false
	s.applyFilterAndSort()
}

func (s *showEpisodeList) CancelFilter() {
	s.filterInput.SetValue(s.filterText)
	s.filterInput.Blur()
	s.filterActive = false
}

func (s *showEpisodeList) ClearFilter() {
	s.filterText = ""
	s.filterInput.SetValue("")
	s.filterInput.Blur()
	s.filterActive = false
	s.applyFilterAndSort()
}

func (s *showEpisodeList) UpdateFilterInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)
	return cmd
}

func (s *showEpisodeList) applyFilterAndSort() {
	var filtered []podsync.SourceEpisode
	filterLower := strings.ToLower(s.filterText)

	for _, ep := range s.allEpisodes {
		if filterLower == "" || strings.Contains(strings.ToLower(ep.Title), filterLower) {
			filtered = append(filtered, ep)
		}
	}

	switch s.sortOrder {
	case sortByPublished:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Published > filtered[j].Published
		})
	case sortByTitle:
		sort.Slice(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].Title) < strings.ToLower(filtered[j].Title)
		})
	case sortByDuration:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].TotalTime > filtered[j].TotalTime
		})
	}

	s.episodes = filtered
	if s.cursor >= len(s.episodes) {
		s.cursor = max(0, len(s.episodes)-1)
	}
	s.updateOffset()
}

func (s *showEpisodeList) View(width int) string {
	var b strings.Builder

	// Header line: podcast title + sort + count + marked
	countStr := fmt.Sprintf("%d/%d episodes", s.Count(), s.TotalCount())
	sortStr := "Sort: " + s.sortOrder.String()
	header := podcastNameStyle.Render(s.podcastTitle) + "  " +
		subtitleStyle.Render(sortStr) + "  " +
		subtitleStyle.Render(countStr)
	if s.MarkedCount() > 0 {
		header += "  " + syncingStyle.Render(fmt.Sprintf("%d marked", s.MarkedCount()))
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Filter line
	if s.filterActive {
		b.WriteString("/ " + s.filterInput.View())
		b.WriteString("\n")
	} else if s.filterText != "" {
		b.WriteString(subtitleStyle.Render("Filter: " + s.filterText))
		b.WriteString("\n")
	}

	if len(s.episodes) == 0 {
		b.WriteString(subtitleStyle.Render("  No episodes found."))
		return b.String()
	}

	visibleEnd := min(s.offset+s.height, len(s.episodes))

	for i := s.offset; i < visibleEnd; i++ {
		ep := s.episodes[i]
		line := s.formatLine(ep, width, i == s.cursor)
		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (s *showEpisodeList) formatLine(ep podsync.SourceEpisode, width int, selected bool) string {
	duration := formatDuration(ep.TotalTime)
	durationWidth := len(duration) + 1

	check := "[ ] "
	if s.marked[ep.ID] {
		check = "[x] "
	}

	marker := "  "
	if ep.IsNew {
		marker = "* "
	}

	prefix := "  "
	if selected {
		prefix = "> "
	}

	availableWidth := width - len(prefix) - len(check) - len(marker) - durationWidth
	if availableWidth < 20 {
		availableWidth = 20
	}

	title := truncate(ep.Title, availableWidth)
	title = padRight(title, availableWidth)

	line := prefix + check + marker + title + " " + duration

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}
