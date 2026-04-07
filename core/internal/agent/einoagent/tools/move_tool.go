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

type moveFileTool struct {
	move FileMover
}

type moveFileInput struct {
	FileIdentity         string `json:"file_identity"`
	TargetFolderIdentity string `json:"target_folder_identity,omitempty"`
	DesiredName          string `json:"desired_name,omitempty"`
}

func NewMoveFileTool(move FileMover) einotool.BaseTool {
	return &moveFileTool{move: move}
}

func (t *moveFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: MoveFileToolName,
		Desc: "Move a file or folder into another folder in the current user's cloud_disk workspace.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_identity":          {Type: schema.String, Desc: "The identity of the file or folder to move.", Required: true},
			"target_folder_identity": {Type: schema.String, Desc: "Optional target folder identity. Leave empty to move to the root folder."},
			"desired_name":           {Type: schema.String, Desc: "Optional name to use after moving. Leave empty to keep the current name."},
		}),
	}, nil
}

func (t *moveFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	if t.move == nil {
		return "", errors.New("move file capability is not configured")
	}
	var input moveFileInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.FileIdentity) == "" {
		return "", errors.New("file_identity is required")
	}
	if err := t.move(ctx, strings.TrimSpace(input.FileIdentity), strings.TrimSpace(input.TargetFolderIdentity), strings.TrimSpace(input.DesiredName)); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.TargetFolderIdentity) == "" {
		return fmt.Sprintf("移动成功：%s -> 根目录", input.FileIdentity), nil
	}
	return fmt.Sprintf("移动成功：%s -> %s", input.FileIdentity, input.TargetFolderIdentity), nil
}
