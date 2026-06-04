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

	"music-downloader/src/domain"
	"music-downloader/src/infra/logs"

	"github.com/kkdai/youtube/v2"
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
	ytClient  *youtube.Client
}

func New(outputDir string) *Service {
	return &Service{
		outputDir: outputDir,
		ytClient:  &youtube.Client{},
	}
}

func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	searchQuery := fmt.Sprintf("ytsearch3:%s", query)

	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"--flat-playlist",
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"--print",
		"%(id)s\t%(title)s\t%(duration)s\t%(thumbnail)s\t%(uploader)s\t%(webpage_url)s",
		searchQuery,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("yt-dlp start error: %w", err)
	}

	results := make([]SearchResult, 0, 3)
	scanner := bufio.NewScanner(stdout)

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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	if err := cmd.Wait(); err != nil && len(results) == 0 {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
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

	video, err := s.ytClient.GetVideo(result.URL)
	if err != nil {
		return "", fmt.Errorf("get video info: %w", err)
	}

	formats := video.Formats.Type("audio")
	if len(formats) == 0 {
		return "", fmt.Errorf("no audio formats found for %s", result.Title)
	}
	format := &formats[0]

	onProgress(domain.DownloadDownloading, 25)

	stream, _, err := s.ytClient.GetStream(video, format)
	if err != nil {
		return "", fmt.Errorf("get stream: %w", err)
	}
	defer stream.Close()

	tempPath := filepath.Join(s.outputDir, filename+".m4a")

	outFile, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}

	_, err = io.Copy(outFile, stream)
	outFile.Close()
	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("download stream: %w", err)
	}

	onProgress(domain.DownloadDownloading, 55)

	thumbPath := filepath.Join(s.outputDir, filename+".jpg")
	hasThumb := false

	if result.Thumbnai == "" || result.Thumbnai == "NA" {
		video, err := s.ytClient.GetVideo(result.URL)
		if err == nil && len(video.Thumbnails) > 0 {

			result.Thumbnai = video.Thumbnails[len(video.Thumbnails)-1].URL
		}
	}

	if result.Thumbnai != "" && result.Thumbnai != "NA" {
		resp, err := http.Get(result.Thumbnai)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			f, err := os.Create(thumbPath)
			if err == nil {
				_, err = io.Copy(f, resp.Body)
				f.Close()

				if err == nil {
					hasThumb = true
				} else {
					os.Remove(thumbPath)
				}
			}
		}
	}

	logs.Info("Thumbnail used: %s (exists=%v)", result.Thumbnai, hasThumb)
	onProgress(domain.DownloadDownloading, 70)

	mp3Path := filepath.Join(s.outputDir, filename+".mp3")

	ffArgs := []string{
		"-i", tempPath,
	}

	if hasThumb {
		ffArgs = append(ffArgs,
			"-i", thumbPath,
			"-map", "0:a:0",
			"-map", "1:v:0",
			"-c:v", "mjpeg",
			"-id3v2_version", "3",
			"-metadata:s:v", "title=Album cover",
			"-metadata:s:v", "comment=Cover (front)",
		)
	} else {
		ffArgs = append(ffArgs,
			"-map", "0:a:0",
		)
	}

	ffArgs = append(ffArgs,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		"-metadata", fmt.Sprintf("title=%s", song.Title),
		"-metadata", fmt.Sprintf("artist=%s", song.Artist),
		"-metadata", fmt.Sprintf("album=%s", song.Album),
		"-metadata", fmt.Sprintf("comment=%s", result.URL),
		"-y", mp3Path,
	)
	cmd := exec.CommandContext(ctx, "ffmpeg", ffArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tempPath)
		if hasThumb {
			os.Remove(thumbPath)
		}
		return "", fmt.Errorf("ffmpeg conversion failed: %w\n%s", err, string(output))
	}

	os.Remove(tempPath)
	if hasThumb {
		os.Remove(thumbPath)
	}

	onProgress(domain.DownloadComplete, 100)

	logs.Success("Downloaded: %s -> %s", song.Title, mp3Path)
	return mp3Path, nil
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
