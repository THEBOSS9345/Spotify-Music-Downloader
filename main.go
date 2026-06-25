package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"music-downloader/src/app"
	"music-downloader/src/app/ui"
	"music-downloader/src/auth"
	"music-downloader/src/db"
	"music-downloader/src/domain"
	"music-downloader/src/infra/config"
	"music-downloader/src/infra/logs"
	spotifyservice "music-downloader/src/spotify"
	"music-downloader/src/ytdl"

	"github.com/zmb3/spotify/v2"
)

func main() {

	addr := "127.0.0.1:50811"

	fmt.Println()
	fmt.Println("  Music Downloader - THEBOSS9345")
	fmt.Println("  " + strings.Repeat("-", 54))
	fmt.Printf("  Server:  http://%s\n", addr)
	fmt.Println()
	fmt.Println("  Spotify Setup Required")
	fmt.Println("  Go to https://developer.spotify.com/dashboard")
	fmt.Println("  and add this Redirect URI:")
	fmt.Printf("    http://%s/api/auth\n", addr)

	cfg, err := config.Init()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Downloads folder: %s\n", cfg.OutputDir)
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println("  " + strings.Repeat("-", 54))
	fmt.Println()

	authServer := auth.NewSpotifyAuthServer(
		cfg.Spotify.ClientId,
		cfg.Spotify.ClientSecret,
		fmt.Sprintf("http://%s/api/auth", addr),
	)

	database, err := db.New("music_downloader.db")

	if err != nil {
		logs.Error("Failed to initialize database: %v", err)
		return
	}

	defer database.Close()

	svc := spotifyservice.New()

	svc.SetDB(database)

	dl := ytdl.New(cfg.OutputDir)

	handler := app.NewHandler(authServer, svc, dl, cfg)

	authServer.OnAuth = func(client *spotify.Client, httpClient *http.Client, user domain.User) {
		svc.SetClient(client, httpClient)
		handler.SetUser(&user)
	}

	distFS, err := ui.DistFS()
	if err != nil {
		logs.Error("Failed to load embedded frontend: %v", err)
		return
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

	logs.Info("Server running at http://%s", addr)

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logs.Info("Received %v, shutting down...", sig)
		handler.Shutdown()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logs.Error("Server error: %v", err)
	}
}
