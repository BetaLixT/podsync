package gpodder

import (
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Episode struct {
	ID               int64
	PodcastTitle     string
	Title            string
	DownloadFilename string
	TotalTime        int
}

type Client struct {
	dbPath       string
	downloadsDir string
}

func New(gpodderHome string) *Client {
	return &Client{
		dbPath:       filepath.Join(gpodderHome, "Database"),
		downloadsDir: filepath.Join(gpodderHome, "Downloads"),
	}
}

func (c *Client) openDB() (*sql.DB, error) {
	return sql.Open("sqlite3", c.dbPath+"?mode=ro")
}

func (c *Client) GetDownloadedUnplayedEpisodes() ([]Episode, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			e.id,
			p.title as podcast_title,
			e.title,
			e.download_filename,
			e.total_time
		FROM episode e
		JOIN podcast p ON e.podcast_id = p.id
		WHERE e.state = 1 AND e.is_new = 1 AND e.download_filename IS NOT NULL
		ORDER BY p.title, e.published DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var ep Episode
		if err := rows.Scan(&ep.ID, &ep.PodcastTitle, &ep.Title, &ep.DownloadFilename, &ep.TotalTime); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	return episodes, rows.Err()
}

func (c *Client) GetLatestEpisodesPerPodcast(limit int) ([]Episode, error) {
	episodes, err := c.GetDownloadedUnplayedEpisodes()
	if err != nil {
		return nil, err
	}

	podcastCounts := make(map[string]int)
	var filtered []Episode

	for _, ep := range episodes {
		if podcastCounts[ep.PodcastTitle] < limit {
			filtered = append(filtered, ep)
			podcastCounts[ep.PodcastTitle]++
		}
	}

	return filtered, nil
}

func (c *Client) MarkEpisodePlayed(episodeID int64) error {
	db, err := sql.Open("sqlite3", c.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("UPDATE episode SET is_new = 0 WHERE id = ?", episodeID)
	return err
}

func (c *Client) GetFullPath(downloadFilename string) string {
	return filepath.Join(c.downloadsDir, downloadFilename)
}

func (c *Client) FormatDuration(totalSeconds int) string {
	if totalSeconds <= 0 {
		return "--:--"
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return formatTime(hours, minutes, seconds, true)
	}
	return formatTime(0, minutes, seconds, false)
}

func formatTime(h, m, s int, showHours bool) string {
	if showHours {
		return padInt(h) + ":" + padInt(m) + ":" + padInt(s)
	}
	return padInt(m) + ":" + padInt(s)
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
