package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

const (
	FileAnswerToolName      = "answer_from_file_urls"
	ImageAnalysisToolName   = "analyze_images_from_urls"
	VideoSummaryToolName    = "summarize_video_from_url"
	ListFilesToolName       = "list_user_files"
	CreateFolderToolName    = "create_folder"
	MoveFileToolName        = "move_file"
	DefaultAttachmentBytes  = 2 << 20
	DefaultVideoFrameCount  = 6
	DefaultVideoHTTPTimeout = 10 * time.Minute
)

type FileRef struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	MIMEType string `json:"mime_type,omitempty"`
}

type FolderCreateResult struct {
	ID       int64  `json:"id"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
	ParentID int64  `json:"parent_id"`
}

type FolderCreator func(ctx context.Context, parentFolderIdentity, name string) (FolderCreateResult, error)
type FileMover func(ctx context.Context, fileIdentity, targetFolderIdentity, desiredName string) error
type FileLister func(ctx context.Context, folderIdentity string, page, size int) ([]ListedFile, int64, error)

type FileExcerpt struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type ListedFile struct {
	ID                 int64  `json:"id"`
	Identity           string `json:"identity"`
	Name               string `json:"name"`
	Ext                string `json:"ext,omitempty"`
	Size               int64  `json:"size,omitempty"`
	RepositoryIdentity string `json:"repository_identity,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	IsFolder           bool   `json:"is_folder"`
}

type ExtractedFrame struct {
	Index        int
	TimestampSec float64
	Base64Data   string
	MIMEType     string
}

func FileArrayParam(desc string) *schema.ParameterInfo {
	return &schema.ParameterInfo{
		Type:     schema.Array,
		Desc:     desc,
		Required: true,
		ElemInfo: &schema.ParameterInfo{
			Type: schema.Object,
			SubParams: map[string]*schema.ParameterInfo{
				"name": {
					Type:     schema.String,
					Desc:     "Display name of the file.",
					Required: true,
				},
				"url": {
					Type:     schema.String,
					Desc:     "Temporary signed URL of the file.",
					Required: true,
				},
				"mime_type": {
					Type: schema.String,
					Desc: "MIME type of the file if known.",
				},
			},
		},
	}
}

func NormalizeMIME(mimeType, fileName string) string {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if mimeType != "" {
		return mimeType
	}
	return strings.ToLower(strings.TrimSpace(mime.TypeByExtension(filepath.Ext(fileName))))
}

func IsTextLike(mimeType, fileName string) bool {
	mimeType = NormalizeMIME(mimeType, fileName)
	if strings.HasPrefix(mimeType, "text/") || mimeType == "application/json" || mimeType == "application/xml" {
		return true
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".txt", ".md", ".markdown", ".json", ".csv", ".log", ".yaml", ".yml", ".xml":
		return true
	default:
		return false
	}
}

func IsImageLike(mimeType, fileName string) bool {
	mimeType = NormalizeMIME(mimeType, fileName)
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func IsVideoLike(mimeType, fileName string) bool {
	mimeType = NormalizeMIME(mimeType, fileName)
	if strings.HasPrefix(mimeType, "video/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v":
		return true
	default:
		return false
	}
}

func FetchTextFile(ctx context.Context, file FileRef) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", file.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download %s: unexpected status %s", file.Name, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, DefaultAttachmentBytes))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file.Name, err)
	}

	mimeType := file.MIMEType
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
	}
	if !IsTextLike(mimeType, file.Name) {
		return "", fmt.Errorf("file %s is not a supported text-like format yet", file.Name)
	}
	return string(body), nil
}

func ExtractVideoFrames(ctx context.Context, file FileRef) ([]ExtractedFrame, func(), error) {
	if err := RequireBinary("ffmpeg"); err != nil {
		return nil, nil, err
	}
	if err := RequireBinary("ffprobe"); err != nil {
		return nil, nil, err
	}

	workDir, err := os.MkdirTemp("", "cloud-disk-video-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }

	videoPath, err := DownloadVideoFile(ctx, workDir, file)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	durationSec, _ := ProbeDuration(ctx, videoPath)
	timestamps := SelectFrameTimestamps(durationSec, DefaultVideoFrameCount)
	if len(timestamps) == 0 {
		timestamps = []float64{1, 3, 5}
	}

	framesDir := filepath.Join(workDir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("create frames dir: %w", err)
	}

	frames := make([]ExtractedFrame, 0, len(timestamps))
	for i, ts := range timestamps {
		framePath := filepath.Join(framesDir, fmt.Sprintf("frame-%03d.jpg", i+1))
		if err := CaptureFrame(ctx, videoPath, ts, framePath); err != nil {
			cleanup()
			return nil, nil, err
		}
		frame, err := LoadFrame(i+1, ts, framePath)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		frames = append(frames, frame)
	}
	return frames, cleanup, nil
}

func DownloadVideoFile(ctx context.Context, workDir string, file FileRef) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: DefaultVideoHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", file.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download %s: unexpected status %s", file.Name, resp.Status)
	}

	ext := strings.ToLower(filepath.Ext(file.Name))
	if ext == "" {
		exts, _ := mime.ExtensionsByType(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
		if len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".mp4"
	}
	videoPath := filepath.Join(workDir, "source"+ext)
	dst, err := os.Create(videoPath)
	if err != nil {
		return "", fmt.Errorf("create video file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return "", fmt.Errorf("save video file: %w", err)
	}
	return videoPath, nil
}

func ProbeDuration(ctx context.Context, videoPath string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("probe video duration: %w", err)
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return 0, nil
	}
	var duration float64
	if _, err := fmt.Sscanf(raw, "%f", &duration); err != nil {
		return 0, nil
	}
	return duration, nil
}

func SelectFrameTimestamps(durationSec float64, maxFrames int) []float64 {
	if maxFrames <= 0 {
		return nil
	}
	if durationSec <= 0 {
		return []float64{1, 3, 5}
	}
	if durationSec < 3 {
		return []float64{0}
	}
	step := durationSec / float64(maxFrames+1)
	out := make([]float64, 0, maxFrames)
	for i := 1; i <= maxFrames; i++ {
		ts := step * float64(i)
		if ts >= durationSec {
			ts = durationSec - 0.5
		}
		if ts < 0 {
			ts = 0
		}
		out = append(out, ts)
	}
	return out
}

func CaptureFrame(ctx context.Context, videoPath string, timestampSec float64, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", timestampSec),
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("capture video frame: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func LoadFrame(index int, timestampSec float64, framePath string) (ExtractedFrame, error) {
	data, err := os.ReadFile(framePath)
	if err != nil {
		return ExtractedFrame{}, fmt.Errorf("read extracted frame: %w", err)
	}
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(framePath)))
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	return ExtractedFrame{
		Index:        index,
		TimestampSec: timestampSec,
		Base64Data:   base64.StdEncoding.EncodeToString(data),
		MIMEType:     mimeType,
	}, nil
}

func RequireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}
	return nil
}
