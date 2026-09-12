package pkg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := `
# Sample comment
JIRA_INSTANCE_URL="https://test.atlassian.net"
JIRA_EMAIL='test@example.com'
JIRA_API_TOKEN=secret_token_123
JIRA_PROJECT_KEY=TEST

# Invalid line
INVALID_LINE_WITHOUT_EQUALS
EMPTY_VAL=
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env file: %v", err)
	}

	envMap := parseEnvFileToMap(envPath)

	if envMap["JIRA_INSTANCE_URL"] != "https://test.atlassian.net" {
		t.Errorf("expected URL https://test.atlassian.net, got %q", envMap["JIRA_INSTANCE_URL"])
	}
	if envMap["JIRA_EMAIL"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %q", envMap["JIRA_EMAIL"])
	}
	if envMap["JIRA_API_TOKEN"] != "secret_token_123" {
		t.Errorf("expected token secret_token_123, got %q", envMap["JIRA_API_TOKEN"])
	}
	if envMap["JIRA_PROJECT_KEY"] != "TEST" {
		t.Errorf("expected key TEST, got %q", envMap["JIRA_PROJECT_KEY"])
	}
}

func TestReadJiraConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".jira.json")

	content := `{
		"instance_url": "https://local.atlassian.net",
		"email": "local@example.com",
		"api_token": "local_token_123",
		"project_key": "LOCALPROJ"
	}`

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .jira.json file: %v", err)
	}

	cfg, ok := readJiraConfigFile(configPath)
	if !ok {
		t.Fatal("expected readJiraConfigFile to succeed, got false")
	}

	if cfg.InstanceURL != "https://local.atlassian.net" {
		t.Errorf("expected InstanceURL https://local.atlassian.net, got %q", cfg.InstanceURL)
	}
	if cfg.Email != "local@example.com" {
		t.Errorf("expected Email local@example.com, got %q", cfg.Email)
	}
	if cfg.APIToken != "local_token_123" {
		t.Errorf("expected APIToken local_token_123, got %q", cfg.APIToken)
	}
	if cfg.ProjectKey != "LOCALPROJ" {
		t.Errorf("expected ProjectKey LOCALPROJ, got %q", cfg.ProjectKey)
	}
}

func TestFormatJiraAPIError(t *testing.T) {
	t.Run("structured error response", func(t *testing.T) {
		jsonErr := `{
			"errorMessages": ["Issue does not exist or you do not have permission to see it."],
			"errors": {"summary": "Summary is required."}
		}`
		err := formatJiraAPIError(400, []byte(jsonErr))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "Issue does not exist") || !strings.Contains(errMsg, "summary: Summary is required.") {
			t.Errorf("unexpected error message format: %s", errMsg)
		}
	})

	t.Run("unstructured raw error", func(t *testing.T) {
		rawErr := `<html>502 Bad Gateway</html>`
		err := formatJiraAPIError(502, []byte(rawErr))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "502 Bad Gateway") {
			t.Errorf("unexpected error message format: %s", err.Error())
		}
	})
}

// ----------------------------------------------------------------------------
// HTTP Client Unit Tests using Go standard httptest.Server (Zero Abstraction)
// ----------------------------------------------------------------------------

func setupTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	cfg := Config{
		InstanceURL: server.URL,
		Email:       "test@example.com",
		APIToken:    "mock_token",
		ProjectKey:  "TEST",
	}
	client := NewClientWithConfig(cfg)
	return client, server
}

func TestClient_ListIssues(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "test@example.com" || pass != "mock_token" {
			t.Errorf("invalid basic auth: user=%s, pass=%s", user, pass)
		}

		resp := SearchResponse{
			Issues: []Issue{
				{
					Key: "TEST-1",
					Fields: IssueFields{
						Summary: "Test Issue 1",
						Status:  Status{Name: "진행 중"},
						IssueType: IssueType{Name: "작업"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	issues, err := client.ListIssues(context.Background(), "")
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Key != "TEST-1" || issues[0].Fields.Summary != "Test Issue 1" {
		t.Errorf("unexpected issue data: %+v", issues[0])
	}
}

func TestClient_GetIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/rest/api/3/issue/TEST-10" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			issue := Issue{
				Key: "TEST-10",
				Fields: IssueFields{
					Summary: "Detail Test",
					Status:  Status{Name: "해야 할 일"},
				},
			}
			json.NewEncoder(w).Encode(issue)
		})

		client, server := setupTestClient(t, handler)
		defer server.Close()

		issue, err := client.GetIssue(context.Background(), "TEST-10")
		if err != nil {
			t.Fatalf("GetIssue failed: %v", err)
		}
		if issue.Key != "TEST-10" || issue.Fields.Summary != "Detail Test" {
			t.Errorf("unexpected issue: %+v", issue)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"errorMessages": ["Issue does not exist."]}`))
		})

		client, server := setupTestClient(t, handler)
		defer server.Close()

		_, err := client.GetIssue(context.Background(), "INVALID-99")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Issue does not exist") {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})
}

func TestClient_CreateIssue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/rest/api/3/issue" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var req CreateIssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode create request: %v", err)
		}

		if req.Fields.Project.Key != "TEST" || req.Fields.Summary != "New Task" {
			t.Errorf("unexpected fields: %+v", req.Fields)
		}
		if req.Fields.DueDate != "2026-09-01" {
			t.Errorf("unexpected due date: %s", req.Fields.DueDate)
		}
		if len(req.Fields.Labels) != 2 || req.Fields.Labels[0] != "AI" {
			t.Errorf("unexpected labels: %v", req.Fields.Labels)
		}

		resp := CreateIssueResponse{
			ID:  "10001",
			Key: "TEST-100",
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	resp, err := client.CreateIssue(context.Background(), "TEST", "New Task", "Detailed task desc", "작업", "2026-09-01", "", []string{"AI", "Planning"})
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if resp.Key != "TEST-100" {
		t.Errorf("expected Key TEST-100, got %s", resp.Key)
	}
}

func TestClient_UpdateIssue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/rest/api/3/issue/TEST-100" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		fields, ok := payload["fields"].(map[string]interface{})
		if !ok {
			t.Fatalf("missing fields in payload: %v", payload)
		}

		if fields["summary"] != "Updated Title" {
			t.Errorf("unexpected summary: %v", fields["summary"])
		}
		if fields["duedate"] != nil {
			t.Errorf("expected null duedate, got: %v", fields["duedate"])
		}

		w.WriteHeader(http.StatusNoContent)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	newSummary := "Updated Title"
	removeDueDate := "none"
	err := client.UpdateIssue(context.Background(), "TEST-100", UpdateIssueOptions{
		Summary: &newSummary,
		DueDate: &removeDueDate,
	})

	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}
}

func TestClient_TransitionIssue(t *testing.T) {
	step := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/TEST-100/transitions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method == "GET" {
			step++
			resp := TransitionsResponse{
				Transitions: []Transition{
					{
						ID:   "11",
						Name: "진행 중",
						To:   Status{Name: "진행 중"},
					},
					{
						ID:   "21",
						Name: "완료",
						To:   Status{Name: "완료"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "POST" {
			step++
			var payload map[string]map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["transition"]["id"] != "11" {
				t.Errorf("expected transition id 11, got %s", payload["transition"]["id"])
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	err := client.TransitionIssue(context.Background(), "TEST-100", "진행 중")
	if err != nil {
		t.Fatalf("TransitionIssue failed: %v", err)
	}

	if step != 2 {
		t.Errorf("expected 2 steps (GET transitions then POST), got %d", step)
	}
}

func TestClient_AddComment(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/rest/api/3/issue/TEST-100/comment" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	err := client.AddComment(context.Background(), "TEST-100", "테스트 코멘트입니다.")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
}

func TestClient_DeleteIssue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || !strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/TEST-100") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("deleteSubtasks") != "true" {
			t.Errorf("expected deleteSubtasks=true, got %s", r.URL.Query().Get("deleteSubtasks"))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	err := client.DeleteIssue(context.Background(), "TEST-100", true)
	if err != nil {
		t.Fatalf("DeleteIssue failed: %v", err)
	}
}
