package downloader

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
    "io"
    "errors"
    
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/utils"
    
)

// VideoDownloader handles video downloading and processing
type VideoDownloader struct {
    downloadDir     string
    logger          *utils.EnhancedLogger
    retryOpts       *utils.RetryOptions
    dependencyPaths map[string]string // New field to store paths
}

// DownloadResult contains paths to downloaded files
type DownloadResult struct {
    VideoPath       string
    VideoWithSubPath string
    AudioPath       string
    SubtitlePath    string
    HasSubtitle     bool
    FileSize        int64
    Duration        int
    Error           error
    ThumbnailPath   string
}

// getCookiePath dynamically generates the absolute path to the cookie file for a given domain
func getCookiePath(domain string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic("Unable to get current directory")
	}
	return filepath.Join(cwd, "app", "config", domain+"_cookies.txt")
}

// NewVideoDownloader creates a new video downloader
// Modified to accept dependencyPaths
func NewVideoDownloader(downloadDir string, logger *utils.EnhancedLogger, maxRetries int, dependencyPaths map[string]string) *VideoDownloader {
	retryOpts := utils.DefaultRetryOptions().
		WithMaxRetries(maxRetries).
		WithLogger(logger)

	return &VideoDownloader{
		downloadDir:     downloadDir,
		logger:          logger,
		retryOpts:       retryOpts,
		dependencyPaths: dependencyPaths, // Store the paths
	}
}

func (d *VideoDownloader) getCookiesArgs(url string) []string {
	domainCookies := map[string]string{
		"tiktok.com":     "tiktok",
		"twitter.com":    "twitter",
		"x.com":          "twitter",
		"youtube.com":    "youtube",
		"instagram.com":  "instagramreels",
		"facebook.com":   "facebook",
		"pinterest.com":  "pinterest",
	}

	// Get user-agent from env or fallback to default Android mobile agent
	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Mobile Safari/537.36"
		d.logger.Info("USER_AGENT env not set, using default mobile UA: %s", userAgent)
	} else {
		d.logger.Info("Using USER_AGENT from env: %s", userAgent)
	}

	args := []string{
		"--geo-bypass-country", "US",
		"--user-agent", userAgent,
	}

	for domain, cookieName := range domainCookies {
		if strings.Contains(url, domain) {
			cookiePath := getCookiePath(cookieName)
			d.logger.Info("Matched domain: %s, looking for cookie file: %s", domain, cookiePath)

			if _, err := os.Stat(cookiePath); err == nil {
				d.logger.Info("Cookie file found: %s", cookiePath)
				args = append(args, "--cookies", cookiePath)
			} else {
				d.logger.Warn("Expected cookie file not found for domain %s: %s", domain, cookiePath)
			}
			break
		}
	}

	return args
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Download downloads a video and returns paths to the downloaded files
func (d *VideoDownloader) Download(ctx context.Context, url string, captionLang string) (*DownloadResult, error) {
    // Create a unique download directory for this request
    downloadID := fmt.Sprintf("%d", time.Now().UnixNano())
    downloadPath := filepath.Join(d.downloadDir, downloadID)
    
    // Create download directory
    if err := os.MkdirAll(downloadPath, 0755); err != nil {
        return nil, fmt.Errorf("failed to create download directory: %w", err)
    }
    
    // Defer cleanup of download directory
    defer func() {
        // Keep files for a while to allow sending to user
        // They will be cleaned up by a separate process
    }()
    
    result := &DownloadResult{}
    
    // Download thumbnail
    d.logger.Info("Downloading high-resolution PNG thumbnail from %s", url)
    err := utils.RetryWithContext(ctx, func() error {
        return d.downloadThumbnail(ctx, url, downloadPath)
    }, d.retryOpts)
    
    if err != nil {
        d.logger.Warn("Failed to download thumbnail: %v", err)
        // Continue without thumbnail
    } else {
        thumbnailPath := filepath.Join(downloadPath, "thumbnail.png")
        if fileExists(thumbnailPath) {
            result.ThumbnailPath = thumbnailPath
            d.logger.Info("Successfully downloaded high-resolution PNG thumbnail to %s", thumbnailPath)
        }
    }
    
    // Download primary video (best video + best audio merged)
    d.logger.Info("Downloading primary video from %s", url)
    err = utils.RetryWithContext(ctx, func() error {
        return d.downloadPrimaryVideo(ctx, url, downloadPath)
    }, d.retryOpts)
    
    if err != nil {
        return nil, fmt.Errorf("failed to download primary video after %d retries: %w", d.retryOpts.MaxRetries, err)
    }
    
    result.VideoPath = filepath.Join(downloadPath, "video_base.mp4")
    
    // Get file size
    fileInfo, err := os.Stat(result.VideoPath)
    if err == nil {
        result.FileSize = fileInfo.Size()
    }
    
    // Download subtitle if available
    d.logger.Info("Downloading subtitle in language %s from %s", captionLang, url)
    var subtitlePath string
    err = utils.RetryWithContext(ctx, func() error {
        var err error
        subtitlePath, err = d.downloadSubtitle(ctx, url, captionLang, downloadPath)
        return err
    }, d.retryOpts)
    
    if err != nil {
        d.logger.Warn("Failed to download subtitle after %d retries: %v", d.retryOpts.MaxRetries, err)
        // Continue without subtitle
    } else if subtitlePath != "" {
        result.SubtitlePath = subtitlePath
        result.HasSubtitle = true
        
        // Embed subtitle into video
        d.logger.Info("Embedding subtitle into video")
        err := utils.RetryWithContext(ctx, func() error {
            return d.embedSubtitle(ctx, result.VideoPath, subtitlePath, downloadPath)
        }, d.retryOpts)
        
        if err != nil {
            d.logger.Warn("Failed to embed subtitle after %d retries: %v", d.retryOpts.MaxRetries, err)
            // Continue without embedded subtitle
        } else {
            result.VideoWithSubPath = filepath.Join(downloadPath, "video_final.mp4")
        }
    }
    
    // Extract audio
    d.logger.Info("Extracting audio from %s", url)
    err = utils.RetryWithContext(ctx, func() error {
        return d.extractAudio(ctx, url, downloadPath)
    }, d.retryOpts)
    
    if err != nil {
        d.logger.Warn("Failed to extract audio after %d retries: %v", d.retryOpts.MaxRetries, err)
        // Continue without audio
    } else {
        result.AudioPath = filepath.Join(downloadPath, "audio.mp3")
    }
    
    // Get video duration
    result.Duration = d.getVideoDuration(result.VideoPath)
    
    // If thumbnail wasn't downloaded, extract it from the video
    if result.ThumbnailPath == "" && result.VideoPath != "" {
        d.logger.Info("Extracting high-resolution PNG thumbnail from video")
        err := d.extractThumbnail(ctx, result.VideoPath, downloadPath)
        if err != nil {
            d.logger.Warn("Failed to extract thumbnail from video: %v", err)
        } else {
            thumbnailPath := filepath.Join(downloadPath, "thumbnail.png")
            if fileExists(thumbnailPath) {
                result.ThumbnailPath = thumbnailPath
                d.logger.Info("Successfully extracted high-resolution PNG thumbnail from video to %s", thumbnailPath)
            }
        }
    }
    
    return result, nil
}

// downloadThumbnail downloads the thumbnail for the video
func (d *VideoDownloader) downloadThumbnail(ctx context.Context, url string, downloadPath string) error {
    ytDlpPath := d.dependencyPaths["yt-dlp"]
    if ytDlpPath == "" {
        return errors.New("yt-dlp executable path not found")
    }

	args := d.getCookiesArgs(url)
	args = append(args,
		"--skip-download",
		"--write-thumbnail",
		"--convert-thumbnails", "png",
		"--write-all-thumbnails",
		"-v", // Add verbose output
		"--print-traffic", // This will show network requests, helpful for debugging
		"-o", filepath.Join(downloadPath, "thumbnail"),
		url,
	)

    cmd := exec.CommandContext(ctx, ytDlpPath, args...)
   output, err := cmd.CombinedOutput()

   if err != nil {
    d.logger.Error("Thumbnail download failed: %v, output: %s", err, string(output))
    return fmt.Errorf("thumbnail download failed: %w", err)
  }

	// Find all downloaded thumbnails
	files, err := filepath.Glob(filepath.Join(downloadPath, "thumbnail*.png"))
	if err != nil || len(files) == 0 {
		return fmt.Errorf("no thumbnail found after download")
	}

	// Sort thumbnails by file size to find the highest resolution one
	var largestThumbnail string
	var largestSize int64

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.Size() > largestSize {
			largestSize = info.Size()
			largestThumbnail = file
		}
	}

	// Use the largest thumbnail found
	if largestThumbnail != "" {
		newPath := filepath.Join(downloadPath, "thumbnail.png")

		if largestThumbnail != newPath {
			if err := os.Rename(largestThumbnail, newPath); err != nil {
				d.logger.Warn("Failed to rename thumbnail: %v", err)
				// Continue with the original path
			}
		}

		// Remove other thumbnails to save space
		for _, file := range files {
			if file != largestThumbnail && file != newPath {
				os.Remove(file)
			}
		}
	}

	return nil
}

// extractThumbnail extracts a thumbnail from the video file
func (d *VideoDownloader) extractThumbnail(ctx context.Context, videoPath string, downloadPath string) error {
    ffmpegPath := d.dependencyPaths["ffmpeg"]
    if ffmpegPath == "" {
        return errors.New("ffmpeg executable path not found")
    }

    thumbnailPath := filepath.Join(downloadPath, "thumbnail.png")
    
    args := []string{
        "-i", videoPath,
        "-ss", "00:00:01", // Take frame at 1 second
        "-vframes", "1",
        "-q:v", "1", // Highest quality (1-31, lower is better)
        "-vf", "scale=1920:-1", // Scale to 1920px width, maintain aspect ratio
        thumbnailPath,
    }
    
    cmd := exec.CommandContext(ctx, ffmpegPath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        d.logger.Error("Thumbnail extraction failed: %v, output: %s", err, string(output))
        return fmt.Errorf("thumbnail extraction failed: %w", err)
    }
    
    return nil
}

// downloadPrimaryVideo downloads the best video + best audio merged
func (d *VideoDownloader) downloadPrimaryVideo(ctx context.Context, url string, downloadPath string) error {
    ytDlpPath := d.dependencyPaths["yt-dlp"]
    aria2cPath := d.dependencyPaths["aria2c"]
    if ytDlpPath == "" || aria2cPath == "" {
        return errors.New("yt-dlp or aria2c executable path not found")
    }

	args := d.getCookiesArgs(url)
	args = append(args,
		"-f", "bv*[vcodec^=avc]+ba/best[ext=mp4][vcodec^=avc]",
		"--merge-output-format", "mp4",
		"--external-downloader", aria2cPath, // Use the stored path
		"--external-downloader-args", "-x 16 -s 16 -k 1M --async-dns=false --async-dns-server=8.8.8.8,1.1.1.1",
		"-o", filepath.Join(downloadPath, "video_base.mp4"),
		url,
	)

	cmd := exec.CommandContext(ctx, ytDlpPath, args...) // Use the stored path
	output, err := cmd.CombinedOutput()

	if err != nil {
		d.logger.Warn("aria2c download failed, trying direct download: %v, output: %s", err, string(output))

		// Try direct download without aria2c
		directArgs := d.getCookiesArgs(url)
		directArgs = append(directArgs,
			"-f", "bv*[vcodec^=avc]+ba/best[ext=mp4][vcodec^=avc]",
			"--merge-output-format", "mp4",
			"-o", filepath.Join(downloadPath, "video_base.mp4"),
			url,
		)

		directCmd := exec.CommandContext(ctx, ytDlpPath, directArgs...) // Use the stored path
		directOutput, directErr := directCmd.CombinedOutput()

		if directErr != nil {
			d.logger.Error("Direct download also failed: %v, output: %s", directErr, string(directOutput))
			return fmt.Errorf("video download failed with both aria2c and direct methods: %w", directErr)
		}
	}

	return nil
}

// downloadSubtitle downloads the subtitle in the specified language
func (d *VideoDownloader) downloadSubtitle(ctx context.Context, url string, lang string, downloadPath string) (string, error) {
    ytDlpPath := d.dependencyPaths["yt-dlp"]
    if ytDlpPath == "" {
        return "", errors.New("yt-dlp executable path not found")
    }

    // First, check available subtitles
    availableSubs, err := d.listAvailableSubtitles(ctx, url)
    if err != nil {
        d.logger.Warn("Failed to list available subtitles: %v", err)
        // Continue with download attempt anyway
    } else {
        d.logger.Info("Available subtitles: %s", availableSubs)
    }

    // Improved subtitle download arguments
    args := d.getCookiesArgs(url)
    args = append(args,
	"--skip-download",
	"--write-subs",
	"--write-auto-sub",
	"--sub-lang", lang,
	"--sub-format", "srt/vtt",
	"-o", filepath.Join(downloadPath, "subtitle.%(language)s.%(ext)s"),
	url,
    )
    
    // Don't use aria2c for subtitle downloads - it's unnecessary and can cause issues
    cmd := exec.CommandContext(ctx, ytDlpPath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        d.logger.Error("Subtitle download failed: %v, output: %s", err, string(output))
        return "", fmt.Errorf("subtitle download failed: %w", err)
    }
    
    // Check if subtitle was downloaded
    outputStr := string(output)
    if strings.Contains(outputStr, "There are no subtitles") || 
       strings.Contains(outputStr, "Subtitle not available") {
        d.logger.Info("No subtitles available in language %s", lang)
        return "", nil
    }
    
    // Look for subtitle files with more flexible patterns
    // First try the expected language-specific pattern
    subtitlePatterns := []string{
        filepath.Join(downloadPath, fmt.Sprintf("subtitle.%s.srt", lang)),
        filepath.Join(downloadPath, fmt.Sprintf("subtitle.%s.vtt", lang)),
        filepath.Join(downloadPath, "subtitle.srt"),
        filepath.Join(downloadPath, "subtitle.vtt"),
    }
    
    // Also check for auto-generated subtitles
    autoSubPatterns := []string{
        filepath.Join(downloadPath, fmt.Sprintf("subtitle.%s.auto.srt", lang)),
        filepath.Join(downloadPath, fmt.Sprintf("subtitle.%s.auto.vtt", lang)),
    }
    
    // Combine all patterns
    allPatterns := append(subtitlePatterns, autoSubPatterns...)
    
    // Try to find any matching subtitle file
    for _, pattern := range allPatterns {
        if fileExists(pattern) {
            d.logger.Info("Successfully found subtitle at %s", pattern)
            return pattern, nil
        }
    }
    
    // If we still haven't found anything, try a more general glob search
    files, err := filepath.Glob(filepath.Join(downloadPath, "subtitle.*"))
    if err == nil && len(files) > 0 {
        d.logger.Info("Found subtitle using glob search: %s", files[0])
        return files[0], nil
    }
    
    d.logger.Warn("Subtitle file not found despite successful download")
    return "", fmt.Errorf("subtitle file not found")
}

// listAvailableSubtitles lists available subtitles for a video
func (d *VideoDownloader) listAvailableSubtitles(ctx context.Context, url string) (string, error) {
    ytDlpPath := d.dependencyPaths["yt-dlp"]
    if ytDlpPath == "" {
        return "", errors.New("yt-dlp executable path not found")
    }

    args := []string{
        "--list-subs",
        url,
    }
    
    cmd := exec.CommandContext(ctx, ytDlpPath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        return "", fmt.Errorf("failed to list subtitles: %w", err)
    }
    
    return string(output), nil
}

// embedSubtitle embeds the subtitle into the video
func (d *VideoDownloader) embedSubtitle(ctx context.Context, videoPath string, subtitlePath string, downloadPath string) error {
    ffmpegPath := d.dependencyPaths["ffmpeg"]
    if ffmpegPath == "" {
        return errors.New("ffmpeg executable path not found")
    }

    outputPath := filepath.Join(downloadPath, "video_final.mp4")
    
    args := []string{
        "-i", videoPath,
        "-vf", fmt.Sprintf("subtitles=%s", subtitlePath),
        "-c:a", "copy",
        outputPath,
    }
    
    cmd := exec.CommandContext(ctx, ffmpegPath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        d.logger.Error("Subtitle embedding failed: %v, output: %s", err, string(output))
        return fmt.Errorf("subtitle embedding failed: %w", err)
    }
    
    d.logger.Info("Successfully embedded subtitle into video at %s", outputPath)
    return nil
}

// extractAudio extracts the audio from the video
func (d *VideoDownloader) extractAudio(ctx context.Context, url string, downloadPath string) error {
    ytDlpPath := d.dependencyPaths["yt-dlp"]
    if ytDlpPath == "" {
        return errors.New("yt-dlp executable path not found")
    }

    args := d.getCookiesArgs(url)
    args = append(args,
	"-f", "ba",
	"--extract-audio",
	"--audio-format", "mp3",
	"-o", filepath.Join(downloadPath, "audio.%(ext)s"),
	url,
     )
    
    cmd := exec.CommandContext(ctx, ytDlpPath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        d.logger.Error("Audio extraction failed: %v, output: %s", err, string(output))
        return fmt.Errorf("audio extraction failed: %w", err)
    }
    
    d.logger.Info("Successfully extracted audio to %s", filepath.Join(downloadPath, "audio.mp3"))
    return nil
}

// getVideoDuration gets the duration of a video in seconds
func (d *VideoDownloader) getVideoDuration(videoPath string) int {
    ffprobePath := d.dependencyPaths["ffprobe"] // Use ffprobe
    if ffprobePath == "" {
        d.logger.Warn("ffprobe executable path not found, cannot get video duration.")
        return 0
    }

    args := []string{
        "-v", "error",
        "-show_entries", "format=duration",
        "-of", "default=noprint_wrappers=1:nokey=1",
        videoPath,
    }
    
    cmd := exec.Command(ffprobePath, args...) // Use the stored path
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        d.logger.Warn("Failed to get video duration: %v", err)
        return 0
    }
    
    // Parse duration
    durationStr := strings.TrimSpace(string(output))
    var duration float64
    _, err = fmt.Sscanf(durationStr, "%f", &duration)
    if err != nil {
        d.logger.Warn("Failed to parse video duration: %v", err)
        return 0
    }
    
    return int(duration)
}

func (d *VideoDownloader) CleanupDownloads(maxAge time.Duration) error {
    entries, err := os.ReadDir(d.downloadDir)
    if err != nil {
        return fmt.Errorf("failed to read download directory: %w", err)
    }

    var cleanupErrors []error
    cutoffTime := time.Now().Add(-maxAge)

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        dirPath := filepath.Join(d.downloadDir, entry.Name())
        dirInfo, err := entry.Info()
        if err != nil {
            cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to get info for %s: %w", dirPath, err))
            continue
        }

        if dirInfo.ModTime().Before(cutoffTime) {
            // Check if directory is empty (optional safety check)
            if isEmpty, err := isDirEmpty(dirPath); err == nil && isEmpty {
                if err := os.RemoveAll(dirPath); err != nil {
                    cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to remove %s: %w", dirPath, err))
                    d.logger.Error("Failed to remove old download directory %s: %v", dirPath, err)
                } else {
                    d.logger.Debug("Removed old download directory %s", dirPath)
                }
            } else if err != nil {
                cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to check if %s is empty: %w", dirPath, err))
            }
            // Skip non-empty directories to avoid deleting active downloads
        }
    }

    if len(cleanupErrors) > 0 {
        return fmt.Errorf("encountered %d errors during cleanup: %v", len(cleanupErrors), errors.Join(cleanupErrors...))
    }
    return nil
}

// Helper function to check if directory is empty
func isDirEmpty(dirPath string) (bool, error) {
    f, err := os.Open(dirPath)
    if err != nil {
        return false, err
    }
    defer f.Close()

    _, err = f.Readdirnames(1)
    if err == io.EOF {
        return true, nil
    }
    return false, err
}

