package domain

type Song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist,omitempty"`
	Duration    int    `json:"duration"`
	AlbumArt    string `json:"albumArt"`
	TrackNum    int    `json:"trackNum"`
	DiscNum     int    `json:"discNum,omitempty"`
	Year        int    `json:"year,omitempty"`
	PlaylistID  string `json:"playlistId"`
}
