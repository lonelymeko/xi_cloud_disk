package utils

import (
	"os"
	"strings"
	"testing"
)

// TestMd5 验证 Md5 计算结果。
func TestMd5(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "d41d8cd98f00b204e9800998ecf8427e"},
		{name: "abc", input: "abc", want: "900150983cd24fb0d6963f7d28e17f72"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Md5(c.input)
			if got != c.want {
				t.Fatalf("md5 mismatch: got %s want %s", got, c.want)
			}
		})
	}
}

// TestUUID 验证 UUID 格式与唯一性。
func TestUUID(t *testing.T) {
	a := UUID()
	b := UUID()
	if a == "" || b == "" {
		t.Fatal("uuid empty")
	}
	if a == b {
		t.Fatal("uuid not unique")
	}
	if len(a) != 36 || len(b) != 36 {
		t.Fatalf("uuid length unexpected: %d %d", len(a), len(b))
	}
	if !strings.Contains(a, "-") || !strings.Contains(b, "-") {
		t.Fatal("uuid format invalid")
	}
}

// TestTokenRoundTrip 验证 Token 生成与解析。
func TestTokenRoundTrip(t *testing.T) {
	payload := JwtPayLoad{Id: 1, Identity: "u-1", Name: "bob"}
	token, err := GenToken(payload, "secret", 1)
	if err != nil {
		t.Fatalf("gen token failed: %v", err)
	}
	claims, err := ParseToken(token, "secret", 1)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.Identity != payload.Identity || claims.Name != payload.Name || claims.Id != payload.Id {
		t.Fatalf("claims mismatch")
	}
}

// TestTokenInvalid 验证无效 Token 解析失败。
func TestTokenInvalid(t *testing.T) {
	_, err := ParseToken("invalid.token", "secret", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRandomPassword 验证随机密码生成。
func TestRandomPassword(t *testing.T) {
	value := randomPassword(16)
	if len(value) != 16 {
		t.Fatalf("length mismatch: %d", len(value))
	}
	allowed := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, r := range value {
		if !strings.ContainsRune(allowed, r) {
			t.Fatalf("invalid char: %q", r)
		}
	}
}

func TestOSSRegionValueNormalization(t *testing.T) {
	old, had := os.LookupEnv("OSS_REGION")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("OSS_REGION", old)
			return
		}
		_ = os.Unsetenv("OSS_REGION")
	})
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "cn-hangzhou", want: "cn-hangzhou"},
		{name: "prefixed", input: "oss-cn-hangzhou", want: "cn-hangzhou"},
		{name: "double-prefixed", input: "oss-oss-cn-hangzhou", want: "cn-hangzhou"},
		{name: "spaces", input: "  oss-cn-beijing  ", want: "cn-beijing"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_ = os.Setenv("OSS_REGION", c.input)
			got := OSSRegionValue()
			if got != c.want {
				t.Fatalf("normalize mismatch: got %s want %s", got, c.want)
			}
		})
	}
}

// BenchmarkMd5 基准测试 Md5。
func BenchmarkMd5(b *testing.B) {
	input := strings.Repeat("a", 128)
	for i := 0; i < b.N; i++ {
		_ = Md5(input)
	}
}
