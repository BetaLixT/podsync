package podsync

type SourcePodcast struct {
	ID           string
	Title        string
	URL          string
	EpisodeCount int
}

type SourceEpisode struct {
	ID               string
	PodcastTitle     string
	Title            string
	DownloadFilename string
	TotalTime        int
	Published        int64
	IsNew            bool
	EpisodeNumber    int
}

type PodcastSource interface {
	DatabaseExists() bool
	DatabasePath() string
	GetPodcasts() ([]SourcePodcast, error)
	GetEpisodesForPodcast(podcastID string) ([]SourceEpisode, error)
	GetDownloadedUnplayedEpisodes() ([]SourceEpisode, error)
	GetLatestEpisodesPerPodcast(limit int) ([]SourceEpisode, error)
	MarkEpisodePlayed(episodeID string) error
	GetFullPath(SourceEpisode) string
}
