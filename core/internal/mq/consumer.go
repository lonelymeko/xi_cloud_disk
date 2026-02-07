package mq

import (
	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"encoding/json"
	"os"
	"path"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type Consumer struct {
	ctx     context.Context
	svcCtx  *svc.ServiceContext // 关键：持有 svcCtx
	channel *amqp.Channel
}

// 工厂方法：注入 svcCtx
func NewConsumer(ctx context.Context, svcCtx *svc.ServiceContext, ch *amqp.Channel) *Consumer {
	return &Consumer{
		ctx:     ctx,
		svcCtx:  svcCtx,
		channel: ch,
	}
}

func (c *Consumer) Start() {
	retryCount := 3
	// 2. 声明队列（确保队列存在）
	q, err := c.channel.QueueDeclare(common.QueueName, true, false, false, false, nil)
	if err != nil {
		logx.Errorf("声明队列失败: %v", err)
		return
	}

	// 3. 设置 QoS (关键！限制并发为 1)
	if qosErr := c.channel.Qos(1, 0, false); qosErr != nil {
		logx.Errorf("设置 QoS 失败: %v", qosErr)
		return
	}

	// 4. 注册消费者
	msgs, err := c.channel.Consume(
		q.Name, // 队列名
		"",     // 消费者标签
		false,  // 手动 Ack
		false,  // 非独占
		false,  // no-local
		false,  // no-wait
		nil,    // 额外参数
	)
	if err != nil {
		logx.Errorf("注册消费者失败: %v", err)
		return
	}

	logx.Info("MQ Consumer started, waiting for messages...")

	// 5. 阻塞消费 (开启 Goroutine 监听)
	go func() {
		for d := range msgs {
			logx.Infof("收到任务: %s", string(d.Body))

			// 重试逻辑
			var processErr error
			for i := 0; i < retryCount; i++ {
				processErr = c.processFile(d.Body)
				if processErr == nil {
					// 处理成功，确认消息
					if ackErr := d.Ack(false); ackErr != nil {
						logx.Errorf("确认消息失败: %v", ackErr)
					} else {
						logx.Info("任务处理成功并已确认")
					}
					break
				}

				// 处理失败，记录日志
				logx.Errorf("第%d次处理任务失败: %v", i+1, processErr)
				if i < retryCount-1 {
					time.Sleep(2 * time.Second) // 等待后重试
				}
			}

			// 重试全部失败，拒绝消息（可选：发送到死信队列）
			if processErr != nil {
				logx.Errorf("任务处理失败，已重试 %d 次: %v", retryCount, processErr)
				// false 表示不重新入队（避免无限循环）
				if nackErr := d.Nack(false, false); nackErr != nil {
					logx.Errorf("拒绝消息失败: %v", nackErr)
				}
			}
		}
	}()
}

func (c *Consumer) processFile(body []byte) (err error) {
	// 消息体 userIdentity,parentId,filePath,ext,name,size,isExisted,repositoryIdentity
	// 原文件存在与否
	// 压缩与否：文件路径，文件后缀
	// 存入 OSS
	// 存入数据库
	var task types.UploadEvent
	err = json.Unmarshal(body, &task)
	if err != nil {
		logx.Errorf("Failed to unmarshal message body: %v", err)
		return err
	}
	ur := new(models.UserRepository)
	// 特判：文件存在且上传的文件在当前父目录下已存在且用户 id 一致
	if task.IsExisted {
		had, queryErr := c.svcCtx.DBEngine.Table("user_repository").Where("repository_identity=? AND user_identity=?", task.RepositoryIdentity, task.UserIdentity).Get(ur)
		if queryErr != nil {
			return queryErr
		}
		// 直接返回：文件已存在
		if had {
			logx.Infof("文件秒传：用户 %s 已拥有此文件（repository_identity: %s）", task.UserIdentity, task.RepositoryIdentity)
			return nil
		}
	} else {
		// 开始处理
		// 文件不存在，进行上传
		// 将临时文件指针重置到开头以便上传
		var tempFile *os.File
		tempFile, err = os.Open(task.FilePath)
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name())
		if _, seekErr := tempFile.Seek(0, 0); seekErr != nil {
			return err
		}

		// 判断是否为视频或图片文件，如果是则先压缩
		videoExts := map[string]bool{
			".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
			".flv": true, ".wmv": true, ".webm": true, ".m4v": true,
		}
		imageExts := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
			".bmp": true, ".webp": true,
		}

		var uploadFile *os.File
		var uploadFilename string
		var compressedFilePath string // 用于记录压缩文件路径，以便清理
		var actualSize int64          // 实际上传的文件大小
		var finalUploadPath string    // 最终要上传的文件路径（用于分片上传）

		if videoExts[task.Ext] {
			logx.Info("是视频文件，需要压缩")
			// 是视频文件，需要压缩
			compressedFile, createErr := os.CreateTemp("", "compressed-*.mp4")
			if createErr != nil {
				return err
			}
			compressedFilePath = compressedFile.Name()
			// 注意：不在这里 defer，避免在秒传时也执行清理

			// 使用 ffmpeg 压缩视频
			_, compressErr := utils.CompressVideoWithFFmpeg(tempFile.Name(), compressedFile.Name(), 23, "128k")
			if compressErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return err
			}

			// 使用压缩后的文件上传
			uploadFile = compressedFile
			uploadFilename = task.Name
			finalUploadPath = compressedFile.Name()

			// 将文件指针重置到开头
			if _, seekErr := uploadFile.Seek(0, 0); seekErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return err
			}

			// 获取压缩后的文件大小
			fileInfo, statErr := uploadFile.Stat()
			if statErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return err
			}
			actualSize = fileInfo.Size()
		} else if imageExts[task.Ext] {
			// 是图片文件，需要压缩
			logx.Info("是图片文件，需要压缩")
			compressedFile, createErr := os.CreateTemp("", "compressed-*"+task.Ext)
			if createErr != nil {
				return err
			}
			compressedFilePath = compressedFile.Name()
			tempCompressedPath := compressedFilePath
			compressedFile.Close() // 先关闭，因为 CompressImage 会重新打开

			// 使用图片压缩（最大 1920x1080，质量 85）
			compressErr := utils.CompressImage(tempFile.Name(), tempCompressedPath, &utils.ImageCompressOptions{
				MaxWidth:  1920,
				MaxHeight: 1080,
				Quality:   85,
			})
			if compressErr != nil {
				os.Remove(tempCompressedPath)
				return err
			}

			// 重新打开压缩后的文件用于上传
			compressedFile, openErr := os.Open(tempCompressedPath)
			if openErr != nil {
				os.Remove(tempCompressedPath)
				return err
			}

			// 使用压缩后的文件上传
			uploadFile = compressedFile
			uploadFilename = task.Name
			finalUploadPath = tempCompressedPath

			// 获取压缩后的文件大小
			fileInfo, statErr := uploadFile.Stat()
			if statErr != nil {
				uploadFile.Close()
				os.Remove(tempCompressedPath)
				return err
			}
			actualSize = fileInfo.Size()
		} else {
			logx.Info("是其他文件类型，直接使用临时文件")
			// 非视频和图片文件，直接使用临时文件
			uploadFile = tempFile
			uploadFilename = task.Name
			finalUploadPath = tempFile.Name()
			actualSize = task.Size // 使用原始文件大小
		}

		// 根据文件大小选择上传方式
		var OssPath string
		if actualSize > common.MultipartUploadThreshold {
			// 大文件：使用分片上传
			logx.Infof("文件大小 %.2f MB 超过阈值，使用分片上传",
				float64(actualSize)/(1024*1024))

			// 关闭文件句柄（分片上传会重新打开）
			if uploadFile != tempFile {
				uploadFile.Close()
			}

			OssPath, err = utils.UploadToOSSMultipart(finalUploadPath, uploadFilename, actualSize)
		} else {
			// 小文件：使用普通上传
			logx.Infof("文件大小 %.2f KB 小于阈值，使用普通上传",
				float64(actualSize)/1024)
			OssPath, err = utils.UploadToOSS(uploadFile, uploadFilename)
		}

		// 上传完成后，立即清理压缩文件
		if compressedFilePath != "" {
			uploadFile.Close()
			logx.Infof("清理临时文件: %s", compressedFilePath)
			os.Remove(compressedFilePath)
		}

		if err != nil {
			return err
		}
		logx.Infof("开始存入数据库")

		// 文件不存在就存入中央数据库
		rp := &models.RepositoryPool{
			Name:      task.Name,
			Hash:      task.Hash,
			Ext:       path.Ext(OssPath),
			Size:      actualSize,
			Path:      OssPath,
			ObjectKey: OssPath,
			Identity:  utils.UUID(),
		}
		_, err = c.svcCtx.DBEngine.Insert(rp)
		if err != nil {
			return err
		}
	}
	// 最终都要逻辑添加到用户文件表
	_, err = c.InsertInToUserRepository(task.UserIdentity, task.RepositoryIdentity, task.Ext, task.Name, task.ParentId)
	if err != nil {
		return err
	}
	return nil

}

func (c *Consumer) InsertInToUserRepository(userIdentity, repositoryIdentity, ext, name string, parentId int64) (userRepositoryIdentity string, err error) {
	ur := &models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       userIdentity,
		RepositoryIdentity: repositoryIdentity,
		ParentId:           parentId,
		Ext:                ext,
		Name:               name,
	}
	_, err = c.svcCtx.DBEngine.Insert(ur)
	if err != nil {
		return "", err
	}
	return ur.Identity, nil
}
