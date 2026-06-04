package app

import (
	"net/http"
)

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	h.json(w, map[string]bool{"authenticated": h.spotify.IsReady()})
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	h.json(w, map[string]string{"url": h.auth.GetLoginURL()})
}

func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}
	if h.user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}
	h.json(w, map[string]interface{}{
		"id":          h.user.ID,
		"displayName": h.user.DisplayName,
		"avatarUrl":   h.user.AvatarURL,
		"email":       h.user.Email,
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout()
	h.spotify.SetClient(nil, nil)
	h.user = nil
	h.downloader.Clear()
	h.json(w, map[string]string{"status": "ok"})
}
