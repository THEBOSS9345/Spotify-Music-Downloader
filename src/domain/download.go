package domain

type DownloadStatus string

const (
	DownloadPending     DownloadStatus = "pending"
	DownloadSearching   DownloadStatus = "searching"
	DownloadDownloading DownloadStatus = "downloading"
	DownloadConverting  DownloadStatus = "converting"
	DownloadComplete    DownloadStatus = "complete"
	DownloadFailed      DownloadStatus = "failed"
)

type Download struct {
	ID         string         `json:"id"`
	Song       Song           `json:"song"`
	Status     DownloadStatus `json:"status"`
	Progress   int            `json:"progress"`
	OutputPath string         `json:"outputPath"`
	Error      string         `json:"error"`
	CreatedAt  int64          `json:"createdAt"`
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	Email       string `json:"email"`
}
