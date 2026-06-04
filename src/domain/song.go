package domain

type Song struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Album     string `json:"album"`
	Duration  int    `json:"duration"`
	AlbumArt  string `json:"albumArt"`
	TrackNum  int    `json:"trackNum"`
	PlaylistID string `json:"playlistId"`
}
