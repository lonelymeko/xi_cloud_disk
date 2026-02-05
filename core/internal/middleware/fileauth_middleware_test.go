package middleware

import (
	"cloud_disk/core/utils"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFileAuthMiddlewareMissingToken(t *testing.T) {
	m := NewFileAuthMiddleware("s", 3600)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	m.Handle(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status")
	}
}

func TestFileAuthMiddlewareInvalidToken(t *testing.T) {
	m := NewFileAuthMiddleware("s", 3600)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	rec := httptest.NewRecorder()
	m.Handle(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status")
	}
}

func TestFileAuthMiddlewareAuthorizationOK(t *testing.T) {
	secret := "secret"
	expire := int64(3600)
	token, err := utils.GenToken(utils.JwtPayLoad{Id: 1, Identity: "u-1", Name: "n"}, secret, expire)
	if err != nil {
		t.Fatalf("token gen failed: %v", err)
	}

	m := NewFileAuthMiddleware(secret, expire)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	called := false
	m.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Context().Value("user_id") == nil || r.Context().Value("user_identity") == nil || r.Context().Value("user_name") == nil {
			t.Fatal("missing ctx values")
		}
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)
	if !called {
		t.Fatal("next not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
}

func TestFileAuthMiddlewareXTokenOK(t *testing.T) {
	secret := "secret"
	expire := int64(3600)
	token, err := utils.GenToken(utils.JwtPayLoad{Id: 1, Identity: "u-1", Name: "n"}, secret, expire)
	if err != nil {
		t.Fatalf("token gen failed: %v", err)
	}

	m := NewFileAuthMiddleware(secret, expire)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Token", token)
	rec := httptest.NewRecorder()
	called := false
	m.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)
	if !called {
		t.Fatal("next not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
}

func TestFileAuthMiddlewareQueryTokenOK(t *testing.T) {
	secret := "secret"
	expire := int64(3600)
	token, err := utils.GenToken(utils.JwtPayLoad{Id: 1, Identity: "u-1", Name: "n"}, secret, expire)
	if err != nil {
		t.Fatalf("token gen failed: %v", err)
	}

	m := NewFileAuthMiddleware(secret, expire)
	req := httptest.NewRequest(http.MethodGet, "/?token="+token, nil)
	rec := httptest.NewRecorder()
	called := false
	m.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)
	if !called {
		t.Fatal("next not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
}
