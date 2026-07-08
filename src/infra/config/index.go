package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type SpotifyConfig struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type Config struct {
	Spotify                SpotifyConfig `json:"spotify"`
	OutputDir              string        `json:"outputDir"`
	MaxConcurrentDownloads int           `json:"maxConcurrentDownloads"`
	MaxDownloadThreads     int           `json:"maxDownloadThreads"`
	Debug                  bool          `json:"debug"`
}

const DefaultOutputDir = "downloads"

func Init() (*Config, error) {
	c, err := Read()
	if err != nil {
		return nil, err
	}

	if c.OutputDir == "" {
		c.OutputDir = DefaultOutputDir
	}

	c.OutputDir = filepath.Clean(filepath.FromSlash(c.OutputDir))

	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return nil, err
	}

	validateConfig, err := Validate(c)
	if err != nil {
		return nil, err
	}

	return validateConfig, nil
}

func Read() (*Config, error) {
	fileBytes, err := os.ReadFile("config.json")

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := Write(&Config{})

			if err != nil {
				return nil, errors.New("Failed to create config file: " + err.Error())
			}

			return &Config{}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(fileBytes, &config); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w\nHint: Windows paths need double backslashes (e.g. \"D:\\\\downloads\") or forward slashes (\"D:/downloads\")", err)
	}

	return &config, nil
}

func Write(config *Config) error {
	indent, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("config.json", indent, 0644); err != nil {
		return err
	}

	return nil
}
