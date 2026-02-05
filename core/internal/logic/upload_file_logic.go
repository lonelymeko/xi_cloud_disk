// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"context"
	"encoding/json"
	"errors"

	"github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadFileLogic) UploadFile(req *types.UploadFileRequest, isExisted bool, repositoryIdentity string, localFilePath string, hash string) (resp *types.UploadFileResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	uploadEvent := &types.UploadEvent{
		UserIdentity:       userIdentity,
		ParentId:           req.ParentId,
		FilePath:           localFilePath,
		Ext:                req.Ext,
		Name:               req.Name,
		Size:               req.Size,
		IsExisted:          isExisted,
		RepositoryIdentity: repositoryIdentity,
		Hash:               hash,
	}
	body, err := json.Marshal(uploadEvent)
	if err != nil {
		return nil, err // 序列化失败直接返回，不用发 MQ
	}
	if err := l.PublishUploadEvent(body); err != nil {
		return nil, err
	}

	return &types.UploadFileResponse{
		Message: "文件上传开始",
	}, nil
}

func (l *UploadFileLogic) PublishUploadEvent(body []byte) error {
	ch, err := l.svcCtx.RabbitMQConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.Publish(
		common.ExchangeName, // 交换机名
		common.RoutingKey,   // 路由键（重要！不是队列名）
		false,               // mandatory
		false,               // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent, // 持久化消息
		},
	)
	if err != nil {
		logx.Errorf("发布上传事件失败: %v", err)
		return err
	}
	logx.Infof("已发布上传事件到 MQ: %s", string(body))
	return nil
}
