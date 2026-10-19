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

func TestListCommand_FilterFlags(t *testing.T) {
	var capturedJQL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			capturedJQL = r.URL.Query().Get("jql")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"issues": []}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	clientProvider := func() (*pkg.Client, error) {
		cfg := pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "token",
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

	t.Run("list with --all and --due <=today", func(t *testing.T) {
		capturedJQL = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "--all", "--due", "<=today"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if strings.Contains(capturedJQL, "project =") {
			t.Errorf("expected no project clause with --all, got %q", capturedJQL)
		}
		if !strings.Contains(capturedJQL, "duedate <= endOfDay()") {
			t.Errorf("expected duedate clause in JQL, got %q", capturedJQL)
		}
		if !strings.Contains(capturedJQL, "statusCategory != Done") {
			t.Errorf("expected default open status in JQL, got %q", capturedJQL)
		}
	})

	t.Run("list with --assignee me and default project", func(t *testing.T) {
		capturedJQL = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "--assignee", "me"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(capturedJQL, "project = JC") {
			t.Errorf("expected project = JC, got %q", capturedJQL)
		}
		if !strings.Contains(capturedJQL, "assignee = currentUser()") {
			t.Errorf("expected assignee = currentUser(), got %q", capturedJQL)
		}
	})

	t.Run("list with invalid due date returns error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "--due", "not-a-date"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected code 1 for invalid due filter, got %d", code)
		}
		if !strings.Contains(stderr.String(), "지원하지 않는 마감일(--due) 형식") {
			t.Errorf("expected error in stderr, got: %s", stderr.String())
		}
	})

	t.Run("raw JQL positional argument takes precedence", func(t *testing.T) {
		capturedJQL = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "project = CUSTOM_PROJ"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if capturedJQL != "project = CUSTOM_PROJ" {
			t.Errorf("expected raw JQL 'project = CUSTOM_PROJ', got %q", capturedJQL)
		}
	})
}

func TestListCommand_DetailFlag(t *testing.T) {
	var capturedFields string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			capturedFields = r.URL.Query().Get("fields")
			w.WriteHeader(http.StatusOK)
			mockIssues := []pkg.Issue{
				{
					Key: "JC-10",
					Fields: pkg.IssueFields{
						Summary:     "Detailed Issue 10",
						Description: "Full description text here",
						Status:      pkg.Status{Name: "진행 중"},
						IssueType:   pkg.IssueType{Name: "Task"},
						Project:     pkg.Project{Key: "JC", Name: "Jira Client"},
					},
				},
			}
			json.NewEncoder(w).Encode(pkg.SearchResponse{Issues: mockIssues})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	clientProvider := func() (*pkg.Client, error) {
		cfg := pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "token",
			ProjectKey:  "JC",
		}
		return pkg.NewClientWithConfig(cfg), nil
	}

	app := NewApp("1.5.0", clientProvider, nil)

	t.Run("without --detail", func(t *testing.T) {
		capturedFields = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "--md"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if strings.Contains(capturedFields, "description") {
			t.Errorf("expected no description in fields, got %s", capturedFields)
		}
		if !strings.Contains(stdout.String(), "| Key | Type | Status |") {
			t.Errorf("expected table markdown, got: %s", stdout.String())
		}
	})

	t.Run("with --detail --md", func(t *testing.T) {
		capturedFields = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "--detail", "--md"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(capturedFields, "description") || !strings.Contains(capturedFields, "comment") {
			t.Errorf("expected description and comment in fields, got %s", capturedFields)
		}
		if !strings.Contains(stdout.String(), "# [JC-10] Detailed Issue 10") {
			t.Errorf("expected detailed markdown header, got: %s", stdout.String())
		}
		if !strings.Contains(stdout.String(), "Full description text here") {
			t.Errorf("expected description in output, got: %s", stdout.String())
		}
	})

	t.Run("with -d alias and --json", func(t *testing.T) {
		capturedFields = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"list", "-d", "--json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if !strings.Contains(capturedFields, "description") {
			t.Errorf("expected description in fields, got %s", capturedFields)
		}
		if !strings.Contains(stdout.String(), `"description": "Full description text here"`) {
			t.Errorf("expected description in JSON, got: %s", stdout.String())
		}
	})
}

func TestJC14_GitBranchContextBinding(t *testing.T) {
	var requestedIssueKey string
	var transitionIssueKey string
	var commentIssueKey string
	var assignIssueKey, assignAccountID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/api/3/myself":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pkg.User{
				AccountID:   "acc-me-123",
				DisplayName: "Current User",
				EmailAddress: "me@example.com",
			})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") && strings.HasSuffix(r.URL.Path, "/transitions"):
			parts := strings.Split(r.URL.Path, "/")
			requestedIssueKey = parts[5]
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"transitions": []map[string]interface{}{
					{"id": "21", "name": "진행 중", "to": map[string]interface{}{"name": "진행 중"}},
				},
			})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") && strings.HasSuffix(r.URL.Path, "/transitions"):
			parts := strings.Split(r.URL.Path, "/")
			transitionIssueKey = parts[5]
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") && strings.HasSuffix(r.URL.Path, "/comment"):
			parts := strings.Split(r.URL.Path, "/")
			commentIssueKey = parts[5]
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "1001"})
		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") && strings.HasSuffix(r.URL.Path, "/assignee"):
			parts := strings.Split(r.URL.Path, "/")
			assignIssueKey = parts[5]
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if acc, ok := body["accountId"].(string); ok {
				assignAccountID = acc
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/"):
			parts := strings.Split(r.URL.Path, "/")
			requestedIssueKey = parts[5]
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pkg.Issue{
				Key: requestedIssueKey,
				Fields: pkg.IssueFields{
					Summary: "Mock Issue for " + requestedIssueKey,
					Status:  pkg.Status{Name: "To Do"},
				},
			})
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	clientProvider := func() (*pkg.Client, error) {
		cfg := pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "token",
			ProjectKey:  "JC",
		}
		return pkg.NewClientWithConfig(cfg), nil
	}

	app := NewApp("1.5.0", clientProvider, nil)

	// Save original resolver
	origResolver := pkg.GitBranchResolver
	defer func() { pkg.GitBranchResolver = origResolver }()

	// 1. Mock branch with ticket key
	pkg.GitBranchResolver = func(ctx context.Context) (string, error) {
		return "feature/JC-14-git-context-binding", nil
	}

	t.Run("zero-argument jira get uses branch issue key", func(t *testing.T) {
		requestedIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"get", "--md"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if requestedIssueKey != "JC-14" {
			t.Errorf("expected requested key JC-14, got %s", requestedIssueKey)
		}
		if !strings.Contains(stdout.String(), "# [JC-14]") {
			t.Errorf("expected markdown header for JC-14, got: %s", stdout.String())
		}
	})

	t.Run("explicit key argument overrides branch key in get", func(t *testing.T) {
		requestedIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"get", "JC-99", "--md"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if requestedIssueKey != "JC-99" {
			t.Errorf("expected requested key JC-99, got %s", requestedIssueKey)
		}
	})

	t.Run("zero-argument jira transitions uses branch key", func(t *testing.T) {
		requestedIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"transitions"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if requestedIssueKey != "JC-14" {
			t.Errorf("expected requested key JC-14, got %s", requestedIssueKey)
		}
	})

	t.Run("jira move <STATUS> with zero key uses branch key", func(t *testing.T) {
		transitionIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"move", "진행 중"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if transitionIssueKey != "JC-14" {
			t.Errorf("expected transition key JC-14, got %s", transitionIssueKey)
		}
	})

	t.Run("jira move <KEY> <STATUS> explicit key takes precedence", func(t *testing.T) {
		transitionIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"move", "JC-77", "진행 중"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if transitionIssueKey != "JC-77" {
			t.Errorf("expected transition key JC-77, got %s", transitionIssueKey)
		}
	})

	t.Run("jira comment <MSG> with zero key uses branch key", func(t *testing.T) {
		commentIssueKey = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"comment", "코드 구현 완료했습니다."}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if commentIssueKey != "JC-14" {
			t.Errorf("expected comment key JC-14, got %s", commentIssueKey)
		}
	})

	t.Run("jira assign me with zero key uses branch key", func(t *testing.T) {
		assignIssueKey = ""
		assignAccountID = ""
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"assign", "me"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d. stderr: %s", code, stderr.String())
		}
		if assignIssueKey != "JC-14" {
			t.Errorf("expected assign key JC-14, got %s", assignIssueKey)
		}
		if assignAccountID != "acc-me-123" {
			t.Errorf("expected assign account ID acc-me-123, got %s", assignAccountID)
		}
	})

	// 2. Mock branch with NO ticket key (e.g. main)
	pkg.GitBranchResolver = func(ctx context.Context) (string, error) {
		return "main", nil
	}

	t.Run("jira get on main branch without key returns error with guidance", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := app.Run(context.Background(), []string{"get"}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("expected non-zero exit code on main branch without key, got 0")
		}
		if !strings.Contains(stderr.String(), "감지하지 못했습니다") {
			t.Errorf("expected guidance message in stderr, got: %s", stderr.String())
		}
	})
}

