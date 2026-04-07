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

type createFolderTool struct {
	create FolderCreator
}

type createFolderInput struct {
	Name                 string `json:"name"`
	ParentFolderIdentity string `json:"parent_folder_identity,omitempty"`
}

func NewCreateFolderTool(create FolderCreator) einotool.BaseTool {
	return &createFolderTool{create: create}
}

func (t *createFolderTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: CreateFolderToolName,
		Desc: "Create a folder in the current user's cloud_disk workspace.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name":                   {Type: schema.String, Desc: "The folder name to create.", Required: true},
			"parent_folder_identity": {Type: schema.String, Desc: "Optional target parent folder identity. Leave empty to create in the root folder."},
		}),
	}, nil
}

func (t *createFolderTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	if t.create == nil {
		return "", errors.New("create folder capability is not configured")
	}
	var input createFolderInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.Name) == "" {
		return "", errors.New("name is required")
	}
	result, err := t.create(ctx, strings.TrimSpace(input.ParentFolderIdentity), strings.TrimSpace(input.Name))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("文件夹创建成功：name=%s identity=%s parent_id=%d", result.Name, result.Identity, result.ParentID), nil
}
