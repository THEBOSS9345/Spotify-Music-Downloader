package app

import (
	"net/http"

	"spotscoop/src/infra/logs"
)

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing query param q", http.StatusBadRequest)
		return
	}

	songs, err := h.spotify.SearchTracks(r.Context(), q)
	if err != nil {
		logs.Error("search: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, len(songs))
	for i, s := range songs {
		result[i] = map[string]interface{}{
			"id":       s.ID,
			"title":    s.Title,
			"artist":   s.Artist,
			"album":    s.Album,
			"duration": s.Duration,
			"albumArt": s.AlbumArt,
		}
	}
	h.json(w, result)
}
