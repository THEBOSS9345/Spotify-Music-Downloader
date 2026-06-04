package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"music-downloader/src/db"
	"music-downloader/src/domain"
	"music-downloader/src/infra/logs"

	"github.com/zmb3/spotify/v2"
)

type Service struct {
	client     *spotify.Client
	httpClient *http.Client
	db         *db.DB
}

func New() *Service {
	return &Service{}
}

func (s *Service) SetDB(database *db.DB) {
	s.db = database
}

func (s *Service) SetClient(client *spotify.Client, httpClient *http.Client) {
	s.client = client
	s.httpClient = httpClient
}

func (s *Service) IsReady() bool {
	return s.client != nil
}

func (s *Service) GetPlaylists(ctx context.Context) ([]domain.Playlist, error) {
	if s.client == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	if s.db != nil {
		cached, stale, err := s.db.GetCachedPlaylistsStale()
		if err == nil && cached != nil {
			if stale {
				go s.refreshPlaylists()
			}
			return cached, nil
		}
	}

	return s.fetchPlaylists(ctx)
}

func (s *Service) refreshPlaylists() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := s.fetchPlaylists(ctx)
	if err != nil {
		logs.Error("background playlist refresh: %v", err)
	}
}

func (s *Service) fetchPlaylists(ctx context.Context) ([]domain.Playlist, error) {
	playlists, err := s.client.CurrentUsersPlaylists(ctx, spotify.Limit(50))
	if err != nil {
		logs.Error("Failed to fetch playlists: %v", err)
		return nil, err
	}

	result := make([]domain.Playlist, 0, len(playlists.Playlists))
	currentUser, err := s.client.CurrentUser(ctx)
	if err != nil {
		logs.Error("Failed to get current user: %v", err)
		return nil, err
	}

	for _, p := range playlists.Playlists {
		if string(p.Owner.ID) != currentUser.ID {
			continue
		}

		pl := domain.Playlist{
			ID:          string(p.ID),
			Name:        p.Name,
			Description: p.Description,
			TrackCount:  int(p.Tracks.Total),
			Owner:       p.Owner.DisplayName,
		}
		if len(p.Images) > 0 {
			pl.ImageURL = p.Images[0].URL
		}
		result = append(result, pl)
	}

	if s.db != nil {
		_ = s.db.CachePlaylists(result)
	}

	return result, nil
}

type rawItem struct {
	AddedAt string      `json:"added_at"`
	Track   rawItemData `json:"item"`
}

type rawItemData struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Artists  []rawArtist `json:"artists"`
	Album    rawAlbum    `json:"album"`
	Duration int         `json:"duration_ms"`
	TrackNum int         `json:"track_number"`
}

type rawArtist struct {
	Name string `json:"name"`
}

type rawAlbum struct {
	Name   string     `json:"name"`
	Images []rawImage `json:"images"`
}

type rawImage struct {
	URL string `json:"url"`
}

type rawPlaylistItemsResponse struct {
	Href  string    `json:"href"`
	Next  string    `json:"next"`
	Items []rawItem `json:"items"`
}

type rawSearchResponse struct {
	Tracks rawSearchTracks `json:"tracks"`
}

type rawSearchTracks struct {
	Items []rawSearchItem `json:"items"`
}

type rawSearchItem struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Artists  []rawArtist `json:"artists"`
	Album    rawAlbum    `json:"album"`
	Duration int         `json:"duration_ms"`
}

func (s *Service) SearchTracks(ctx context.Context, query string) ([]domain.Song, error) {
	if s.httpClient == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	url := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=10", url.QueryEscape(query))
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify search: HTTP %d", resp.StatusCode)
	}

	var data rawSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("parse search response: %v", err)
	}

	result := make([]domain.Song, 0, len(data.Tracks.Items))
	for _, item := range data.Tracks.Items {
		song := domain.Song{
			ID:    item.ID,
			Title: item.Name,
		}
		if len(item.Artists) > 0 {
			song.Artist = item.Artists[0].Name
		}
		if item.Album.Name != "" {
			song.Album = item.Album.Name
			if len(item.Album.Images) > 0 {
				song.AlbumArt = item.Album.Images[0].URL
			}
		}
		if item.Duration > 0 {
			song.Duration = item.Duration / 1000
		}
		result = append(result, song)
	}

	return result, nil
}

func (s *Service) GetPlaylistTracks(ctx context.Context, playlistID string) ([]domain.Song, error) {
	if s.client == nil {
		return nil, fmt.Errorf("spotify client not initialized")
	}

	if s.db != nil {
		cached, stale, err := s.db.GetCachedTracksStale(playlistID)
		if err == nil && cached != nil {
			if stale {
				go s.refreshTracks(playlistID)
			}
			return cached, nil
		}
	}

	return s.fetchTracks(ctx, playlistID)
}

func (s *Service) refreshTracks(playlistID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, err := s.fetchTracks(ctx, playlistID)
	if err != nil {
		logs.Error("background tracks refresh for %s: %v", playlistID, err)
	}
}

func (s *Service) fetchTracks(ctx context.Context, playlistID string) ([]domain.Song, error) {
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/items?limit=50", playlistID)

	var result []domain.Song
	for {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("spotify: HTTP %d", resp.StatusCode)
		}

		var data rawPlaylistItemsResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("parse response: %v", err)
		}

		for _, item := range data.Items {
			song := domain.Song{
				ID:         item.Track.ID,
				Title:      item.Track.Name,
				PlaylistID: playlistID,
			}
			if len(item.Track.Artists) > 0 {
				song.Artist = item.Track.Artists[0].Name
			}
			if item.Track.Album.Name != "" {
				song.Album = item.Track.Album.Name
				if len(item.Track.Album.Images) > 0 {
					song.AlbumArt = item.Track.Album.Images[0].URL
				}
			}
			if item.Track.Duration > 0 {
				song.Duration = item.Track.Duration / 1000
			}
			if item.Track.TrackNum > 0 {
				song.TrackNum = item.Track.TrackNum
			}
			result = append(result, song)
		}

		if data.Next == "" {
			break
		}
		url = data.Next
	}

	if s.db != nil {
		_ = s.db.CacheTracks(playlistID, result)
	}

	return result, nil
}
