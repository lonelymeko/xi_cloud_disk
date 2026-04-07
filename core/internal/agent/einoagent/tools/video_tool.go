package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type videoSummaryTool struct {
	cm einomodel.BaseChatModel
}

type videoSummaryInput struct {
	Question string  `json:"question"`
	File     FileRef `json:"file"`
}

func NewVideoSummaryTool(cm einomodel.BaseChatModel) einotool.BaseTool {
	return &videoSummaryTool{cm: cm}
}

func (t *videoSummaryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: VideoSummaryToolName,
		Desc: "Use a dedicated video explanation tool to download one attached video, extract key frames, and answer the user's question in Chinese.",
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

	frames, cleanup, err := ExtractVideoFrames(ctx, input.File)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if len(frames) == 0 {
		return "", errors.New("no key frames extracted from video")
	}

	parts := make([]schema.MessageInputPart, 0, len(frames)+1)
	parts = append(parts, schema.MessageInputPart{
		Type: schema.ChatMessagePartTypeText,
		Text: buildVideoPrompt(input.Question, input.File, frames),
	})
	for _, frame := range frames {
		frameBase64 := frame.Base64Data
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &frameBase64,
					MIMEType:   frame.MIMEType,
				},
			},
		})
	}

	resp, err := t.cm.Generate(ctx, []*schema.Message{{
		Role:                  schema.User,
		UserInputMultiContent: parts,
	}})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func buildVideoPrompt(question string, file FileRef, frames []ExtractedFrame) string {
	var sb strings.Builder
	sb.WriteString("你是视频解释助手。当前由专门的视频工具提供关键帧，不要假设未看到的内容。\n")
	sb.WriteString("请根据关键帧解释视频内容，并结合用户问题作答。\n")
	sb.WriteString("要求：\n1. 用中文回答。\n2. 先概括视频主题，再回答问题。\n3. 对不确定的时序或细节明确说明“仅能从关键帧推断”。\n\n")
	sb.WriteString("用户问题：\n")
	sb.WriteString(question)
	sb.WriteString("\n\n视频文件：")
	sb.WriteString(file.Name)
	sb.WriteString("\n关键帧时间点：\n")
	for _, frame := range frames {
		sb.WriteString(fmt.Sprintf("- 第 %d 帧，时间 %.2f 秒\n", frame.Index, frame.TimestampSec))
	}
	return sb.String()
}
