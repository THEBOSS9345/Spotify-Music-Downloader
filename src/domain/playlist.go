package domain

type Playlist struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ImageURL      string `json:"imageUrl"`
	TrackCount    int    `json:"trackCount"`
	Owner         string `json:"owner"`
	OwnerID       string `json:"-"`
	Collaborative bool   `json:"-"`
}
