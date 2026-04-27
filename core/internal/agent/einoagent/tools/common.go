package tools

import (
	"bytes"
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
	pdf "github.com/ledongthuc/pdf"
	"github.com/zeromicro/go-zero/core/logx"
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
	DefaultVideoAudioMIME   = "audio/mpeg"
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

type ExtractedAudio struct {
	Path       string
	Base64Data string
	MIMEType   string
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

func IsPDFLike(mimeType, fileName string) bool {
	mimeType = NormalizeMIME(mimeType, fileName)
	if mimeType == "application/pdf" {
		return true
	}
	return strings.EqualFold(filepath.Ext(fileName), ".pdf")
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

func FetchDocumentText(ctx context.Context, file FileRef) (string, error) {
	start := time.Now()
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
	logx.Infof("document fetch stage=download_completed file=%s duration=%s bytes=%d", file.Name, time.Since(start), len(body))

	mimeType := file.MIMEType
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
	}
	switch {
	case IsTextLike(mimeType, file.Name):
		logx.Infof("document fetch stage=text_ready file=%s total_duration=%s", file.Name, time.Since(start))
		return string(body), nil
	case IsPDFLike(mimeType, file.Name):
		pdfStart := time.Now()
		content, err := ExtractPDFText(file.Name, body)
		if err != nil {
			return "", err
		}
		logx.Infof("document fetch stage=pdf_ready file=%s parse_duration=%s total_duration=%s content_len=%d", file.Name, time.Since(pdfStart), time.Since(start), len(strings.TrimSpace(content)))
		return content, nil
	default:
		return "", fmt.Errorf("file %s is not a supported document format yet", file.Name)
	}
}

func ExtractPDFText(fileName string, body []byte) (string, error) {
	start := time.Now()
	tmpFile, err := os.CreateTemp("", "cloud-disk-doc-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp pdf for %s: %w", fileName, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, bytes.NewReader(body)); err != nil {
		return "", fmt.Errorf("write temp pdf for %s: %w", fileName, err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp pdf for %s: %w", fileName, err)
	}
	logx.Infof("pdf extract stage=temp_written file=%s duration=%s bytes=%d", fileName, time.Since(start), len(body))

	openStart := time.Now()
	f, reader, err := pdf.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("open pdf %s: %w", fileName, err)
	}
	defer f.Close()
	logx.Infof("pdf extract stage=open_completed file=%s duration=%s", fileName, time.Since(openStart))

	plainTextStart := time.Now()
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text %s: %w", fileName, err)
	}

	plainText, err := io.ReadAll(io.LimitReader(textReader, 4*DefaultAttachmentBytes))
	if err != nil {
		return "", fmt.Errorf("read pdf text %s: %w", fileName, err)
	}
	logx.Infof("pdf extract stage=plain_text_read file=%s duration=%s text_bytes=%d", fileName, time.Since(plainTextStart), len(plainText))
	content := strings.TrimSpace(string(plainText))
	if content == "" {
		return "", fmt.Errorf("pdf %s does not contain extractable text; it may be scanned and require OCR", fileName)
	}
	logx.Infof("pdf extract stage=completed file=%s total_duration=%s content_len=%d", fileName, time.Since(start), len(content))
	return content, nil
}

func ExtractVideoFrames(ctx context.Context, file FileRef) ([]ExtractedFrame, func(), error) {
	frames, _, cleanup, err := ExtractVideoMedia(ctx, file)
	return frames, cleanup, err
}

func ExtractVideoMedia(ctx context.Context, file FileRef) ([]ExtractedFrame, *ExtractedAudio, func(), error) {
	start := time.Now()
	if err := RequireBinary("ffmpeg"); err != nil {
		return nil, nil, nil, err
	}
	if err := RequireBinary("ffprobe"); err != nil {
		return nil, nil, nil, err
	}

	workDir, err := os.MkdirTemp("", "cloud-disk-video-*")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }

	videoPath, err := DownloadVideoFile(ctx, workDir, file)
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	logx.Infof("video media stage=download_completed file=%s duration=%s path=%s", file.Name, time.Since(start), videoPath)

	probeStart := time.Now()
	durationSec, _ := ProbeDuration(ctx, videoPath)
	logx.Infof("video media stage=probe_completed file=%s duration=%s video_duration_sec=%.2f", file.Name, time.Since(probeStart), durationSec)
	timestamps := SelectFrameTimestamps(durationSec, DefaultVideoFrameCount)
	if len(timestamps) == 0 {
		timestamps = []float64{1, 3, 5}
	}

	framesDir := filepath.Join(workDir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("create frames dir: %w", err)
	}

	frames := make([]ExtractedFrame, 0, len(timestamps))
	for i, ts := range timestamps {
		frameStart := time.Now()
		framePath := filepath.Join(framesDir, fmt.Sprintf("frame-%03d.jpg", i+1))
		if err := CaptureFrame(ctx, videoPath, ts, framePath); err != nil {
			cleanup()
			return nil, nil, nil, err
		}
		frame, err := LoadFrame(i+1, ts, framePath)
		if err != nil {
			cleanup()
			return nil, nil, nil, err
		}
		frames = append(frames, frame)
		logx.Infof("video media stage=frame_completed file=%s frame=%d timestamp_sec=%.2f duration=%s", file.Name, i+1, ts, time.Since(frameStart))
	}

	audioStart := time.Now()
	audio, err := ExtractAudioIfExists(ctx, videoPath, workDir)
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	logx.Infof("video media stage=audio_completed file=%s duration=%s has_audio=%t total_duration=%s", file.Name, time.Since(audioStart), audio != nil, time.Since(start))
	return frames, audio, cleanup, nil
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

func ExtractAudioIfExists(ctx context.Context, videoPath, workDir string) (*ExtractedAudio, error) {
	hasAudio, err := HasAudioStream(ctx, videoPath)
	if err != nil {
		return nil, err
	}
	if !hasAudio {
		return nil, nil
	}

	audioPath := filepath.Join(workDir, "audio.mp3")
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", videoPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		audioPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("extract audio: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	data, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("read extracted audio: %w", err)
	}

	return &ExtractedAudio{
		Path:       audioPath,
		Base64Data: base64.StdEncoding.EncodeToString(data),
		MIMEType:   DefaultVideoAudioMIME,
	}, nil
}

func HasAudioStream(ctx context.Context, videoPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("probe audio stream: %w", err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func RequireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}
	return nil
}

func trimToolContent(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	return strings.TrimSpace(content[:maxLen]) + "\n...[truncated]"
}
