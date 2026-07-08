package config

import (
	"errors"
	"path/filepath"
)

func Validate(config *Config) (*Config, error) {
	if config.Spotify.ClientId == "" {
		return nil, errors.New("spotify client id is required")
	}

	if config.Spotify.ClientSecret == "" {
		return nil, errors.New("spotify client secret is required")
	}

	if config.OutputDir == "" {
		config.OutputDir = "downloads"
	}

	config.OutputDir = filepath.Clean(filepath.FromSlash(config.OutputDir))

	if config.MaxConcurrentDownloads <= 0 {
		config.MaxConcurrentDownloads = 3
	}

	if config.MaxDownloadThreads <= 0 {
		config.MaxDownloadThreads = 15
	}

	_ = Write(config)

	return config, nil
}
