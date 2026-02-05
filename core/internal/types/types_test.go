package types

import "testing"

// TestHelperBuilders 验证请求构建辅助函数。
func TestHelperBuilders(t *testing.T) {
	listReq := NewUserFileListRequest(1, 2, 3)
	if listReq.Id != 1 || listReq.Page != 2 || listReq.Size != 3 {
		t.Fatalf("unexpected list req: %+v", listReq)
	}

	loginReq := NewLoginRequest("u", "p")
	if loginReq.Name != "u" || loginReq.Password != "p" {
		t.Fatalf("unexpected login req: %+v", loginReq)
	}
}
