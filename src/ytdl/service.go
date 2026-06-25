package ytdl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"music-downloader/src/domain"
	"music-downloader/src/infra/logs"

	"github.com/lrstanley/go-ytdlp"
)

type SearchResult struct {
	Title    string `json:"title"`
	VideoID  string `json:"id"`
	Duration int    `json:"duration"`
	Thumbnai string `json:"thumbnail"`
	Uploader string `json:"uploader"`
	URL      string `json:"webpage_url"`
}

type Service struct {
	outputDir string
	dl        *ytdlp.Command
}

func New(outputDir string) *Service {
	dl := ytdlp.New().
		NoPlaylist().
		NoWarnings()

	if ffmpeg, err := exec.LookPath("ffmpeg"); err == nil {
		dl = dl.FFmpegLocation(filepath.Dir(ffmpeg))
	} else {
		logs.Warning("ffmpeg not found in PATH: %v", err)
	}

	return &Service{
		outputDir: outputDir,
		dl:          dl,
	}
}

func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	searchQuery := fmt.Sprintf("ytsearch3:%s", query)

	dl := s.dl.Clone().
		FlatPlaylist().
		Quiet().
		Print("%(id)s\t%(title)s\t%(duration)s\t%(thumbnail)s\t%(uploader)s\t%(webpage_url)s")

	result, err := dl.Run(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	results := make([]SearchResult, 0, 3)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 6)
		if len(parts) < 6 {
			continue
		}

		duration := 0
		if parts[2] != "" {
			if d, err := strconv.Atoi(parts[2]); err == nil {
				duration = d
			}
		}

		results = append(results, SearchResult{
			VideoID:  parts[0],
			Title:    parts[1],
			Duration: duration,
			Thumbnai: parts[3],
			Uploader: parts[4],
			URL:      parts[5],
		})
	}

	return results, nil
}

func (s *Service) Download(
	ctx context.Context,
	result SearchResult,
	song domain.Song,
	onProgress func(domain.DownloadStatus, int),
) (string, error) {

	onProgress(domain.DownloadDownloading, 10)

	safeArtist := safeFilename(song.Artist)
	safeTitle := safeFilename(song.Title)
	filename := fmt.Sprintf("%s - %s", safeArtist, safeTitle)

	os.Remove(filepath.Join(s.outputDir, filename+".m4a"))
	os.Remove(filepath.Join(s.outputDir, filename+".mp3"))
	os.Remove(filepath.Join(s.outputDir, filename+".jpg"))

	onProgress(domain.DownloadDownloading, 25)

	if err := s.downloadWithYtDlp(ctx, result, filename, onProgress); err != nil {
		logs.Error("yt-dlp download failed for %s: %v", song.Title, err)
		return "", err
	}

	onProgress(domain.DownloadDownloading, 55)

	hasThumb := s.fetchThumbnail(result, filename)

	onProgress(domain.DownloadDownloading, 70)

	m4aPath := filepath.Join(s.outputDir, filename+".m4a")
	mp3Path := filepath.Join(s.outputDir, filename+".mp3")
	thumbPath := filepath.Join(s.outputDir, filename+".jpg")

	ffmpegArgs := buildFFmpegArgs(m4aPath, mp3Path, thumbPath, hasThumb, song, result.URL)
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(m4aPath)
		os.Remove(thumbPath)
		logs.Error("ffmpeg conversion failed for %s: %v", song.Title, err)
		return "", fmt.Errorf("ffmpeg conversion failed: %w\n%s", err, string(output))
	}

	os.Remove(m4aPath)
	os.Remove(thumbPath)

	onProgress(domain.DownloadComplete, 100)

	logs.Success("Downloaded: %s -> %s", song.Title, mp3Path)
	return mp3Path, nil
}

func buildFFmpegArgs(m4aPath, mp3Path, thumbPath string, hasThumb bool, song domain.Song, sourceURL string) []string {
	args := []string{"-i", m4aPath}

	if hasThumb {
		args = append(args,
			"-i", thumbPath,
			"-map", "0:a:0",
			"-map", "1:v:0",
			"-c:v", "mjpeg",
			"-vf", "crop='min(iw,ih)':'min(iw,ih)',scale=800:800",
			"-id3v2_version", "3",
			"-metadata:s:v", "title=Album cover",
			"-metadata:s:v", "comment=Cover (front)",
			"-disposition:v", "attached_pic",
		)
	} else {
		args = append(args, "-map", "0:a:0")
	}

	args = append(args,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		"-metadata", fmt.Sprintf("title=%s", song.Title),
		"-metadata", fmt.Sprintf("artist=%s", song.Artist),
		"-metadata", fmt.Sprintf("album=%s", song.Album),
		"-metadata", fmt.Sprintf("comment=%s", sourceURL),
		"-y", mp3Path,
	)
	return args
}

func (s *Service) downloadWithYtDlp(ctx context.Context, result SearchResult, filename string, onProgress func(domain.DownloadStatus, int)) error {
	outputPath := filepath.Join(s.outputDir, filename+".m4a")

	dl := s.dl.Clone().
		ExtractAudio().
		AudioFormat("m4a").
		AudioQuality("0").
		AgeLimit(99).
		Output(outputPath)

	dl.ProgressFunc(500*time.Millisecond, func(prog ytdlp.ProgressUpdate) {
		if prog.Status == ytdlp.ProgressStatusDownloading && prog.TotalBytes > 0 {
			pct := int(prog.Percent())
			if pct >= 0 && pct <= 100 {
				onProgress(domain.DownloadDownloading, 25+pct/5)
			}
		}
	})

	cmd := dl.BuildCommand(ctx, result.URL)
	logs.Info("yt-dlp command: %s %v", cmd.Path, cmd.Args[1:])

	r, err := dl.Run(ctx, result.URL)
	if r != nil {
		logs.Info("yt-dlp stdout: %s", r.Stdout)
		logs.Info("yt-dlp stderr: %s", r.Stderr)
	}
	if err != nil {
		return fmt.Errorf("yt-dlp download failed: %w", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		entries, _ := os.ReadDir(s.outputDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), filename+".") {
				ext := filepath.Ext(e.Name())
				if ext != ".m4a" {
					os.Rename(filepath.Join(s.outputDir, e.Name()), outputPath)
					break
				}
			}
		}
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			return fmt.Errorf("yt-dlp did not produce output file for %s", result.Title)
		}
	}

	entries, _ := os.ReadDir(s.outputDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), filename+".") && filepath.Ext(e.Name()) != ".m4a" {
			os.Remove(filepath.Join(s.outputDir, e.Name()))
		}
	}

	return nil
}

func (s *Service) fetchThumbnail(result SearchResult, filename string) bool {
	thumbPath := filepath.Join(s.outputDir, filename+".jpg")

	urls := []string{
		fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", result.VideoID),
		fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", result.VideoID),
	}
	if result.Thumbnai != "" && result.Thumbnai != "NA" {
		urls = append(urls, result.Thumbnai)
	}

	for _, thumbURL := range urls {
		for attempt := range 3 {
			if attempt > 0 {
				time.Sleep(time.Second)
			}

			resp, err := http.Get(thumbURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				continue
			}

			f, err := os.Create(thumbPath)
			if err != nil {
				resp.Body.Close()
				continue
			}

			_, err = io.Copy(f, resp.Body)
			resp.Body.Close()
			f.Close()

			if err != nil {
				os.Remove(thumbPath)
				continue
			}
			return true
		}
	}
	return false
}

func (s *Service) CleanupOrphans() {
	entries, err := os.ReadDir(s.outputDir)

	if err != nil {
		logs.Warning("cleanup: cannot read output dir: %v", err)
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".m4a" && ext != ".jpg" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ext)
		mp3Path := filepath.Join(s.outputDir, base+".mp3")
		if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
			path := filepath.Join(s.outputDir, e.Name())
			os.Remove(path)
			logs.Info("cleanup: removed orphaned temp file: %s", path)
		}
	}
	logs.Info("cleanup: finished scanning output dir")
}

func safeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "-",
		"*", "_", "?", "", "<", "",
		">", "", "|", "_", "\"", "'",
	)
	return strings.TrimSpace(replacer.Replace(name))
}
