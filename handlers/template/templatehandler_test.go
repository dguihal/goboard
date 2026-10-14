package template

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeSwagger_NotConfigured(t *testing.T) {
	h := NewTemplateHandler()

	req := httptest.NewRequest(http.MethodGet, "/swagger/swagger.yaml", nil)
	rr := httptest.NewRecorder()
	h.ServeSwagger(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when swagger dir not set, got %d", rr.Code)
	}
}

func TestServeSwagger_ServesFile(t *testing.T) {
	dir := t.TempDir()
	content := `swagger: "2.0"
host: "{{.Hostname}}"`
	if err := os.WriteFile(filepath.Join(dir, "swagger.yaml"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write swagger.yaml: %v", err)
	}

	h := NewTemplateHandler()
	h.SetSwaggerBaseDir(dir)

	req := httptest.NewRequest(http.MethodGet, "/swagger/swagger.yaml", nil)
	req.Host = "localhost:8080"
	rr := httptest.NewRecorder()
	h.ServeSwagger(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() == 0 {
		t.Error("expected non-empty swagger response body")
	}
}

func TestGetSwaggerOp_ReturnsCorrectPath(t *testing.T) {
	h := NewTemplateHandler()
	op := h.GetSwaggerOp()

	if op.RestPath != "/swagger/swagger.yaml" {
		t.Errorf("unexpected RestPath: %q", op.RestPath)
	}
	if op.Method != http.MethodGet {
		t.Errorf("expected GET method, got %q", op.Method)
	}
}

func TestSetWebUIBaseDir(t *testing.T) {
	h := NewTemplateHandler()
	if h.webuiBaseDirSet {
		t.Error("expected webuiBaseDirSet to be false initially")
	}
	h.SetWebUIBaseDir(t.TempDir())
	if !h.webuiBaseDirSet {
		t.Error("expected webuiBaseDirSet to be true after set")
	}
}

func TestSetSwaggerBaseDir(t *testing.T) {
	h := NewTemplateHandler()
	if h.swaggerBaseDirSet {
		t.Error("expected swaggerBaseDirSet to be false initially")
	}
	h.SetSwaggerBaseDir(t.TempDir())
	if !h.swaggerBaseDirSet {
		t.Error("expected swaggerBaseDirSet to be true after set")
	}
}
