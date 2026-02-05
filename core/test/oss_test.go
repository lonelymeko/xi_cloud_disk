package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"cloud_disk/core/common"
	"cloud_disk/core/utils"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// TestUploadToOSS 验证上传函数调用链。
func TestUploadToOSS(t *testing.T) {
	oldLoad := utils.OSSLoadEnv()
	oldKeyGen := utils.OSSKeyGen()
	oldUpload := utils.OSSUpload()
	oldRegion := common.OSSRegion
	oldBucket := common.OSSBucketName

	common.OSSRegion = "r1"
	common.OSSBucketName = "b1"
	utils.SetOSSLoadEnv(func() error { return nil })
	utils.SetOSSKeyGen(func(originalFilename string) string { return "k.txt" })
	called := false
	utils.SetOSSUpload(func(region, bucket, key string, body io.Reader) (string, error) {
		called = true
		if region != common.OSSRegion || bucket != common.OSSBucketName || key != "k.txt" {
			return "", io.EOF
		}
		data, err := io.ReadAll(body)
		if err != nil {
			return "", err
		}
		if string(data) != "data" {
			return "", io.ErrUnexpectedEOF
		}
		return "etag", nil
	})

	t.Cleanup(func() {
		utils.SetOSSLoadEnv(oldLoad)
		utils.SetOSSKeyGen(oldKeyGen)
		utils.SetOSSUpload(oldUpload)
		common.OSSRegion = oldRegion
		common.OSSBucketName = oldBucket
	})

	objectKey, err := utils.UploadToOSS(bytes.NewBufferString("data"), "a.txt")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if !called {
		t.Fatal("upload not called")
	}
	if objectKey != "k.txt" {
		t.Fatalf("unexpected object key: %s", objectKey)
	}
}

// TestUploadToOSS_ErrorWrap 验证错误包装行为。
func TestUploadToOSS_ErrorWrap(t *testing.T) {
	oldLoad := utils.OSSLoadEnv()
	oldKeyGen := utils.OSSKeyGen()
	oldUpload := utils.OSSUpload()

	utils.SetOSSLoadEnv(func() error { return nil })
	utils.SetOSSKeyGen(func(originalFilename string) string { return "k.txt" })
	sentinel := errors.New("sentinel")
	utils.SetOSSUpload(func(region, bucket, key string, body io.Reader) (string, error) {
		return "", sentinel
	})

	t.Cleanup(func() {
		utils.SetOSSLoadEnv(oldLoad)
		utils.SetOSSKeyGen(oldKeyGen)
		utils.SetOSSUpload(oldUpload)
	})

	_, err := utils.UploadToOSS(bytes.NewBufferString("data"), "a.txt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestOSSHost 验证 OSS 域名拼接逻辑。
func TestOSSHost(t *testing.T) {
	oldBucket := common.OSSBucketName
	oldRegion := common.OSSRegion
	common.OSSBucketName = "bucket-default"
	common.OSSRegion = "region-default"
	t.Cleanup(func() {
		common.OSSBucketName = oldBucket
		common.OSSRegion = oldRegion
	})

	type tc struct {
		name   string
		host   string
		bucket string
		region string
		expect string
	}
	cases := []tc{
		{
			name:   "host override",
			host:   "example.com:443",
			expect: "example.com:443",
		},
		{
			name:   "bucket region env",
			bucket: "b",
			region: "r",
			expect: "b.r.aliyuncs.com:443",
		},
		{
			name:   "default common",
			expect: fmt.Sprintf("%s.%s.aliyuncs.com:443", common.OSSBucketName, common.OSSRegion),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setEnv(t, "OSS_HOST", c.host)
			setEnv(t, "OSS_BUCKET_NAME", c.bucket)
			setEnv(t, "OSS_REGION", c.region)
			if got := utils.OSSHost(); got != c.expect {
				t.Fatalf("unexpected host: %s", got)
			}
		})
	}
}

// TestOSSConnectivity 验证连通性检查成功场景。
func TestOSSConnectivity(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer l.Close()

	old := os.Getenv("OSS_HOST")
	_ = os.Setenv("OSS_HOST", l.Addr().String())
	t.Cleanup(func() {
		_ = os.Setenv("OSS_HOST", old)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := utils.OSSConnectivity(ctx); err != nil {
		t.Fatalf("connectivity failed: %v", err)
	}
}

// TestOSSConnectivity_Failure 验证连通性检查失败场景。
func TestOSSConnectivity_Failure(t *testing.T) {
	setEnv(t, "OSS_HOST", "127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := utils.OSSConnectivity(ctx); err == nil {
		t.Fatal("expected error")
	}
}

// TestOSSConnectivity_Timeout 验证连通性检查超时场景。
func TestOSSConnectivity_Timeout(t *testing.T) {
	setEnv(t, "OSS_HOST", "10.255.255.1:81")
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if err := utils.OSSConnectivity(ctx); err == nil {
		t.Fatal("expected timeout error")
	}
}

// TestOSSUploadDownloadDelete_Integration 验证 OSS 上传下载删除流程。
func TestOSSUploadDownloadDelete_Integration(t *testing.T) {
	accessKey := os.Getenv("OSS_ACCESS_KEY_ID")
	accessSecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
	region := os.Getenv("OSS_REGION")
	bucket := os.Getenv("OSS_BUCKET_NAME")
	if accessKey == "" || accessSecret == "" || region == "" || bucket == "" {
		t.Skip("oss env not set")
	}

	oldRegion := common.OSSRegion
	oldBucket := common.OSSBucketName
	common.OSSRegion = region
	common.OSSBucketName = bucket
	oldLoad := utils.OSSLoadEnv()
	oldKeyGen := utils.OSSKeyGen()
	oldUpload := utils.OSSUpload()
	key := fmt.Sprintf("test/%s.txt", utils.UUID())
	utils.SetOSSLoadEnv(func() error { return nil })
	utils.SetOSSKeyGen(func(originalFilename string) string { return key })

	t.Cleanup(func() {
		common.OSSRegion = oldRegion
		common.OSSBucketName = oldBucket
		utils.SetOSSLoadEnv(oldLoad)
		utils.SetOSSKeyGen(oldKeyGen)
		utils.SetOSSUpload(oldUpload)
	})

	_, filePath := testFilePath(t)
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open file failed: %v", err)
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}

	objectKey, err := utils.UploadToOSS(file, "test.txt")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if objectKey == "" {
		t.Fatal("empty object key")
	}

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion(region)
	client := oss.NewClient(cfg)

	getResp, err := client.GetObject(context.Background(), &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		t.Fatalf("get object failed: %v", err)
	}
	defer getResp.Body.Close()

	got, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("read object failed: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("object content mismatch")
	}

	_, err = client.DeleteObject(context.Background(), &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	})
	if err != nil {
		t.Fatalf("delete object failed: %v", err)
	}
}

// setEnv 设置环境变量并在测试结束时恢复。
func setEnv(t *testing.T, key, value string) {
	old, ok := os.LookupEnv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}

// testFilePath 获取测试文件路径。
func testFilePath(t *testing.T) (string, string) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller failed")
	}
	dir := filepath.Dir(file)
	path := filepath.Join(dir, "test.txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test file missing: %v", err)
	}
	return dir, path
}
