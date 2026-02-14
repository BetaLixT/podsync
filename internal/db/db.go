package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/BetaLixT/podsync"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	path string
}

func New(dbPath string) *DB {
	return &DB{path: dbPath}
}

func (d *DB) Init() error {
	dir := filepath.Dir(d.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
		CREATE TABLE IF NOT EXISTS episodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_episode_id TEXT NOT NULL,
			podcast_name TEXT NOT NULL,
			episode_title TEXT NOT NULL,
			filename TEXT NOT NULL,
			ipod_path TEXT NOT NULL,
			duration INTEGER DEFAULT 0,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_episode_id)
		);
		CREATE INDEX IF NOT EXISTS idx_source_episode_id ON episodes(source_episode_id);
	`
	_, err = db.Exec(schema)
	return err
}

func (d *DB) AddEpisode(ep podsync.Episode) error {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		INSERT OR REPLACE INTO episodes
		(source_episode_id, podcast_name, episode_title, filename, ipod_path, duration, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ep.SourcePodcastId, ep.PodcastName, ep.EpisodeTitle, ep.Filename, ep.IPodPath, ep.Duration, time.Now(),
	)
	return err
}

func (d *DB) RemoveEpisode(id int64) error {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM episodes WHERE id = ?", id)
	return err
}

func (d *DB) RemoveBySourceId(sourceId string) error {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM episodes WHERE source_episode_id = ?", sourceId)
	return err
}

func (d *DB) GetAllEpisodes() ([]podsync.Episode, error) {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, source_episode_id, podcast_name, episode_title, filename, ipod_path, duration, synced_at
		FROM episodes
		ORDER BY podcast_name, synced_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []podsync.Episode
	for rows.Next() {
		var ep podsync.Episode
		if err := rows.Scan(&ep.ID, &ep.SourcePodcastId, &ep.PodcastName, &ep.EpisodeTitle, &ep.Filename, &ep.IPodPath, &ep.Duration, &ep.SyncedAt); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}

	return episodes, rows.Err()
}

func (d *DB) GetEpisodeCount() (int, error) {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM episodes").Scan(&count)
	return count, err
}

func (d *DB) IsEpisodeSynced(sourceId string) (bool, error) {
	db, err := sql.Open("sqlite3", d.path)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM episodes WHERE source_episode_id = ?", sourceId).Scan(&count)
	return count > 0, err
}
