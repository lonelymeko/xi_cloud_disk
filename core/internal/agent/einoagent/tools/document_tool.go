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

type documentAnswerTool struct {
	cm einomodel.BaseChatModel
}

type documentAnswerInput struct {
	Question string    `json:"question"`
	Files    []FileRef `json:"files"`
}

func NewDocumentAnswerTool(cm einomodel.BaseChatModel) einotool.BaseTool {
	return &documentAnswerTool{cm: cm}
}

func (t *documentAnswerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: FileAnswerToolName,
		Desc: "Read one or more cloud_disk file URLs, extract textual content from supported text-like files or PDFs, and answer the user's question with citations.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"question": {Type: schema.String, Desc: "The user question to answer from the attached files.", Required: true},
			"files":    FileArrayParam("The attached cloud_disk files to inspect."),
		}),
	}, nil
}

func (t *documentAnswerTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
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
		content, err := FetchDocumentText(ctx, file)
		if err != nil {
			return "", err
		}
		excerpts = append(excerpts, FileExcerpt{Name: file.Name, URL: file.URL, Content: content})
	}

	resp, err := t.cm.Generate(ctx, []*schema.Message{schema.UserMessage(buildDocumentPrompt(input.Question, excerpts))})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func buildDocumentPrompt(question string, excerpts []FileExcerpt) string {
	var sb strings.Builder
	sb.WriteString("You are a file analysis assistant. Answer using only the provided file contents.\n")
	sb.WriteString("Requirements:\n1. Reply in Chinese.\n2. If the answer cannot be determined, say so clearly.\n3. Quote the source file names when making claims.\n\n")
	sb.WriteString("Question:\n")
	sb.WriteString(question)
	sb.WriteString("\n\nFiles:\n")
	for i, item := range excerpts {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, item.Name))
		sb.WriteString(item.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}
