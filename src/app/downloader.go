package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"music-downloader/src/db"
	"music-downloader/src/domain"
	"music-downloader/src/infra/logs"
	"music-downloader/src/ytdl"

	"github.com/google/uuid"
)

type task struct {
	Download domain.Download
	cancel   context.CancelFunc
}

const maxConcurrent = 1

type Downloader struct {
	ytdl      *ytdl.Service
	db        *db.DB
	mu        sync.RWMutex
	tasks     map[string]*task
	semaphore chan struct{}
	broker    *Broker
}

func NewDownloader(ytdl *ytdl.Service, database *db.DB, broker *Broker) *Downloader {
	return &Downloader{
		ytdl:      ytdl,
		db:        database,
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
		_ = d.db.SaveDownload(dl)
		t := &task{Download: dl}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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

	if len(tasks) > 0 {
		return tasks
	}

	fromDB, err := d.db.GetDownloads()
	if err != nil {
		return nil
	}
	return fromDB
}

func (d *Downloader) CleanupStale() {
	all, err := d.db.GetDownloads()
	if err != nil {
		logs.Warning("cleanup: failed to get downloads: %v", err)
		return
	}
	count := 0
	for _, dl := range all {
		if dl.Status == domain.DownloadSearching || dl.Status == domain.DownloadDownloading || dl.Status == domain.DownloadConverting {
			dl.Status = domain.DownloadFailed
			dl.Error = "Interrupted (process was restarted)"
			_ = d.db.SaveDownload(dl)
			count++
		}
	}
	if count > 0 {
		logs.Info("cleanup: marked %d stale download(s) as failed", count)
	}
	d.ytdl.CleanupOrphans()
	d.publishState()
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
		_ = d.db.SaveDownload(t.Download)
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
	_ = d.db.SaveDownload(t.Download)
	d.publishState()
	logs.Success("Finished %s in %v", t.Download.Song.Title, time.Since(start))
}
