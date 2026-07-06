package main

import (
	"context"
	"fmt"
	"os"

	"music-downloader/src/domain"
	"music-downloader/src/infra/config"
	"music-downloader/src/ytdl"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config init: %v\n", err)
		os.Exit(1)
	}

	svc := ytdl.New(cfg.OutputDir, cfg.MaxDownloadThreads)

	ctx := context.Background()

	query := "Tyla - Water"
	if len(os.Args) > 1 {
		query = os.Args[1]
	}

	fmt.Printf("Searching YouTube for: %s\n", query)
	results, err := svc.Search(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "no results\n")
		os.Exit(1)
	}

	r := results[0]
	fmt.Printf("Found: %s (%s)\n", r.Title, r.URL)

	song := domain.Song{
		Title:  r.Title,
		Artist: "Test",
		Album:  "Test",
	}

	path, err := svc.Download(ctx, r, song, func(p domain.DownloadProgress) {
		fmt.Printf("Progress: %s %d%%\n", p.Status, p.Progress)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "download: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Downloaded: %s\n", path)
	defer os.Remove(path)
}
