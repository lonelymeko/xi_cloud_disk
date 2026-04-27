package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

type documentAnswerTool struct{}

type documentAnswerInput struct {
	Question string    `json:"question"`
	Files    []FileRef `json:"files"`
}

func NewDocumentAnswerTool(_ any) einotool.BaseTool {
	return &documentAnswerTool{}
}

func (t *documentAnswerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: FileAnswerToolName,
		Desc: "Read one or more cloud_disk file URLs, extract textual content from supported text-like files or PDFs, and return the relevant source material for the agent to answer.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {Type: schema.String, Desc: "The user question to answer from the attached files.", Required: true},
			"files":    FileArrayParam("The attached cloud_disk files to inspect."),
		}),
	}, nil
}

func (t *documentAnswerTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	start := time.Now()
	var input documentAnswerInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.Question) == "" {
		return "", errors.New("question is required")
	}
	if len(input.Files) == 0 {
		return "", errors.New("files are required")
	}

	excerpts := make([]FileExcerpt, 0, len(input.Files))
	for _, file := range input.Files {
		fileStart := time.Now()
		content, err := FetchDocumentText(ctx, file)
		if err != nil {
			logx.Errorf("document tool stage=fetch_failed file=%s duration=%s err=%v", file.Name, time.Since(fileStart), err)
			return "", err
		}
		logx.Infof("document tool stage=fetch_completed file=%s duration=%s content_len=%d", file.Name, time.Since(fileStart), len(strings.TrimSpace(content)))
		excerpts = append(excerpts, FileExcerpt{Name: file.Name, URL: file.URL, Content: content})
	}

	result := buildDocumentContext(input.Question, excerpts)
	logx.Infof("document tool stage=context_completed files=%d total_duration=%s reply_len=%d", len(input.Files), time.Since(start), len(strings.TrimSpace(result)))
	return result, nil
}

func buildDocumentContext(question string, excerpts []FileExcerpt) string {
	var sb strings.Builder
	sb.WriteString("Document source material for the agent.\n")
	sb.WriteString("Use only the following extracted file contents when answering the user.\n\n")
	sb.WriteString("User question:\n")
	sb.WriteString(question)
	sb.WriteString("\n\nFiles:\n")
	for i, item := range excerpts {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, item.Name))
		sb.WriteString(trimToolContent(item.Content, 6000))
		sb.WriteString("\n\n")
	}
	return sb.String()
}
