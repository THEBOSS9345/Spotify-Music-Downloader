package config

import "errors"

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

	_ = Write(config)

	return config, nil
}
