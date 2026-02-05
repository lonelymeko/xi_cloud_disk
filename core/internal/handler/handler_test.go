package handler

import (
	"cloud_disk/core/internal/svc"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlersParseError(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	invalidJSON := "{invalid}"
	methods := []struct {
		name    string
		method  string
		handler func(*svc.ServiceContext) http.HandlerFunc
	}{
		{name: "login", method: http.MethodPost, handler: LoginHandler},
		{name: "register", method: http.MethodPost, handler: RegisterHandler},
		{name: "userDetail", method: http.MethodPost, handler: UserDetailHandler},
		{name: "sendCode", method: http.MethodPost, handler: SendVerificationCodeHandler},
		{name: "changePassword", method: http.MethodPost, handler: ChangePasswordHandler},
		{name: "resetPassword", method: http.MethodPost, handler: ResetPasswordHandler},
		{name: "userFileList", method: http.MethodPost, handler: UserFileListHandler},
		{name: "userFileMove", method: http.MethodPut, handler: UserFileMoveHandler},
		{name: "userFileNameUpdate", method: http.MethodPost, handler: UserFileNameUpdateHandler},
		{name: "userFolderCreate", method: http.MethodPost, handler: UserFolderCreateHandler},
		{name: "userFolderDelete", method: http.MethodDelete, handler: UserFolderDeleteHandler},
		{name: "createShareRecord", method: http.MethodPost, handler: CreateShareRecordHandler},
		{name: "getShareRecord", method: http.MethodGet, handler: GetShareRecordHandler},
		{name: "saveResource", method: http.MethodPost, handler: SaveResourceHandler},
		{name: "uploadFile", method: http.MethodPost, handler: UploadFileHandler},
	}
	for _, h := range methods {
		t.Run(h.name, func(t *testing.T) {
			req := httptest.NewRequest(h.method, "/", strings.NewReader(invalidJSON))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.handler(svcCtx).ServeHTTP(rec, req)
			if rec.Code == http.StatusOK {
				t.Fatalf("expected error status")
			}
		})
	}
}

func TestUploadFileHandlerMissingFile(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	UploadFileHandler(svcCtx).ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status")
	}
}

func TestUploadFileHandlerParseError(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	UploadFileHandler(svcCtx).ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status")
	}
}

func TestHealthHandler(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(svcCtx).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodHead, "/health", nil)
	rec = httptest.NewRecorder()
	HealthHandler(svcCtx).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
}
