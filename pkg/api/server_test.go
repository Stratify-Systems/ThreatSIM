package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validPolicy = `version: "1.0"
simulations:
  - name: health check
    request:
      method: GET
      path: /health
    expected:
      status_code: 200
`

func TestPolicyAPIReadCreateUpdateAndValidate(t *testing.T) {
	server := NewServer(t.TempDir())
	if err := os.MkdirAll(filepath.Join(server.RootDir, "tests", "simulations"), 0750); err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	body, _ := json.Marshal(map[string]string{"name": "editor.yaml", "content": validPolicy})
	response := request(http.MethodPost, "/api/policies", string(body))
	if response.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body)
	}

	response = request(http.MethodGet, "/api/policies/editor.yaml", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "health check") {
		t.Fatalf("read response = %d, body = %s", response.Code, response.Body)
	}

	response = request(http.MethodPost, "/api/policies/validate", `{"content":"not: [valid"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid validation status = %d", response.Code)
	}

	updated := strings.Replace(validPolicy, "/health", "/ready", 1)
	body, _ = json.Marshal(map[string]string{"content": updated})
	response = request(http.MethodPut, "/api/policies/editor.yaml", string(body))
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", response.Code, response.Body)
	}
	response = request(http.MethodGet, "/api/policies/editor.yaml", "")
	if !strings.Contains(response.Body.String(), "/ready") {
		t.Fatalf("updated policy was not persisted: %s", response.Body)
	}
}

func TestPolicyAPIRejectsTraversal(t *testing.T) {
	server := NewServer(t.TempDir())
	handler := server.Handler()
	body := bytes.NewBufferString(`{"content":"` + strings.ReplaceAll(validPolicy, `"`, `\"`) + `"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/policies/%2e%2e%2foutside.yaml", body)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("traversal status = %d, body = %s", recorder.Code, recorder.Body)
	}
}
