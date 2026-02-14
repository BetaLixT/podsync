package gpodder

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/BetaLixT/podsync"
	_ "github.com/mattn/go-sqlite3"
)

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

func (c *Client) DatabaseExists() bool {
	_, err := os.Stat(c.dbPath)
	return err == nil
}

func (c *Client) DatabasePath() string {
	return c.dbPath
}

func (c *Client) openDB() (*sql.DB, error) {
	return sql.Open("sqlite3", c.dbPath+"?mode=ro")
}

func (c *Client) GetPodcasts() ([]podsync.SourcePodcast, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT
			p.id,
			p.title,
			p.url,
			COUNT(e.id) as episode_count
		FROM podcast p
		LEFT JOIN episode e ON e.podcast_id = p.id
			AND e.state = 1 AND e.is_new = 1 AND e.download_filename IS NOT NULL
		GROUP BY p.id
		ORDER BY p.title
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var podcasts []podsync.SourcePodcast
	for rows.Next() {
		var p podsync.SourcePodcast
		if err := rows.Scan(&p.ID, &p.Title, &p.URL, &p.EpisodeCount); err != nil {
			return nil, err
		}
		podcasts = append(podcasts, p)
	}

	return podcasts, rows.Err()
}

func (c *Client) GetEpisodesForPodcast(podcastID string) ([]podsync.SourceEpisode, error) {
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
			e.total_time,
			e.published,
			e.is_new,
			ROW_NUMBER() OVER (ORDER BY e.published ASC) as episode_number
		FROM episode e
		JOIN podcast p ON e.podcast_id = p.id
		WHERE e.podcast_id = ? AND e.state = 1 AND e.download_filename IS NOT NULL
		ORDER BY e.published DESC
	`

	rows, err := db.Query(query, podcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []podsync.SourceEpisode
	for rows.Next() {
		var ep podsync.SourceEpisode
		if err := rows.Scan(&ep.ID, &ep.PodcastTitle, &ep.Title, &ep.DownloadFilename, &ep.TotalTime, &ep.Published, &ep.IsNew, &ep.EpisodeNumber); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	return episodes, rows.Err()
}

func (c *Client) GetDownloadedUnplayedEpisodes() ([]podsync.SourceEpisode, error) {
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
			e.total_time,
			e.published,
			ROW_NUMBER() OVER (PARTITION BY e.podcast_id ORDER BY e.published ASC) as episode_number
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

	var episodes []podsync.SourceEpisode
	for rows.Next() {
		var ep podsync.SourceEpisode
		if err := rows.Scan(&ep.ID, &ep.PodcastTitle, &ep.Title, &ep.DownloadFilename, &ep.TotalTime, &ep.Published, &ep.EpisodeNumber); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	return episodes, rows.Err()
}

func (c *Client) GetLatestEpisodesPerPodcast(limit int) ([]podsync.SourceEpisode, error) {
	episodes, err := c.GetDownloadedUnplayedEpisodes()
	if err != nil {
		return nil, err
	}

	podcastCounts := make(map[string]int)
	var filtered []podsync.SourceEpisode

	for _, ep := range episodes {
		if podcastCounts[ep.PodcastTitle] < limit {
			filtered = append(filtered, ep)
			podcastCounts[ep.PodcastTitle]++
		}
	}

	return filtered, nil
}

func (c *Client) MarkEpisodePlayed(episodeID string) error {
	db, err := sql.Open("sqlite3", c.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("UPDATE episode SET is_new = 0 WHERE id = ?", episodeID)
	return err
}

func (c *Client) GetFullPath(ep podsync.SourceEpisode) string {
	return filepath.Join(c.downloadsDir, ep.PodcastTitle, ep.DownloadFilename)
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
