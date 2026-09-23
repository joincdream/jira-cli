package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"tools/jira/pkg"
)

// setupMockApp creates an App connected to a mock Jira server and returns call counters.
func setupMockApp(t *testing.T) (*App, *int32, *pkg.CreateIssueRequest) {
	var createCalled int32
	var lastCreateReq pkg.CreateIssueRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue"):
			atomic.AddInt32(&createCalled, 1)
			var req pkg.CreateIssueRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				lastCreateReq = req
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(pkg.CreateIssueResponse{
				ID:   "10001",
				Key:  "JC-100",
				Self: "https://mock.atlassian.net/rest/api/3/issue/10001",
			})
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/JC-100"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pkg.Issue{
				Key: "JC-100",
				Fields: pkg.IssueFields{
					Summary: "Mock Issue",
					Status:  pkg.Status{Name: "To Do"},
				},
			})
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	clientProvider := func() (*pkg.Client, error) {
		cfg := pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "dummy_token",
			ProjectKey:  "JC",
		}
		return pkg.NewClientWithConfig(cfg), nil
	}

	mockCatalogLoader := func() (*pkg.LabelCatalog, error) {
		return &pkg.LabelCatalog{
			Version:    "1.0.0",
			Categories: map[string]pkg.LabelCategory{},
		}, nil
	}

	app := NewApp("1.5.0", clientProvider, mockCatalogLoader)
	return app, &createCalled, &lastCreateReq
}

func TestJC8_CLI_Parsing_And_Guardrails(t *testing.T) {
	// TC-01: jira create --help -> Exit 0, usage in stdout, 0 API calls
	t.Run("TC-01: create --help exits 0 without calling API", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "--help"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "사용법: jira create <SUMMARY>") {
			t.Errorf("expected usage in stdout, got: %s", stdout.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("expected 0 create API calls, but got %d", atomic.LoadInt32(createCalled))
		}
	})

	// TC-02: jira create -h -> Exit 0, usage in stdout, 0 API calls
	t.Run("TC-02: create -h exits 0 without calling API", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "-h"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "사용법: jira create <SUMMARY>") {
			t.Errorf("expected usage in stdout, got: %s", stdout.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("expected 0 create API calls, but got %d", atomic.LoadInt32(createCalled))
		}
	})

	// TC-03: Order independence (flag before positional)
	t.Run("TC-03: flag before positional argument", func(t *testing.T) {
		app, createCalled, lastReq := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "-d", "상세 설명 본문입니다", "순서가 바뀐 유효한 제목"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 1 {
			t.Fatalf("expected 1 API call, got %d", atomic.LoadInt32(createCalled))
		}
		if lastReq.Fields.Summary != "순서가 바뀐 유효한 제목" {
			t.Errorf("expected summary '순서가 바뀐 유효한 제목', got %q", lastReq.Fields.Summary)
		}
	})

	// TC-04: Order independence (positional before flag)
	t.Run("TC-04: positional argument before flag", func(t *testing.T) {
		app, createCalled, lastReq := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "기존 방식의 유효한 제목", "-d", "상세 설명 본문입니다"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 1 {
			t.Fatalf("expected 1 API call, got %d", atomic.LoadInt32(createCalled))
		}
		if lastReq.Fields.Summary != "기존 방식의 유효한 제목" {
			t.Errorf("expected summary '기존 방식의 유효한 제목', got %q", lastReq.Fields.Summary)
		}
	})

	// TC-05: Double dash literal protection
	t.Run("TC-05: double dash protects literal argument starting with hyphen", func(t *testing.T) {
		app, createCalled, lastReq := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "-d", "상세 설명 내용", "--", "-d 플래그 버그 수정 작업"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 1 {
			t.Fatalf("expected 1 API call, got %d", atomic.LoadInt32(createCalled))
		}
		if lastReq.Fields.Summary != "-d 플래그 버그 수정 작업" {
			t.Errorf("expected literal summary '-d 플래그 버그 수정 작업', got %q", lastReq.Fields.Summary)
		}
	})

	// TC-06: Guardrail - Missing summary
	t.Run("TC-06: guardrail fails when summary is missing", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "이슈 제목(Summary)을 입력해야 합니다") {
			t.Errorf("expected error about missing summary, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-07: Guardrail - Summary too short (< 5 chars)
	t.Run("TC-07: guardrail fails when summary is under 5 characters", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "짧음", "-d", "본문"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "최소 5자 이상") {
			t.Errorf("expected error about minimum 5 chars, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-08: Guardrail - Whitespace-only summary
	t.Run("TC-08: guardrail fails when summary is only whitespace", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "     ", "-d", "본문"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "최소 5자 이상") {
			t.Errorf("expected error about minimum 5 chars, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-09: Guardrail - Summary starting with flag prefix without double dash
	t.Run("TC-09: guardrail fails when summary starts with hyphen without double dash", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "-invalid-summary", "-d", "본문 설명입니다"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "flag provided but not defined") && !strings.Contains(stderr.String(), "플래그 형식") {
			t.Errorf("expected error about flag prefix or undefined flag, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-10: Guardrail - Missing description
	t.Run("TC-10: guardrail fails when description is missing without allow flag", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "충분히 긴 유효한 제목입니다"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "상세 설명(-d, --desc)이 입력되지 않았습니다") {
			t.Errorf("expected error about missing description, got: %s", stderr.String())
		}
		if !strings.Contains(stderr.String(), "--allow-empty-desc") {
			t.Errorf("expected guidance on --allow-empty-desc, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-11: Guardrail - Allow empty description
	t.Run("TC-11: create succeeds without description when --allow-empty-desc is passed", func(t *testing.T) {
		app, createCalled, lastReq := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "--allow-empty-desc", "설명 없는 간이 티켓 생성"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 1 {
			t.Fatalf("expected 1 API call, got %d", atomic.LoadInt32(createCalled))
		}
		if lastReq.Fields.Summary != "설명 없는 간이 티켓 생성" {
			t.Errorf("expected summary, got %q", lastReq.Fields.Summary)
		}
	})

	// TC-12: Guardrail - Invalid due date format
	t.Run("TC-12: guardrail fails with invalid due date format", func(t *testing.T) {
		app, createCalled, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "유효한 제목입니다", "-d", "본문", "--due", "2026/12/31"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "YYYY-MM-DD") {
			t.Errorf("expected error about date format, got: %s", stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 0 {
			t.Errorf("API should not be called")
		}
	})

	// TC-13: Guardrail - Valid due date format
	t.Run("TC-13: create succeeds with valid due date format", func(t *testing.T) {
		app, createCalled, lastReq := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"create", "유효한 제목입니다", "-d", "본문", "--due", "2026-12-31"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(createCalled) != 1 {
			t.Fatalf("expected 1 API call, got %d", atomic.LoadInt32(createCalled))
		}
		if lastReq.Fields.DueDate != "2026-12-31" {
			t.Errorf("expected due date '2026-12-31', got %q", lastReq.Fields.DueDate)
		}
	})

	// TC-14: Regression - edit --help exits 0
	t.Run("TC-14: edit --help exits 0 without calling API", func(t *testing.T) {
		app, _, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"edit", "--help"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "사용법: jira edit <KEY>") {
			t.Errorf("expected usage in stdout, got: %s", stdout.String())
		}
	})

	// TC-15: Regression - get --help exits 0
	t.Run("TC-15: get --help exits 0 without calling API", func(t *testing.T) {
		app, _, _ := setupMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"get", "--help"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "사용법: jira get <KEY>") {
			t.Errorf("expected usage in stdout, got: %s", stdout.String())
		}
	})
}
