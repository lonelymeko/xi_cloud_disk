package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	openaiext "github.com/cloudwego/eino-ext/components/model/openai"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

const defaultVideoModelTimeout = 60 * time.Second

const defaultASRChunkSeconds = 300

type videoSummaryTool struct{}

type videoSummaryInput struct {
	Question string  `json:"question"`
	File     FileRef `json:"file"`
}

func NewVideoSummaryTool(_ any) einotool.BaseTool {
	return &videoSummaryTool{}
}

func (t *videoSummaryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: VideoSummaryToolName,
		Desc: "Download one attached video, extract key frame timestamps and the full audio transcript, and return source material for the agent to answer.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {Type: schema.String, Desc: "The user question about the video.", Required: true},
			"file": {
				Type:     schema.Object,
				Desc:     "The attached video file to inspect.",
				Required: true,
				SubParams: map[string]*schema.ParameterInfo{
					"name":      {Type: schema.String, Desc: "Display name of the file.", Required: true},
					"url":       {Type: schema.String, Desc: "Temporary signed URL of the file.", Required: true},
					"mime_type": {Type: schema.String, Desc: "MIME type of the file if known."},
				},
			},
		}),
	}, nil
}

func (t *videoSummaryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	start := time.Now()
	var input videoSummaryInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.Question) == "" {
		return "", errors.New("question is required")
	}
	if strings.TrimSpace(input.File.URL) == "" {
		return "", errors.New("file.url is required")
	}
	if !IsVideoLike(input.File.MIMEType, input.File.Name) {
		return "", fmt.Errorf("file %s is not a supported video format", input.File.Name)
	}

	frames, audio, cleanup, err := ExtractVideoMedia(ctx, input.File)
	if err != nil {
		logx.Errorf("video tool stage=extract_media_failed file=%s duration=%s err=%v", input.File.Name, time.Since(start), err)
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if len(frames) == 0 {
		return "", errors.New("no key frames extracted from video")
	}

	transcript, err := transcribeVideoAudio(ctx, audio)
	if err != nil {
		logx.Errorf("video tool stage=transcribe_failed file=%s duration=%s err=%v", input.File.Name, time.Since(start), err)
		return "", err
	}
	logx.Infof("video tool stage=transcribe_completed file=%s duration=%s transcript_len=%d", input.File.Name, time.Since(start), len(strings.TrimSpace(transcript)))

	result := buildVideoContext(input.Question, input.File, frames, transcript)
	logx.Infof("video tool stage=context_completed file=%s frames=%d duration=%s reply_len=%d", input.File.Name, len(frames), time.Since(start), len(strings.TrimSpace(result)))
	return result, nil
}

func buildVideoContext(question string, file FileRef, frames []ExtractedFrame, transcript string) string {
	var sb strings.Builder
	sb.WriteString("Video source material for the agent.\n")
	sb.WriteString("Use the following extracted frame timestamps and audio transcript when answering the user.\n\n")
	sb.WriteString("User question:\n")
	sb.WriteString(question)
	sb.WriteString("\n\n视频文件：")
	sb.WriteString(file.Name)
	sb.WriteString("\n关键帧时间点：\n")
	for _, frame := range frames {
		sb.WriteString(fmt.Sprintf("- 第 %d 帧，时间 %s\n", frame.Index, formatVideoTimestamp(frame.TimestampSec)))
	}
	sb.WriteString("\n音频转写：\n")
	if strings.TrimSpace(transcript) == "" {
		sb.WriteString("未提取到可用音频，或音频转写为空。\n")
	} else {
		sb.WriteString(trimToolContent(transcript, 12000))
		sb.WriteString("\n")
	}
	return sb.String()
}

func transcribeVideoAudio(ctx context.Context, audio *ExtractedAudio) (string, error) {
	start := time.Now()
	if audio == nil || strings.TrimSpace(audio.Base64Data) == "" {
		return "", nil
	}

	cfg := loadVideoASRChatModelConfig()
	transcript, err := callQwenASR(ctx, cfg, audio.MIMEType, audio.Base64Data)
	if err == nil {
		logx.Infof("video asr stage=single_request_completed duration=%s transcript_len=%d", time.Since(start), len(strings.TrimSpace(transcript)))
		return strings.TrimSpace(transcript), nil
	}
	if !isAudioTooLongError(err) {
		return "", fmt.Errorf("transcribe video audio: %w", err)
	}

	transcript, err = transcribeLongVideoAudio(ctx, cfg, audio)
	if err != nil {
		return "", fmt.Errorf("transcribe video audio: %w", err)
	}
	logx.Infof("video asr stage=chunked_request_completed duration=%s transcript_len=%d", time.Since(start), len(strings.TrimSpace(transcript)))
	return strings.TrimSpace(transcript), nil
}

func loadVideoASRChatModelConfig() openaiext.ChatModelConfig {
	return openaiext.ChatModelConfig{
		APIKey:  firstNonEmptyEnv("ASR_API_KEY", "VIDEO_ASR_API_KEY", "DASHSCOPE_API_KEY", "OPENAI_API_KEY"),
		Model:   firstNonEmptyEnv("ASR_MODEL", "VIDEO_ASR_MODEL", "QWEN_ASR_MODEL", "OPENAI_MODEL"),
		BaseURL: firstNonEmptyEnv("ASR_BASE_URL", "VIDEO_ASR_BASE_URL", "DASHSCOPE_BASE_URL", "OPENAI_BASE_URL"),
		Timeout: defaultVideoModelTimeout,
	}
}

func callQwenASR(ctx context.Context, cfg openaiext.ChatModelConfig, audioMIMEType, audioBase64 string) (string, error) {
	start := time.Now()
	if err := validateChatModelConfig("Qwen-ASR", cfg); err != nil {
		return "", err
	}

	endpoint, err := resolveChatCompletionsEndpoint(cfg.BaseURL)
	if err != nil {
		return "", err
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", audioMIMEType, audioBase64)
	payload := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_audio",
						"input_audio": map[string]any{
							"data": dataURI,
						},
					},
				},
			},
		},
		"stream": false,
		"asr_options": map[string]any{
			"enable_itn": false,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal ASR request: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("build ASR request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send ASR request: %w", err)
	}
	defer resp.Body.Close()
	logx.Infof("video asr stage=http_completed duration=%s", time.Since(start))

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ASR response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("error, status code: %d, status: %s, message: %s", resp.StatusCode, resp.Status, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode ASR response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("ASR response contained no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

func validateChatModelConfig(name string, cfg openaiext.ChatModelConfig) error {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" || strings.TrimSpace(cfg.BaseURL) == "" {
		return fmt.Errorf("%s chat model config is empty; set the related environment variables before using %s", name, VideoSummaryToolName)
	}
	return nil
}

func resolveChatCompletionsEndpoint(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("Qwen-ASR base URL is empty")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse ASR base URL: %w", err)
	}
	if strings.HasSuffix(u.Path, "/chat/completions") {
		return u.String(), nil
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/chat/completions"
	return u.String(), nil
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func formatVideoTimestamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	totalSeconds := int(sec)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func isAudioTooLongError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "audio is too long")
}

func transcribeLongVideoAudio(ctx context.Context, cfg openaiext.ChatModelConfig, audio *ExtractedAudio) (string, error) {
	if audio == nil || strings.TrimSpace(audio.Path) == "" {
		return "", errors.New("audio path is required for long-audio transcription")
	}
	if err := RequireBinary("ffmpeg"); err != nil {
		return "", err
	}

	chunksDir, err := os.MkdirTemp("", "cloud-disk-audio-chunks-*")
	if err != nil {
		return "", fmt.Errorf("create audio chunks dir: %w", err)
	}
	defer os.RemoveAll(chunksDir)

	pattern := filepath.Join(chunksDir, "chunk-%03d.mp3")
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", audio.Path,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", defaultASRChunkSeconds),
		"-c", "copy",
		pattern,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("split audio into chunks: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	chunkPaths, err := filepath.Glob(filepath.Join(chunksDir, "chunk-*.mp3"))
	if err != nil {
		return "", fmt.Errorf("list audio chunks: %w", err)
	}
	if len(chunkPaths) == 0 {
		return "", errors.New("no audio chunks generated")
	}

	parts := make([]string, 0, len(chunkPaths))
	for _, chunkPath := range chunkPaths {
		audioBytes, err := os.ReadFile(chunkPath)
		if err != nil {
			return "", fmt.Errorf("read audio chunk: %w", err)
		}
		chunkTranscript, err := callQwenASR(ctx, cfg, audio.MIMEType, base64.StdEncoding.EncodeToString(audioBytes))
		if err != nil {
			return "", fmt.Errorf("transcribe audio chunk %s: %w", filepath.Base(chunkPath), err)
		}
		chunkTranscript = strings.TrimSpace(chunkTranscript)
		if chunkTranscript != "" {
			parts = append(parts, chunkTranscript)
		}
	}

	return strings.Join(parts, "\n"), nil
}
