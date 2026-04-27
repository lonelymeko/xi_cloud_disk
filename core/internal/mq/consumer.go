package mq

import (
	"bytes"
	"cloud_disk/core/common"
	"cloud_disk/core/internal/cache"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"strings"
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

			// 记录临时文件路径，统一在消息处理结束后清理，避免重试时文件被提前删除。
			var cleanupTask types.UploadEvent
			_ = json.Unmarshal(d.Body, &cleanupTask)

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

			if cleanupTask.FilePath != "" {
				_ = os.Remove(cleanupTask.FilePath)
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
	_ = cache.SetUploadTaskState(c.ctx, c.svcCtx.RedisClient, task.UserIdentity, task.Hash, 1)

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
		defer tempFile.Close()
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

		if videoExts[task.Ext] && task.Size > common.MultipartUploadThreshold {
			logx.Info("是视频文件，需要压缩")
			// 是视频文件，需要压缩
			compressedFile, createErr := os.CreateTemp("", "compressed-*.mp4")
			if createErr != nil {
				return createErr
			}
			compressedFilePath = compressedFile.Name()
			// 注意：不在这里 defer，避免在秒传时也执行清理

			// 使用 ffmpeg 压缩视频
			_, compressErr := utils.CompressVideoWithFFmpeg(tempFile.Name(), compressedFile.Name(), 26, "96k")
			if compressErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return compressErr
			}

			// 使用压缩后的文件上传
			uploadFile = compressedFile
			uploadFilename = task.Name
			finalUploadPath = compressedFile.Name()

			// 将文件指针重置到开头
			if _, seekErr := uploadFile.Seek(0, 0); seekErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return seekErr
			}

			// 获取压缩后的文件大小
			fileInfo, statErr := uploadFile.Stat()
			if statErr != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				return statErr
			}
			actualSize = fileInfo.Size()
		} else if imageExts[task.Ext] {
			// 是图片文件，需要压缩
			logx.Info("是图片文件，需要压缩")
			compressedFile, createErr := os.CreateTemp("", "compressed-*"+task.Ext)
			if createErr != nil {
				return createErr
			}
			compressedFilePath = compressedFile.Name()
			tempCompressedPath := compressedFilePath
			compressedFile.Close() // 先关闭，因为 CompressImage 会重新打开

			// 使用图片压缩（优先耗时，兼顾体积与质量）
			compressErr := utils.CompressImage(tempFile.Name(), tempCompressedPath, &utils.ImageCompressOptions{
				MaxWidth:  1600,
				MaxHeight: 900,
				Quality:   80,
			})
			if compressErr != nil {
				os.Remove(tempCompressedPath)
				logx.Errorf("图片压缩失败，回退原图上传: %v", compressErr)
				uploadFile = tempFile
				uploadFilename = task.Name
				finalUploadPath = tempFile.Name()
				actualSize = task.Size
			} else {
				// 重新打开压缩后的文件用于上传
				compressedFile, openErr := os.Open(tempCompressedPath)
				if openErr != nil {
					os.Remove(tempCompressedPath)
					return openErr
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
					return statErr
				}
				actualSize = fileInfo.Size()
			}
		} else {
			logx.Info("是其他文件类型，直接使用临时文件")
			// 非视频和图片文件，直接使用临时文件
			uploadFile = tempFile
			uploadFilename = task.Name
			finalUploadPath = tempFile.Name()
			actualSize = task.Size // 使用原始文件大小
		}

		// 业务层按环境变量选择存储实现（OSS/TOS）。
		_ = cache.SetUploadTaskState(c.ctx, c.svcCtx.RedisClient, task.UserIdentity, task.Hash, 2)
		uploadStartedAt := time.Now()
		OssPath, err := c.uploadByStorageType(uploadFile, tempFile, finalUploadPath, uploadFilename, actualSize, task.Hash)
		uploadElapsed := time.Since(uploadStartedAt)
		if err != nil {
			logx.Errorf("上传对象存储失败，文件名: %s，上传文件总大小: %s，耗时: %s，错误: %v",
				uploadFilename, formatUploadSize(actualSize), uploadElapsed.Round(time.Millisecond), err)
		} else {
			logx.Infof("上传对象存储完成，文件名: %s，ObjectKey: %s，上传文件总大小: %s，耗时: %s",
				uploadFilename, OssPath, formatUploadSize(actualSize), uploadElapsed.Round(time.Millisecond))
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
		_ = cache.SetUploadTaskState(c.ctx, c.svcCtx.RedisClient, task.UserIdentity, task.Hash, 3)

		// 文件不存在就存入中央数据库
		rp := &models.RepositoryPool{
			Name:      task.Name,
			Hash:      task.Hash,
			Ext:       path.Ext(OssPath),
			Size:      actualSize,
			Path:      OssPath,
			ObjectKey: OssPath,
			Identity:  task.RepositoryIdentity,
		}
		_, err = c.svcCtx.DBEngine.Insert(rp)
		// 存入布隆过滤器
		c.svcCtx.MyBloomFilter.AddFileHash(task.Hash)
		if err != nil {
			return err
		}
	}
	// 最终都要逻辑添加到用户文件表
	_, err = c.InsertInToUserRepository(task.UserIdentity, task.RepositoryIdentity, task.Ext, task.Name, task.ParentId)
	if err != nil {
		return err
	}
	_ = cache.SaveUploadTaskMeta(c.ctx, c.svcCtx.RedisClient, task.UserIdentity, cache.UploadTaskMeta{
		Hash:      task.Hash,
		Name:      task.Name,
		Ext:       task.Ext,
		Size:      task.Size,
		TaskKey:   cache.BuildUploadTaskUniqueFeature(task.UserIdentity, task.Hash),
		UpdatedAt: "",
	})
	_ = cache.SetUploadTaskState(c.ctx, c.svcCtx.RedisClient, task.UserIdentity, task.Hash, 4)
	return nil

}

func formatUploadSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	for _, unitName := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.2f %s", value, unitName)
		}
	}
	return fmt.Sprintf("%.2f PB", value/unit)
}

func (c *Consumer) uploadByStorageType(uploadFile, tempFile *os.File, finalUploadPath, uploadFilename string, actualSize int64, hash string) (string, error) {
	storageType := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_TYPE")))
	if storageType == "" {
		storageType = "oss"
	}
	storageName := strings.ToUpper(storageType)

	if actualSize > common.MultipartUploadThreshold {
		logx.Infof("文件大小 %.2f MB 超过阈值，使用 %s 统一接口分片上传", float64(actualSize)/(1024*1024), storageName)
		if uploadFile != tempFile {
			uploadFile.Close()
		}
		return c.uploadMultipartViaStorage(finalUploadPath, uploadFilename, actualSize, hash)
	}

	logx.Infof("文件大小 %.2f KB 小于阈值，使用 %s 统一接口普通上传", float64(actualSize)/1024, storageName)
	if _, err := uploadFile.Seek(0, 0); err != nil {
		return "", err
	}
	return c.uploadSingleViaStorage(uploadFile, uploadFilename, actualSize)
}

func (c *Consumer) uploadSingleViaStorage(uploadFile *os.File, uploadFilename string, actualSize int64) (string, error) {
	storage, err := utils.GetStorage()
	if err != nil {
		return "", err
	}

	objectKey := utils.UUID() + path.Ext(uploadFilename)
	contentType := mime.TypeByExtension(path.Ext(uploadFilename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Minute)
	defer cancel()

	if err := storage.PutObject(ctx, objectKey, uploadFile, actualSize, contentType); err != nil {
		return "", err
	}
	return objectKey, nil
}

func (c *Consumer) uploadMultipartViaStorage(filePath, uploadFilename string, fileSize int64, hash string) (string, error) {
	storage, err := utils.GetStorage()
	if err != nil {
		return "", err
	}

	objectKey := utils.UUID() + path.Ext(uploadFilename)
	contentType := mime.TypeByExtension(path.Ext(uploadFilename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	timeout := time.Duration(fileSize/1024/1024) * time.Second * 2
	if timeout < 5*time.Minute {
		timeout = 5 * time.Minute
	}
	if timeout > 30*time.Minute {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	uploadID := ""
	if session, ok, sessionErr := cache.GetMultipartUploadSession(ctx, c.svcCtx.RedisClient, hash); sessionErr != nil {
		logx.Errorf("读取分片上传会话缓存失败: %v", sessionErr)
	} else if ok && (session.FileSize == 0 || session.FileSize == fileSize) {
		uploadID = session.UploadID
		objectKey = session.ObjectKey
		logx.Infof("复用分片上传会话，ObjectKey: %s，UploadID: %s", objectKey, uploadID)
	} else if ok {
		if err := cache.ClearUploadPartState(ctx, c.svcCtx.RedisClient, hash); err != nil {
			logx.Errorf("清理过期分片上传缓存失败: %v", err)
		}
	}
	if uploadID == "" {
		if err := cache.ClearUploadPartState(ctx, c.svcCtx.RedisClient, hash); err != nil {
			logx.Errorf("清理旧分片上传缓存失败: %v", err)
		}
		uploadID, err = storage.InitiateMultipartUpload(ctx, objectKey, contentType)
		if err != nil {
			return "", fmt.Errorf("初始化分片上传失败: %w", err)
		}
		if err := cache.SaveMultipartUploadSession(ctx, c.svcCtx.RedisClient, hash, cache.MultipartUploadSession{
			UploadID:  uploadID,
			ObjectKey: objectKey,
			FileSize:  fileSize,
		}); err != nil {
			logx.Errorf("保存分片上传会话缓存失败: %v", err)
		}
	}

	success := false
	defer func() {
		if !success {
			logx.Errorf("分片上传未完成，保留 UploadID 以便任务重试复用: ObjectKey=%s, UploadID=%s", objectKey, uploadID)
		}
	}()

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	partSize := int64(common.PartSize)
	totalParts := (fileSize + partSize - 1) / partSize
	parts := make([]utils.Part, 0, totalParts)
	uploadedPartETags, err := cache.GetUploadedPartETags(ctx, c.svcCtx.RedisClient, hash)
	if err != nil {
		logx.Errorf("读取已上传分片缓存失败: %v", err)
		uploadedPartETags = map[int]string{}
	}

	for i := int64(0); i < totalParts; i++ {
		partNumber := int(i + 1)
		if etag := uploadedPartETags[partNumber]; etag != "" {
			parts = append(parts, utils.Part{PartNumber: partNumber, ETag: etag})
			logx.Infof("跳过已上传分片 %d/%d", partNumber, totalParts)
			continue
		}

		offset := i * partSize
		currentPartSize := partSize
		if offset+currentPartSize > fileSize {
			currentPartSize = fileSize - offset
		}

		partData := make([]byte, currentPartSize)
		n, readErr := file.ReadAt(partData, offset)
		if readErr != nil && readErr != io.EOF {
			return "", fmt.Errorf("读取分片 %d 失败: %w", i+1, readErr)
		}

		partCtx, partCancel := context.WithTimeout(ctx, 3*time.Minute)
		etag, uploadErr := storage.UploadPart(partCtx, objectKey, uploadID, partNumber, bytes.NewReader(partData[:n]), int64(n))
		partCancel()
		if uploadErr != nil {
			return "", fmt.Errorf("上传分片 %d 失败: %w", partNumber, uploadErr)
		}
		if err := cache.SetUploadPartState(ctx, c.svcCtx.RedisClient, hash, partNumber, etag); err != nil {
			logx.Errorf("保存分片 %d 缓存失败: %v", partNumber, err)
		}

		parts = append(parts, utils.Part{PartNumber: partNumber, ETag: etag})
	}

	completeCtx, completeCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer completeCancel()
	if err := storage.CompleteMultipartUpload(completeCtx, objectKey, uploadID, parts); err != nil {
		return "", fmt.Errorf("完成分片上传失败: %w", err)
	}

	success = true
	if err := cache.ClearUploadPartState(context.Background(), c.svcCtx.RedisClient, hash); err != nil {
		logx.Errorf("清理分片上传缓存失败: %v", err)
	}
	return objectKey, nil
}

func (c *Consumer) InsertInToUserRepository(userIdentity, repositoryIdentity, ext, name string, parentId int64) (userRepositoryIdentity string, err error) {
	ur := &models.UserRepository{
		Identity:           utils.UUID(),
		UserIdentity:       userIdentity,
		RepositoryIdentity: repositoryIdentity,
		ParentId:           parentId,
		Ext:                ext,
		Name:               name,
		Status:             common.StatusActive,
	}
	_, err = c.svcCtx.DBEngine.Insert(ur)
	if err != nil {
		return "", err
	}
	return ur.Identity, nil
}
