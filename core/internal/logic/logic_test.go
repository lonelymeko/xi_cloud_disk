package logic

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"

	"github.com/redis/go-redis/v9"
	"xorm.io/xorm"

	_ "modernc.org/sqlite"
)

type testEnv struct {
	ctx context.Context
	svc *svc.ServiceContext
	rdb *fakeRedisClient
	eng *xorm.Engine
}

func newTestEnv(t *testing.T) *testEnv {
	eng, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("engine init failed: %v", err)
	}
	if err := utils.EnsureSchema(eng); err != nil {
		t.Fatalf("ensure schema failed: %v", err)
	}

	oldEnabled, oldHost, oldPort, oldUser, oldPass := utils.EmailEnabled(), utils.EmailHost(), utils.EmailPort(), utils.EmailUser(), utils.EmailPassword()
	utils.SetEmailConfig(false, "", "", "", "")

	rdb := newFakeRedisClient()

	cfg := config.Config{}
	cfg.Auth.AccessSecret = "secret"
	cfg.Auth.AccessExpire = 3600
	ctx := context.WithValue(context.Background(), "user_identity", "u-1")
	svcCtx := svc.NewServiceContextWithDeps(cfg, eng, rdb, func(next http.HandlerFunc) http.HandlerFunc { return next })

	t.Cleanup(func() {
		utils.SetEmailConfig(oldEnabled, oldHost, oldPort, oldUser, oldPass)
		_ = eng.Close()
	})

	return &testEnv{ctx: ctx, svc: svcCtx, rdb: rdb, eng: eng}
}

type fakeRedisClient struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{data: map[string]string{}}
}

func (f *fakeRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	val, ok := f.data[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(val, nil)
}

func (f *fakeRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	f.mu.Lock()
	f.data[key] = fmt.Sprint(value)
	f.mu.Unlock()
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[key]; ok {
		return redis.NewBoolResult(false, nil)
	}
	f.data[key] = fmt.Sprint(value)
	return redis.NewBoolResult(true, nil)
}

func (f *fakeRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	f.mu.Lock()
	var count int64
	for _, key := range keys {
		if _, ok := f.data[key]; ok {
			delete(f.data, key)
			count++
		}
	}
	f.mu.Unlock()
	return redis.NewIntResult(count, nil)
}

func (f *fakeRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", nil)
}

func TestLogin(t *testing.T) {
	env := newTestEnv(t)
	user := &models.UserBasic{
		Name:     "alice",
		Password: utils.Md5("pass1234"),
		Identity: "uid-1",
	}
	if _, err := env.eng.InsertOne(user); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	tests := []struct {
		name    string
		req     types.LoginRequest
		wantErr bool
	}{
		{name: "ok", req: types.LoginRequest{Name: "alice", Password: "pass1234"}},
		{name: "bad password", req: types.LoginRequest{Name: "alice", Password: "bad"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logic := NewLoginLogic(env.ctx, env.svc)
			resp, err := logic.Login(&tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("login failed: %v", err)
			}
			if resp.Token == "" || resp.Name != "alice" {
				t.Fatalf("unexpected response: %+v", resp)
			}
			claims, err := utils.ParseToken(resp.Token, env.svc.Config.Auth.AccessSecret, env.svc.Config.Auth.AccessExpire)
			if err != nil {
				t.Fatalf("parse token failed: %v", err)
			}
			if claims.Name != "alice" || claims.Identity != "uid-1" {
				t.Fatalf("claims mismatch: %+v", claims)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	env := newTestEnv(t)
	if err := env.rdb.Set(env.ctx, "verification_code:alice@example.com", "123456", time.Minute).Err(); err != nil {
		t.Fatalf("set code failed: %v", err)
	}

	logic := NewRegisterLogic(env.ctx, env.svc)
	resp, err := logic.Register(&types.RegisterRequest{
		Name:     "alice",
		Email:    "alice@example.com",
		Password: "pass1234",
		Code:     "123456",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.Token == "" || resp.Name != "alice" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	user := new(models.UserBasic)
	has, err := env.eng.Where("email = ?", "alice@example.com").Get(user)
	if err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if !has {
		t.Fatal("user not found")
	}
}

func TestChangePassword(t *testing.T) {
	env := newTestEnv(t)
	user := &models.UserBasic{Identity: "u-1", Name: "alice", Password: utils.Md5("oldpass1")}
	if _, err := env.eng.InsertOne(user); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	logic := NewChangePasswordLogic(env.ctx, env.svc)
	resp, err := logic.ChangePassword(&types.ChangePasswordRequest{
		Identity:    "u-1",
		OldPassword: "oldpass1",
		NewPassword: "newpass1",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("empty response")
	}

	updated := new(models.UserBasic)
	_, err = env.eng.Where("identity = ?", "u-1").Get(updated)
	if err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if updated.Password != utils.Md5("newpass1") {
		t.Fatalf("password not updated")
	}
}

func TestResetPassword(t *testing.T) {
	env := newTestEnv(t)
	user := &models.UserBasic{Identity: "u-1", Name: "alice", Email: "alice@example.com", Password: utils.Md5("oldpass1")}
	if _, err := env.eng.InsertOne(user); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	if err := env.rdb.Set(env.ctx, "verification_code:alice@example.com", "654321", time.Minute).Err(); err != nil {
		t.Fatalf("set code failed: %v", err)
	}

	logic := NewResetPasswordLogic(env.ctx, env.svc)
	resp, err := logic.ResetPassword(&types.ResetPasswordRequest{
		Email:       "alice@example.com",
		Code:        "654321",
		NewPassword: "newpass1",
	})
	if err != nil {
		t.Fatalf("reset password failed: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("empty response")
	}

	updated := new(models.UserBasic)
	_, err = env.eng.Where("email = ?", "alice@example.com").Get(updated)
	if err != nil {
		t.Fatalf("query user failed: %v", err)
	}
	if updated.Password != utils.Md5("newpass1") {
		t.Fatalf("password not updated")
	}

	val, err := env.rdb.Get(env.ctx, "verification_code:alice@example.com").Result()
	if err == nil || val != "" {
		t.Fatal("verification code not deleted")
	}
}

func TestSendVerificationCode(t *testing.T) {
	env := newTestEnv(t)
	logic := NewSendVerificationCodeLogic(env.ctx, env.svc)
	resp, err := logic.SendVerificationCode(&types.SendVerificationCodeRequest{Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("send code failed: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("empty response")
	}
	val, err := env.rdb.Get(env.ctx, "verification_code:alice@example.com").Result()
	if err != nil {
		t.Fatalf("code missing: %v", err)
	}
	if len(val) != 6 {
		t.Fatalf("code length mismatch: %s", val)
	}
}

func TestUserDetail(t *testing.T) {
	env := newTestEnv(t)
	user := &models.UserBasic{Identity: "uid-1", Name: "alice", Email: "alice@example.com"}
	if _, err := env.eng.InsertOne(user); err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	logic := NewUserDetailLogic(env.ctx, env.svc)
	resp, err := logic.UserDetail(&types.UserDetailRequest{Identity: "uid-1"})
	if err != nil {
		t.Fatalf("user detail failed: %v", err)
	}
	if resp.Name != "alice" || resp.Email != "alice@example.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUploadFile(t *testing.T) {
	env := newTestEnv(t)
	logic := NewUploadFileLogic(env.ctx, env.svc)
	resp, err := logic.UploadFile(&types.UploadFileRequest{Name: "a.txt", Hash: "h", Ext: ".txt", Size: 10, ObjectKey: "k", ParentId: 0}, false, "")
	if err != nil {
		t.Fatalf("upload file failed: %v", err)
	}
	if resp.Message == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	data := new(models.RepositoryPool)
	_, err = env.eng.Where("hash = ?", "h").Get(data)
	if err != nil {
		t.Fatalf("query repo failed: %v", err)
	}
	if data.Identity == "" {
		t.Fatal("repo not inserted")
	}
}

func TestUserFolderCreate(t *testing.T) {
	env := newTestEnv(t)
	logic := NewUserFolderCreateLogic(env.ctx, env.svc)
	_, err := logic.UserFolderCreate(&types.UserFolderCreateRequest{ParentId: 0, Name: "docs"})
	if err != nil {
		t.Fatalf("create folder failed: %v", err)
	}

	count, err := env.eng.Table("user_repository").Where("name = ?", "docs").Count(new(models.UserRepository))
	if err != nil {
		t.Fatalf("query folder failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("folder count mismatch: %d", count)
	}
}

func TestUserFileList(t *testing.T) {
	env := newTestEnv(t)
	repo := &models.RepositoryPool{Identity: "r1", Name: "file", Ext: ".txt", Size: 12, ObjectKey: "k"}
	if _, err := env.eng.InsertOne(repo); err != nil {
		t.Fatalf("insert repo failed: %v", err)
	}
	file := &models.UserRepository{Identity: "f1", UserIdentity: "u-1", ParentId: 0, Name: "file", RepositoryIdentity: "r1", Ext: ".txt"}
	if _, err := env.eng.InsertOne(file); err != nil {
		t.Fatalf("insert file failed: %v", err)
	}

	logic := NewUserFileListLogic(env.ctx, env.svc)
	resp, err := logic.UserFileList(&types.UserFileListRequest{Id: 0})
	if err != nil {
		t.Fatalf("file list failed: %v", err)
	}
	if resp.Count != 1 || len(resp.List) != 1 {
		t.Fatalf("list mismatch: %+v", resp)
	}
}

func TestUserFileMove(t *testing.T) {
	env := newTestEnv(t)
	parent := &models.UserRepository{Identity: "p1", UserIdentity: "u-1", ParentId: 0, Name: "dst"}
	if _, err := env.eng.InsertOne(parent); err != nil {
		t.Fatalf("insert parent failed: %v", err)
	}
	file := &models.UserRepository{Identity: "f1", UserIdentity: "u-1", ParentId: 0, Name: "file"}
	if _, err := env.eng.InsertOne(file); err != nil {
		t.Fatalf("insert file failed: %v", err)
	}

	logic := NewUserFileMoveLogic(env.ctx, env.svc)
	_, err := logic.UserFileMove(&types.UserFileMoveRequest{Identity: "f1", Name: "file", ParentId: int64(parent.Id)})
	if err != nil {
		t.Fatalf("move file failed: %v", err)
	}

	updated := new(models.UserRepository)
	_, err = env.eng.Where("identity = ?", "f1").Get(updated)
	if err != nil {
		t.Fatalf("query file failed: %v", err)
	}
	if updated.ParentId != int64(parent.Id) {
		t.Fatalf("parent id mismatch: %d", updated.ParentId)
	}
}

func TestUserFileNameUpdate(t *testing.T) {
	env := newTestEnv(t)
	file := &models.UserRepository{Identity: "f1", UserIdentity: "u-1", ParentId: 0, Name: "old"}
	if _, err := env.eng.InsertOne(file); err != nil {
		t.Fatalf("insert file failed: %v", err)
	}
	logic := NewUserFileNameUpdateLogic(env.ctx, env.svc)
	_, err := logic.UserFileNameUpdate(&types.UserFileNameUpdateRequest{Identity: "f1", Name: "new"})
	if err != nil {
		t.Fatalf("update name failed: %v", err)
	}
	updated := new(models.UserRepository)
	_, err = env.eng.Where("identity = ?", "f1").Get(updated)
	if err != nil {
		t.Fatalf("query file failed: %v", err)
	}
	if updated.Name != "new" {
		t.Fatalf("name mismatch: %s", updated.Name)
	}
}

func TestSaveResource(t *testing.T) {
	env := newTestEnv(t)
	base := &models.UserRepository{Identity: "r1", UserIdentity: "u-1", ParentId: 0, RepositoryIdentity: "r1", Ext: ".txt", Name: "src"}
	if _, err := env.eng.InsertOne(base); err != nil {
		t.Fatalf("insert base failed: %v", err)
	}
	logic := NewSaveResourceLogic(env.ctx, env.svc)
	resp, err := logic.SaveResource(&types.SaveResourceRequest{ParentId: 0, RepositoryIdentity: "r1", Name: "dst"})
	if err != nil {
		t.Fatalf("save resource failed: %v", err)
	}
	if resp.Identity == "" {
		t.Fatal("empty identity")
	}

	count, err := env.eng.Table("user_repository").Where("name = ?", "dst").Count(new(models.UserRepository))
	if err != nil {
		t.Fatalf("query resource failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("count mismatch: %d", count)
	}
}

func TestCreateShareRecord(t *testing.T) {
	env := newTestEnv(t)
	logic := NewCreateShareRecordLogic(env.ctx, env.svc)
	resp, err := logic.CreateShareRecord(&types.CreateShareRecordRequest{Identity: "r1", ExpiredTime: 10})
	if err != nil {
		t.Fatalf("create share failed: %v", err)
	}
	if resp.Identity == "" {
		t.Fatal("empty identity")
	}
}

func TestGetShareRecord(t *testing.T) {
	env := newTestEnv(t)
	share := &models.ShareBasic{Identity: "s1", UserIdentity: "u-1", RepositoryIdentity: "r1", ExpiredTime: 10}
	if _, err := env.eng.InsertOne(share); err != nil {
		t.Fatalf("insert share failed: %v", err)
	}
	repo := &models.RepositoryPool{Identity: "r1", Name: "file", Ext: ".txt", Size: 12, ObjectKey: "k"}
	if _, err := env.eng.InsertOne(repo); err != nil {
		t.Fatalf("insert repo failed: %v", err)
	}
	file := &models.UserRepository{Identity: "f1", UserIdentity: "u-1", ParentId: 0, Name: "file", RepositoryIdentity: "r1", Ext: ".txt"}
	if _, err := env.eng.InsertOne(file); err != nil {
		t.Fatalf("insert file failed: %v", err)
	}

	logic := NewGetShareRecordLogic(env.ctx, env.svc)
	resp, err := logic.GetShareRecord(&types.GetShareRecordRequest{Identity: "s1"})
	if err != nil {
		t.Fatalf("get share failed: %v", err)
	}
	if resp.Name != "file" || resp.Ext != ".txt" || resp.Size != 12 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUserFolderDelete(t *testing.T) {
	env := newTestEnv(t)
	root := &models.UserRepository{Identity: "root", UserIdentity: "u-1", ParentId: 0, Name: "root"}
	if _, err := env.eng.InsertOne(root); err != nil {
		t.Fatalf("insert root failed: %v", err)
	}
	if root.Id == 0 {
		reload := new(models.UserRepository)
		has, err := env.eng.Where("identity = ?", "root").Get(reload)
		if err != nil {
			t.Fatalf("reload root failed: %v", err)
		}
		if !has {
			t.Fatal("root not found")
		}
		root.Id = reload.Id
	}
	child := &models.UserRepository{Identity: "child", UserIdentity: "u-1", ParentId: int64(root.Id), Name: "child"}
	if _, err := env.eng.InsertOne(child); err != nil {
		t.Fatalf("insert child failed: %v", err)
	}

	logic := NewUserFolderDeleteLogic(env.ctx, env.svc)
	_, err := logic.UserFolderDelete(&types.UserFolderDeleteRequest{Identity: "root"})
	if err != nil {
		t.Fatalf("delete folder failed: %v", err)
	}

	rootCheck := new(models.UserRepository)
	has, err := env.eng.Where("identity = ?", "root").Get(rootCheck)
	if err != nil {
		t.Fatalf("query root failed: %v", err)
	}
	if has {
		t.Fatal("root not deleted")
	}
	childCheck := new(models.UserRepository)
	has, err = env.eng.Where("identity = ?", "child").Get(childCheck)
	if err != nil {
		t.Fatalf("query child failed: %v", err)
	}
	if has {
		t.Fatal("child not deleted")
	}
}
