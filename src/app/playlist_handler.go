package app

import (
	"net/http"

	"music-downloader/src/infra/logs"
)

func (h *Handler) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	playlists, err := h.spotify.GetPlaylists(r.Context())
	if err != nil {
		logs.Error("fetch playlists: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(playlists))
	for i, p := range playlists {

		
		if p.OwnerID != h.user.ID {
			p.TrackCount = 0
		}

		result[i] = map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"imageUrl":    p.ImageURL,
			"trackCount":  p.TrackCount,
			"owner":       p.Owner,
		}
	}
	h.json(w, result)
}

func (h *Handler) handlePlaylistTracks(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	id := r.PathValue("id")
	songs, err := h.spotify.GetPlaylistTracks(r.Context(), id)
	if err != nil {
		logs.Error("fetch tracks: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(songs))
	for i, s := range songs {
		result[i] = map[string]interface{}{
			"id":         s.ID,
			"title":      s.Title,
			"artist":     s.Artist,
			"album":      s.Album,
			"duration":   s.Duration,
			"albumArt":   s.AlbumArt,
			"trackNum":   s.TrackNum,
			"playlistId": s.PlaylistID,
		}
	}
	h.json(w, result)
}
