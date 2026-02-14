package podsync

import "time"

type Episode struct {
	ID              int64
	SourcePodcastId string
	PodcastName     string
	EpisodeTitle    string
	Filename        string
	IPodPath        string
	Duration        int
	SyncedAt        time.Time
}

type PodcastDevice interface {
	Init() error
	AddEpisode(ep Episode) error
	RemoveEpisode(id int64) error
	RemoveBySourceId(sourceId string)
	GetAllEpisodes() ([]Episode, error)
	GetEpisodeCount() (int, error)
	IsEpisodeSynced(sourceId string) (bool, error)
}
