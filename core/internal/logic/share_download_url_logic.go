package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// ShareDownloadURLLogic 分享下载链接逻辑。
type ShareDownloadURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewShareDownloadURLLogic 创建分享下载链接逻辑。
func NewShareDownloadURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShareDownloadURLLogic {
	return &ShareDownloadURLLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ShareDownloadURL 获取分享下载链接。
func (l *ShareDownloadURLLogic) ShareDownloadURL(req *types.ShareDownloadURLRequest) (resp *types.ShareDownloadURLResponse, err error) {
	if req.ShareIdentity == "" {
		return nil, errors.New("分享标识不能为空")
	}
	expires := normalizeExpires(req.Expires)

	share := new(models.ShareBasic)
	has, err := l.svcCtx.DBEngine.Where("identity = ?", req.ShareIdentity).Get(share)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("分享不存在")
	}
	if share.ExpiredTime > 0 {
		createdAt, parseErr := time.Parse(common.DataTimeFormat, share.CreatedAt)
		if parseErr != nil {
			return nil, parseErr
		}
		if createdAt.Add(time.Duration(share.ExpiredTime) * time.Second).Before(time.Now()) {
			return nil, errors.New("分享已过期")
		}
	}

	repo := new(models.RepositoryPool)
	has, err = l.svcCtx.DBEngine.Where("identity = ?", share.RepositoryIdentity).Get(repo)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("文件不存在")
	}
	objectKey := repo.ObjectKey
	if objectKey == "" {
		objectKey = utils.ObjectKeyFromPath(repo.Path)
	}
	if objectKey == "" {
		return nil, errors.New("文件未绑定对象键")
	}

	cacheKey := fmt.Sprintf("share_download_url:%s:%d", req.ShareIdentity, expires)
	if url, ok := getCachedShareURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
		return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
	}

	lockKey := "lock:" + cacheKey
	locked, err := utils.AcquireLock(l.ctx, l.svcCtx.RedisClient, lockKey, 10*time.Second)
	if err != nil {
		url, genErr := utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
		if genErr != nil {
			return nil, genErr
		}
		setCachedShareURL(l.ctx, l.svcCtx.RedisClient, cacheKey, url, expires)
		return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
	}
	if !locked {
		time.Sleep(120 * time.Millisecond)
		if url, ok := getCachedShareURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
			return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
		}
	}
	if locked {
		defer utils.ReleaseLock(l.ctx, l.svcCtx.RedisClient, lockKey)
	}

	if url, ok := getCachedShareURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
		return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
	}

	url, err := utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
	if err != nil {
		return nil, err
	}
	setCachedShareURL(l.ctx, l.svcCtx.RedisClient, cacheKey, url, expires)
	return &types.ShareDownloadURLResponse{URL: url, Expires: expires}, nil
}

// getCachedShareURL 读取缓存的分享下载链接。
func getCachedShareURL(ctx context.Context, rdb svc.RedisClient, key string) (string, bool) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return "", false
	}
	if val == "" {
		return "", false
	}
	return val, true
}

// setCachedShareURL 写入缓存的分享下载链接。
func setCachedShareURL(ctx context.Context, rdb svc.RedisClient, key, url string, expires int) {
	_ = rdb.Set(ctx, key, url, time.Duration(expires)*time.Second).Err()
}
