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

type DownloadURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDownloadURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadURLLogic {
	return &DownloadURLLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DownloadURLLogic) DownloadURL(req *types.DownloadURLRequest) (resp *types.DownloadURLResponse, err error) {
	userIdentity, ok := l.ctx.Value("user_identity").(string)
	if !ok {
		return nil, errors.New("用户身份验证失败")
	}
	if req.RepositoryIdentity == "" {
		return nil, errors.New("文件标识不能为空")
	}
	expires := normalizeExpires(req.Expires)

	ref := new(models.UserRepository)
	has, err := l.svcCtx.DBEngine.
		Where("user_identity = ? AND repository_identity = ? AND (status != ? OR status IS NULL)", userIdentity, req.RepositoryIdentity, common.StatusDeleted).
		Get(ref)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("文件不存在")
	}

	repo := new(models.RepositoryPool)
	has, err = l.svcCtx.DBEngine.Where("identity = ?", req.RepositoryIdentity).Get(repo)
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

	cacheKey := fmt.Sprintf("download_url:%s:%d", req.RepositoryIdentity, expires)
	if url, ok := getCachedURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
		return &types.DownloadURLResponse{URL: url, Expires: expires}, nil
	}

	lockKey := "lock:" + cacheKey
	locked, err := utils.AcquireLock(l.ctx, l.svcCtx.RedisClient, lockKey, 10*time.Second)
	if err != nil {
		url, genErr := utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
		if genErr != nil {
			return nil, genErr
		}
		setCachedURL(l.ctx, l.svcCtx.RedisClient, cacheKey, url, expires)
		return &types.DownloadURLResponse{URL: url, Expires: expires}, nil
	}
	if !locked {
		time.Sleep(120 * time.Millisecond)
		if url, ok := getCachedURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
			return &types.DownloadURLResponse{URL: url, Expires: expires}, nil
		}
	}
	if locked {
		defer utils.ReleaseLock(l.ctx, l.svcCtx.RedisClient, lockKey)
	}

	if url, ok := getCachedURL(l.ctx, l.svcCtx.RedisClient, cacheKey); ok {
		return &types.DownloadURLResponse{URL: url, Expires: expires}, nil
	}

	url, err := utils.PresignGetObject(l.ctx, objectKey, time.Duration(expires)*time.Second)
	if err != nil {
		return nil, err
	}
	setCachedURL(l.ctx, l.svcCtx.RedisClient, cacheKey, url, expires)
	return &types.DownloadURLResponse{URL: url, Expires: expires}, nil
}

func normalizeExpires(expires int) int {
	if expires <= 0 {
		return 3600
	}
	max := int((7 * 24 * time.Hour).Seconds())
	if expires > max {
		return max
	}
	return expires
}

func getCachedURL(ctx context.Context, rdb svc.RedisClient, key string) (string, bool) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return "", false
	}
	if val == "" {
		return "", false
	}
	return val, true
}

func setCachedURL(ctx context.Context, rdb svc.RedisClient, key, url string, expires int) {
	_ = rdb.Set(ctx, key, url, time.Duration(expires)*time.Second).Err()
}
