package podsync

type Podcast struct {
	ID           int64
	Title        string
	URL          string
	EpisodeCount int
}

type Episode struct {
	ID               int64
	PodcastTitle     string
	Title            string
	DownloadFilename string
	TotalTime        int
	Published        int64
	IsNew            bool
}

type PodcastSource interface {
	DatabaseExists() bool
	DatabasePath() string
	GetPodcasts() ([]Podcast, error)
	GetEpisodesForPodcast(podcastID int64) ([]Episode, error)
	GetDownloadedUnplayedEpisodes() ([]Episode, error)
	GetLatestEpisodesPerPodcast(limit int) ([]Episode, error)
	MarkEpisodePlayed(episodeID int64) error
	GetFullPath(downloadFilename string) string
}
