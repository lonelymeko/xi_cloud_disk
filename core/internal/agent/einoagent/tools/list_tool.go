package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type listFilesTool struct {
	list FileLister
}

type listFilesInput struct {
	FolderIdentity string `json:"folder_identity,omitempty"`
	Page           int    `json:"page,omitempty"`
	Size           int    `json:"size,omitempty"`
}

type listFilesOutput struct {
	FolderIdentity string       `json:"folder_identity,omitempty"`
	Page           int          `json:"page"`
	Size           int          `json:"size"`
	Count          int64        `json:"count"`
	List           []ListedFile `json:"list"`
}

func NewListFilesTool(list FileLister) einotool.BaseTool {
	return &listFilesTool{list: list}
}

func (t *listFilesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ListFilesToolName,
		Desc: "List the current user's cloud_disk files in the root folder or in a specified folder.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"folder_identity": {
				Type: schema.String,
				Desc: "Optional folder identity. Leave empty to list the root folder.",
			},
			"page": {
				Type: schema.Integer,
				Desc: "Optional page number. Defaults to 1.",
			},
			"size": {
				Type: schema.Integer,
				Desc: "Optional page size. Defaults to 50 and is capped by the server.",
			},
		}),
	}, nil
}

func (t *listFilesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	if t.list == nil {
		return "", errors.New("list files capability is not configured")
	}
	var input listFilesInput
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			return "", err
		}
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}
	size := input.Size
	if size <= 0 {
		size = 50
	}

	list, count, err := t.list(ctx, strings.TrimSpace(input.FolderIdentity), page, size)
	if err != nil {
		return "", err
	}
	out := listFilesOutput{
		FolderIdentity: strings.TrimSpace(input.FolderIdentity),
		Page:           page,
		Size:           size,
		Count:          count,
		List:           list,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("marshal file list: %w", err)
	}
	return string(raw), nil
}
