package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"music-downloader/src/domain"
	"music-downloader/src/infra/logs"
	"music-downloader/src/ytdl"

	"github.com/google/uuid"
)

type task struct {
	Download domain.Download
	cancel   context.CancelFunc
}

type Downloader struct {
	ytdl      *ytdl.Service
	mu        sync.RWMutex
	tasks     map[string]*task
	semaphore chan struct{}
	broker    *Broker
}

func NewDownloader(ytdl *ytdl.Service, broker *Broker, maxConcurrent int) *Downloader {
	return &Downloader{
		ytdl:      ytdl,
		tasks:     make(map[string]*task),
		semaphore: make(chan struct{}, maxConcurrent),
		broker:    broker,
	}
}

func (d *Downloader) publishState() {
	if !d.broker.HasClients() {
		return
	}
	all := d.GetAll()
	active, queued := d.GetActive()
	d.broker.Publish(BuildDownloadState(all, active, queued))
}

func (d *Downloader) Start(songs []domain.Song) string {
	batchID := uuid.NewString()
	for i := range songs {
		dl := domain.Download{
			ID:        uuid.NewString(),
			Song:      songs[i],
			Status:    domain.DownloadPending,
			CreatedAt: time.Now().Unix(),
		}
		t := &task{Download: dl}
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour*6)
		t.cancel = cancel
		d.mu.Lock()
		d.tasks[t.Download.ID] = t
		d.mu.Unlock()
		go d.process(ctx, t)
	}
	return batchID
}

func (d *Downloader) GetActive() (active []domain.Download, queued []domain.Download) {
	d.mu.RLock()
	for _, t := range d.tasks {
		switch t.Download.Status {
		case domain.DownloadSearching, domain.DownloadDownloading, domain.DownloadConverting:
			active = append(active, t.Download)
		case domain.DownloadPending:
			queued = append(queued, t.Download)
		}
	}
	d.mu.RUnlock()
	return
}

func (d *Downloader) GetAll() []domain.Download {
	d.mu.RLock()
	tasks := make([]domain.Download, 0, len(d.tasks))
	for _, t := range d.tasks {
		tasks = append(tasks, t.Download)
	}
	d.mu.RUnlock()
	return tasks
}

func (d *Downloader) CleanupStale() {
	d.ytdl.CleanupOrphans()
	d.publishState()
}

func (d *Downloader) Retry(ids []string) {
	for _, id := range ids {
		d.mu.RLock()
		t, exists := d.tasks[id]
		d.mu.RUnlock()
		if exists && t.Download.Status == domain.DownloadFailed {
			t.Download.Status = domain.DownloadPending
			t.Download.Error = ""
			go d.process(context.Background(), t)
		}
	}
}

func (d *Downloader) Clear() {
	d.mu.Lock()
	for _, t := range d.tasks {
		if t.cancel != nil {
			t.cancel()
		}
	}
	d.tasks = make(map[string]*task)
	d.mu.Unlock()
	d.publishState()
}

func (d *Downloader) process(ctx context.Context, t *task) {
	d.semaphore <- struct{}{}
	defer func() { <-d.semaphore }()

	update := func(status domain.DownloadStatus, progress int) {
		d.mu.Lock()
		t.Download.Status = status
		t.Download.Progress = progress
		d.mu.Unlock()
		d.publishState()
	}
	update(domain.DownloadSearching, 0)
	query := fmt.Sprintf("%s - %s", t.Download.Song.Artist, t.Download.Song.Title)
	logs.Info("Searching: %s", query)

	results, err := d.ytdl.Search(ctx, query)

	if err != nil {
		logs.Error("Search failed for %s: %v", query, err)
		t.Download.Error = fmt.Sprintf("Search failed: %v", err)
		update(domain.DownloadFailed, 0)
		return
	}

	if len(results) == 0 {
		t.Download.Error = "No YouTube results found"
		update(domain.DownloadFailed, 0)
		return
	}
	logs.Info("Found %d results for %s", len(results), query)
	update(domain.DownloadDownloading, 10)
	start := time.Now()
	path, err := d.ytdl.Download(ctx, results[0], t.Download.Song, func(status domain.DownloadStatus, progress int) {
		update(status, progress)
	})
	if err != nil {
		logs.Error("Download failed for %s: %v", t.Download.Song.Title, err)
		t.Download.Error = err.Error()
		update(domain.DownloadFailed, 0)
		return
	}
	t.Download.OutputPath = path
	t.Download.Status = domain.DownloadComplete
	t.Download.Progress = 100
	d.publishState()
	logs.Success("Finished: %s in %v", t.Download.Song.Title, time.Since(start))
}
