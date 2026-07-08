package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"spotscoop/src/app"
	"spotscoop/src/app/ui"
	"spotscoop/src/auth"
	"spotscoop/src/db"
	"spotscoop/src/domain"
	"spotscoop/src/infra/config"
	"spotscoop/src/infra/logs"
	"spotscoop/src/infra/tui"
	spotifyservice "spotscoop/src/spotify"
	"spotscoop/src/ytdl"

	"github.com/zmb3/spotify/v2"
)

func main() {
	addr := "127.0.0.1:50811"

	tui.Run(addr, func(cfg *config.Config) (*http.Server, func(), error) {
		logs.SetDebug(cfg.Debug)

		srv, handler, cleanup, err := setupServer(cfg, addr)
		if err != nil {
			return nil, nil, err
		}

		return srv, func() { handler.Shutdown(); cleanup() }, nil
	})
}

func setupServer(cfg *config.Config, addr string) (*http.Server, *app.Handler, func(), error) {
	authServer := auth.NewSpotifyAuthServer(
		cfg.Spotify.ClientId,
		cfg.Spotify.ClientSecret,
		fmt.Sprintf("http://%s/api/auth", addr),
	)

	database, err := db.New("music_downloader.db")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("database: %w", err)
	}

	svc := spotifyservice.New()
	svc.SetDB(database)

	dl := ytdl.New(cfg.OutputDir, cfg.MaxDownloadThreads)

	handler := app.NewHandler(authServer, svc, dl, cfg)

	authServer.OnAuth = func(client *spotify.Client, httpClient *http.Client, user domain.User) {
		svc.SetClient(client, httpClient)
		handler.SetUser(&user)
	}

	distFS, err := ui.DistFS()
	if err != nil {
		database.Close()
		return nil, nil, nil, fmt.Errorf("frontend: %w", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux := http.NewServeMux()
	authServer.RegisterRoutes(mux)
	handler.RegisterRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, err := distFS.Open(path); err != nil {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	go func() {
		client, httpClient, user, err := authServer.LoadToken(context.Background())
		if err != nil {
			logs.Info("No saved session, waiting for login")
			return
		}

		svc.SetClient(client, httpClient)

		if user != nil {
			handler.SetUser(user)
		}
	}()

	return &http.Server{Addr: addr, Handler: mux}, handler, func() { database.Close() }, nil
}
