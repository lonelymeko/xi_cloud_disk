package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseOK(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	Response(req, rec, map[string]string{"k": "v"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
	var body Body
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body.Code != 0 || body.Msg != "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestResponseError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	Response(req, rec, nil, errors.New("boom"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: %d", rec.Code)
	}
	var body Body
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body.Code != 404 || body.Msg != "boom" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestDefaults(t *testing.T) {
	if OSSRegion == "" || OSSBucketName == "" || PageSize == 0 || DataTimeFormat == "" {
		t.Fatal("defaults invalid")
	}
}
