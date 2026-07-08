package app

import (
	"encoding/json"
	"net/http"

	"spotscoop/src/auth"
	"spotscoop/src/domain"
	"spotscoop/src/infra/config"
	"spotscoop/src/infra/logs"
	"spotscoop/src/spotify"
	"spotscoop/src/ytdl"
)

type Handler struct {
	auth       *auth.SpotifyAuthServer
	spotify    *spotify.Service
	user       *domain.User
	downloader *Downloader
	broker     *Broker
}

func NewHandler(auth *auth.SpotifyAuthServer, spotify *spotify.Service, ytdl *ytdl.Service, conf *config.Config) *Handler {
	broker := NewBroker()

	return &Handler{
		auth:       auth,
		spotify:    spotify,
		downloader: NewDownloader(ytdl, broker, conf.MaxConcurrentDownloads),
		broker:     broker,
	}
}

func (h *Handler) CleanupStale()             { h.downloader.CleanupStale() }
func (h *Handler) Shutdown()                 { h.downloader.Clear(); h.downloader.CleanupStale() }
func (h *Handler) SetUser(user *domain.User) { h.user = user }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/status", h.handleStatus)
	mux.HandleFunc("GET /api/login", h.handleLogin)
	mux.HandleFunc("GET /api/user", h.handleUser)
	mux.HandleFunc("GET /api/playlists", h.handlePlaylists)
	mux.HandleFunc("POST /api/playlists/refresh", h.handlePlaylistsRefresh)
	mux.HandleFunc("POST /api/playlists/import", h.handlePlaylistImport)
	mux.HandleFunc("GET /api/playlists/{id}/tracks", h.handlePlaylistTracks)
	mux.HandleFunc("POST /api/playlists/{id}/tracks/refresh", h.handlePlaylistTracksRefresh)
	mux.HandleFunc("GET /api/search", h.handleSearch)
	mux.HandleFunc("POST /api/download", h.handleDownload)
	mux.HandleFunc("POST /api/retry", h.handleRetry)
	mux.HandleFunc("GET /api/downloads", h.handleDownloads)
	mux.HandleFunc("POST /api/logout", h.handleLogout)
	mux.HandleFunc("GET /api/events", h.handleEvents)
}

func (h *Handler) json(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logs.Error("json encode: %v", err)
	}
}

func (h *Handler) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if !h.spotify.IsReady() {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return false
	}
	return true
}
