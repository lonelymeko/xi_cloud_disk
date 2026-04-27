package cache

import (
	"cloud_disk/core/common"
	"cloud_disk/core/internal/svc"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// UploadTaskMeta 上传任务元信息。
type UploadTaskMeta struct {
	Hash      string `json:"hash"`
	Name      string `json:"name"`
	Ext       string `json:"ext"`
	Size      int64  `json:"size"`
	TaskKey   string `json:"task_key"`
	UpdatedAt string `json:"updated_at"`
}

// MultipartUploadSession 记录可恢复的分片上传会话。
type MultipartUploadSession struct {
	UploadID  string
	ObjectKey string
	FileSize  int64
}

func BuildUploadTaskUniqueFeature(userIdentity, hash string) string {
	return fmt.Sprintf("%s:%s", userIdentity, hash)
}

func BuildUploadTaskStateKey(uniqueFeature string) string {
	return common.UploadTaskStatePrefix + uniqueFeature
}

func BuildUploadTaskHashKey(md5 string) string {
	return common.UploadTaskHashPrefix + md5
}

func BuildUploadTaskUserIndexKey(userIdentity string) string {
	return common.UploadTaskUserIndexPrefix + userIdentity
}

func BuildUserFileListCountKey(userIdentity string, parentID int64) string {
	return fmt.Sprintf("%s%s:%d", common.UserFileListCountPrefix, userIdentity, parentID)
}

func SetUploadTaskState(ctx context.Context, rdb svc.RedisClient, userIdentity, hash string, state int) error {
	if userIdentity == "" || hash == "" {
		return nil
	}
	stateKey := BuildUploadTaskStateKey(BuildUploadTaskUniqueFeature(userIdentity, hash))
	return rdb.Set(ctx, stateKey, state, common.UploadTaskTTL).Err()
}

func GetUploadTaskState(ctx context.Context, rdb svc.RedisClient, userIdentity, hash string) (int, bool) {
	if userIdentity == "" || hash == "" {
		return 0, false
	}
	stateKey := BuildUploadTaskStateKey(BuildUploadTaskUniqueFeature(userIdentity, hash))
	val, err := rdb.Get(ctx, stateKey).Result()
	if err != nil {
		return 0, false
	}
	n, convErr := strconv.Atoi(val)
	if convErr != nil {
		return 0, false
	}
	return n, true
}

func SaveUploadTaskMeta(ctx context.Context, rdb svc.RedisClient, userIdentity string, meta UploadTaskMeta) error {
	if userIdentity == "" || meta.Hash == "" {
		return nil
	}
	if meta.TaskKey == "" {
		meta.TaskKey = BuildUploadTaskUniqueFeature(userIdentity, meta.Hash)
	}
	if meta.UpdatedAt == "" {
		meta.UpdatedAt = time.Now().Format(common.DataTimeFormat)
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	key := BuildUploadTaskUserIndexKey(userIdentity)
	if err := rdb.HSet(ctx, key, meta.Hash, string(payload)).Err(); err != nil {
		return err
	}
	return rdb.Expire(ctx, key, common.UploadTaskTTL).Err()
}

func ListUploadTaskMeta(ctx context.Context, rdb svc.RedisClient, userIdentity string) ([]UploadTaskMeta, error) {
	key := BuildUploadTaskUserIndexKey(userIdentity)
	items, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	result := make([]UploadTaskMeta, 0, len(items))
	for hash, raw := range items {
		meta := UploadTaskMeta{Hash: hash}
		if unmarshalErr := json.Unmarshal([]byte(raw), &meta); unmarshalErr != nil {
			meta.Hash = hash
		}
		if meta.Hash == "" {
			meta.Hash = hash
		}
		result = append(result, meta)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt > result[j].UpdatedAt
	})
	return result, nil
}

func SetUploadPartState(ctx context.Context, rdb svc.RedisClient, md5 string, partIndex int, state string) error {
	if md5 == "" || partIndex <= 0 {
		return nil
	}
	if state == "" {
		state = "1"
	}
	key := BuildUploadTaskHashKey(md5)
	if err := rdb.HSet(ctx, key, strconv.Itoa(partIndex), state).Err(); err != nil {
		return err
	}
	return rdb.Expire(ctx, key, common.UploadTaskTTL).Err()
}

func SaveMultipartUploadSession(ctx context.Context, rdb svc.RedisClient, md5 string, session MultipartUploadSession) error {
	if md5 == "" || session.UploadID == "" || session.ObjectKey == "" {
		return nil
	}
	key := BuildUploadTaskHashKey(md5)
	if err := rdb.HSet(ctx, key,
		"__upload_id", session.UploadID,
		"__object_key", session.ObjectKey,
		"__file_size", strconv.FormatInt(session.FileSize, 10),
	).Err(); err != nil {
		return err
	}
	return rdb.Expire(ctx, key, common.UploadTaskTTL).Err()
}

func GetMultipartUploadSession(ctx context.Context, rdb svc.RedisClient, md5 string) (MultipartUploadSession, bool, error) {
	if md5 == "" {
		return MultipartUploadSession{}, false, nil
	}
	items, err := rdb.HGetAll(ctx, BuildUploadTaskHashKey(md5)).Result()
	if err != nil {
		return MultipartUploadSession{}, false, err
	}
	uploadID := items["__upload_id"]
	objectKey := items["__object_key"]
	if uploadID == "" || objectKey == "" {
		return MultipartUploadSession{}, false, nil
	}
	fileSize, _ := strconv.ParseInt(items["__file_size"], 10, 64)
	return MultipartUploadSession{
		UploadID:  uploadID,
		ObjectKey: objectKey,
		FileSize:  fileSize,
	}, true, nil
}

func GetUploadedPartETags(ctx context.Context, rdb svc.RedisClient, md5 string) (map[int]string, error) {
	if md5 == "" {
		return map[int]string{}, nil
	}
	items, err := rdb.HGetAll(ctx, BuildUploadTaskHashKey(md5)).Result()
	if err != nil {
		return nil, err
	}
	parts := make(map[int]string, len(items))
	for field, etag := range items {
		idx, convErr := strconv.Atoi(field)
		if convErr != nil || idx <= 0 || etag == "" {
			continue
		}
		parts[idx] = etag
	}
	return parts, nil
}

func ClearUploadPartState(ctx context.Context, rdb svc.RedisClient, md5 string) error {
	if md5 == "" {
		return nil
	}
	return rdb.Del(ctx, BuildUploadTaskHashKey(md5)).Err()
}

func GetUploadedPartIndexes(ctx context.Context, rdb svc.RedisClient, md5 string) ([]int, error) {
	if md5 == "" {
		return []int{}, nil
	}
	items, err := rdb.HGetAll(ctx, BuildUploadTaskHashKey(md5)).Result()
	if err != nil {
		return nil, err
	}
	parts := make([]int, 0, len(items))
	for field := range items {
		idx, convErr := strconv.Atoi(field)
		if convErr != nil {
			continue
		}
		parts = append(parts, idx)
	}
	sort.Ints(parts)
	return parts, nil
}
