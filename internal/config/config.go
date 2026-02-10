package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IPodMount          string `yaml:"ipod_mount"`
	GPodderHome        string `yaml:"gpodder_home"`
	EpisodesPerPodcast int    `yaml:"episodes_per_podcast"`
	PodcastFolder      string `yaml:"podcast_folder"`
}

func DefaultConfig() *Config {
	return &Config{
		IPodMount:          "/run/media/bazzite/IPOD",
		GPodderHome:        expandPath("~/gPodder"),
		EpisodesPerPodcast: 5,
		PodcastFolder:      "Podcasts",
	}
}

func ConfigDir() string {
	return expandPath("~/.podsync")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := ConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.GPodderHome = expandPath(cfg.GPodderHome)
	cfg.IPodMount = expandPath(cfg.IPodMount)

	return cfg, nil
}

func (c *Config) Save() error {
	configDir := ConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}

func (c *Config) GPodderDatabase() string {
	return filepath.Join(c.GPodderHome, "Database")
}

func (c *Config) GPodderDownloads() string {
	return filepath.Join(c.GPodderHome, "Downloads")
}

func (c *Config) PodsyncDatabase() string {
	return filepath.Join(ConfigDir(), "podsync.db")
}
