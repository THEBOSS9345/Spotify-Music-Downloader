package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"spotscoop/src/domain"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

type cachedPlaylist struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	TrackCount  int    `json:"track_count"`
	Owner       string `json:"owner"`
}

type cachedTrack struct {
	ID          string `json:"id"`
	PlaylistID  string `json:"playlist_id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"album_artist"`
	Duration    int    `json:"duration"`
	AlbumArt    string `json:"album_art"`
	TrackNum    int    `json:"track_num"`
	DiscNum     int    `json:"disc_num"`
	Year        int    `json:"year"`
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS playlists (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		image_url TEXT NOT NULL DEFAULT '',
		track_count INTEGER NOT NULL DEFAULT 0,
		owner TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS tracks (
		id TEXT NOT NULL,
		playlist_id TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		artist TEXT NOT NULL DEFAULT '',
		album TEXT NOT NULL DEFAULT '',
		album_artist TEXT NOT NULL DEFAULT '',
		duration INTEGER NOT NULL DEFAULT 0,
		album_art TEXT NOT NULL DEFAULT '',
		track_num INTEGER NOT NULL DEFAULT 0,
		disc_num INTEGER NOT NULL DEFAULT 0,
		year INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (id, playlist_id)
	);
	CREATE TABLE IF NOT EXISTS downloads (
		id TEXT PRIMARY KEY,
		song_id TEXT NOT NULL DEFAULT '',
		song_title TEXT NOT NULL DEFAULT '',
		song_artist TEXT NOT NULL DEFAULT '',
		song_album TEXT NOT NULL DEFAULT '',
		playlist_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		progress INTEGER NOT NULL DEFAULT 0,
		output_path TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);`
	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	db.conn.Exec("ALTER TABLE tracks ADD COLUMN album_artist TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("ALTER TABLE tracks ADD COLUMN disc_num INTEGER NOT NULL DEFAULT 0")
	db.conn.Exec("ALTER TABLE tracks ADD COLUMN year INTEGER NOT NULL DEFAULT 0")
	return nil
}

const cacheTTL = 5 * time.Minute

func (db *DB) GetCachedPlaylists() ([]domain.Playlist, error) {
	playlists, stale, err := db.GetCachedPlaylistsStale()
	if err != nil {
		return nil, err
	}
	if stale {
		return nil, nil
	}
	return playlists, nil
}

func (db *DB) GetCachedPlaylistsStale() ([]domain.Playlist, bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query(`SELECT id, name, description, image_url, track_count, owner, updated_at FROM playlists`)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var playlists []domain.Playlist
	var anyStale bool
	cutoff := time.Now().Add(-cacheTTL).Unix()
	for rows.Next() {
		var cp cachedPlaylist
		var updatedAt int64
		if err := rows.Scan(&cp.ID, &cp.Name, &cp.Description, &cp.ImageURL, &cp.TrackCount, &cp.Owner, &updatedAt); err != nil {
			continue
		}
		if updatedAt < cutoff {
			anyStale = true
		}
		playlists = append(playlists, domain.Playlist{
			ID:          cp.ID,
			Name:        cp.Name,
			Description: cp.Description,
			ImageURL:    cp.ImageURL,
			TrackCount:  cp.TrackCount,
			Owner:       cp.Owner,
		})
	}
	if len(playlists) == 0 {
		return nil, false, nil
	}
	return playlists, anyStale, nil
}

func (db *DB) CachePlaylists(playlists []domain.Playlist) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	for _, p := range playlists {
		_, err := tx.Exec(`INSERT OR REPLACE INTO playlists (id, name, description, image_url, track_count, owner, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.Description, p.ImageURL, p.TrackCount, p.Owner, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetCachedTracks(playlistID string) ([]domain.Song, error) {
	tracks, stale, err := db.GetCachedTracksStale(playlistID)
	if err != nil {
		return nil, err
	}
	if stale {
		return nil, nil
	}
	return tracks, nil
}

func (db *DB) GetCachedTracksStale(playlistID string) ([]domain.Song, bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var updatedAt int64
	err := db.conn.QueryRow(`SELECT updated_at FROM playlists WHERE id = ?`, playlistID).Scan(&updatedAt)
	if err != nil {
		return nil, false, err
	}
	stale := time.Now().Add(-cacheTTL).Unix() > updatedAt

	rows, err := db.conn.Query(`SELECT id, title, artist, album, album_artist, duration, album_art, track_num, disc_num, year FROM tracks WHERE playlist_id = ? ORDER BY track_num`, playlistID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var songs []domain.Song
	for rows.Next() {
		var ct cachedTrack
		if err := rows.Scan(&ct.ID, &ct.Title, &ct.Artist, &ct.Album, &ct.AlbumArtist, &ct.Duration, &ct.AlbumArt, &ct.TrackNum, &ct.DiscNum, &ct.Year); err != nil {
			continue
		}
		songs = append(songs, domain.Song{
			ID:          ct.ID,
			Title:       ct.Title,
			Artist:      ct.Artist,
			Album:       ct.Album,
			AlbumArtist: ct.AlbumArtist,
			Duration:    ct.Duration,
			AlbumArt:    ct.AlbumArt,
			TrackNum:    ct.TrackNum,
			DiscNum:     ct.DiscNum,
			Year:        ct.Year,
			PlaylistID:  playlistID,
		})
	}
	if len(songs) == 0 {
		return nil, false, nil
	}
	return songs, stale, nil
}

func (db *DB) CacheTracks(playlistID string, songs []domain.Song) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	_, _ = tx.Exec(`UPDATE playlists SET track_count = ?, updated_at = ? WHERE id = ?`, len(songs), now, playlistID)

	for _, s := range songs {
		_, err := tx.Exec(`INSERT OR REPLACE INTO tracks (id, playlist_id, title, artist, album, album_artist, duration, album_art, track_num, disc_num, year) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, playlistID, s.Title, s.Artist, s.Album, s.AlbumArtist, s.Duration, s.AlbumArt, s.TrackNum, s.DiscNum, s.Year)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) SaveDownload(d domain.Download) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	now := time.Now().Unix()
	_, err := db.conn.Exec(`INSERT OR REPLACE INTO downloads (id, song_id, song_title, song_artist, song_album, playlist_id, status, progress, output_path, error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM downloads WHERE id = ?), ?), ?)`,
		d.ID, d.Song.ID, d.Song.Title, d.Song.Artist, d.Song.Album, d.Song.PlaylistID,
		string(d.Status), d.Progress, d.OutputPath, d.Error,
		d.ID, now, now)
	return err
}

func (db *DB) GetDownloads() ([]domain.Download, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query(`SELECT id, song_id, song_title, song_artist, song_album, playlist_id, status, progress, output_path, error, created_at FROM downloads ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []domain.Download
	for rows.Next() {
		var d domain.Download
		var status string
		if err := rows.Scan(&d.ID, &d.Song.ID, &d.Song.Title, &d.Song.Artist, &d.Song.Album, &d.Song.PlaylistID, &status, &d.Progress, &d.OutputPath, &d.Error, &d.CreatedAt); err != nil {
			continue
		}
		d.Status = domain.DownloadStatus(status)
		downloads = append(downloads, d)
	}
	return downloads, nil
}

func (db *DB) ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (db *DB) FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
