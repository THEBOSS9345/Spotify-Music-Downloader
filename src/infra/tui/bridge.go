package tui

type TrackState struct {
	ID              string
	Title           string
	Artist          string
	Status          string
	Progress        int
	DownloadedBytes int64
	TotalBytes      int64
	Error           string
}

type DownloadState struct {
	Tracks   []TrackState
	Active   int
	Queued   int
	Complete int
	Failed   int
}

var downloadStateCh chan DownloadState

func SetDownloadStateChannel(ch chan DownloadState) {
	downloadStateCh = ch
}

func SendDownloadState(s DownloadState) {
	if downloadStateCh != nil {
		select {
		case downloadStateCh <- s:
		default:
		}
	}
}
