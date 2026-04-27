package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

type imageAnalysisTool struct {
	cm einomodel.BaseChatModel
}

type imageAnalysisInput struct {
	Question string    `json:"question"`
	Files    []FileRef `json:"files"`
}

func NewImageAnalysisTool(cm einomodel.BaseChatModel) einotool.BaseTool {
	return &imageAnalysisTool{cm: cm}
}

func (t *imageAnalysisTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ImageAnalysisToolName,
		Desc: "Analyze one or more attached image URLs with a multimodal model and answer the user's question in Chinese.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {Type: schema.String, Desc: "The user question about the images.", Required: true},
			"files":    FileArrayParam("One or more image files to analyze."),
		}),
	}, nil
}

func (t *imageAnalysisTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	start := time.Now()
	var input imageAnalysisInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.Question) == "" {
		return "", errors.New("question is required")
	}
	if len(input.Files) == 0 {
		return "", errors.New("files are required")
	}

	parts := make([]schema.MessageInputPart, 0, len(input.Files)+1)
	parts = append(parts, schema.MessageInputPart{
		Type: schema.ChatMessagePartTypeText,
		Text: buildImagePrompt(input.Question, input.Files),
	})
	for _, file := range input.Files {
		if !IsImageLike(file.MIMEType, file.Name) {
			return "", errors.New("only image files are supported")
		}
		fileURL := file.URL
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeImageURL,
			Image: &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					URL:      &fileURL,
					MIMEType: NormalizeMIME(file.MIMEType, file.Name),
				},
			},
		})
	}

	resp, err := t.cm.Generate(ctx, []*schema.Message{{
		Role:                  schema.User,
		UserInputMultiContent: parts,
	}})
	if err != nil {
		logx.Errorf("image tool stage=model_failed files=%d duration=%s err=%v", len(input.Files), time.Since(start), err)
		return "", err
	}
	logx.Infof("image tool stage=model_completed files=%d duration=%s reply_len=%d", len(input.Files), time.Since(start), len(strings.TrimSpace(resp.Content)))
	return strings.TrimSpace(resp.Content), nil
}

func buildImagePrompt(question string, files []FileRef) string {
	var sb strings.Builder
	sb.WriteString("你是图片分析助手。请根据随附图片回答用户问题。\n")
	sb.WriteString("要求：\n1. 用中文回答。\n2. 明确区分你从哪一张图得出结论。\n3. 如果图片不足以支持结论，要直接说明。\n\n")
	sb.WriteString("用户问题：\n")
	sb.WriteString(question)
	sb.WriteString("\n\n图片列表：\n")
	for i, file := range files {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, file.Name))
	}
	return sb.String()
}
