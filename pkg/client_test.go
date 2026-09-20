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

func TestReadProfileConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	content := `
# Sample Jira Configuration
[default]
instance_url = https://default.atlassian.net
email = default@example.com
api_token = token_default_123
project_key = DEF

; Secondary profile
[work]
instance_url = "https://work.atlassian.net"
email = 'work@example.com'
api_token = token_work_456
project_key = WORK
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	t.Run("read default profile", func(t *testing.T) {
		cfg, found, err := ReadProfileConfig(configPath, "default")
		if err != nil {
			t.Fatalf("ReadProfileConfig failed: %v", err)
		}
		if !found {
			t.Fatal("expected profile 'default' to be found")
		}
		if cfg.InstanceURL != "https://default.atlassian.net" {
			t.Errorf("expected URL https://default.atlassian.net, got %q", cfg.InstanceURL)
		}
		if cfg.Email != "default@example.com" {
			t.Errorf("expected email default@example.com, got %q", cfg.Email)
		}
		if cfg.APIToken != "token_default_123" {
			t.Errorf("expected token token_default_123, got %q", cfg.APIToken)
		}
		if cfg.ProjectKey != "DEF" {
			t.Errorf("expected project DEF, got %q", cfg.ProjectKey)
		}
	})

	t.Run("read work profile case-insensitively", func(t *testing.T) {
		cfg, found, err := ReadProfileConfig(configPath, "WORK")
		if err != nil {
			t.Fatalf("ReadProfileConfig failed: %v", err)
		}
		if !found {
			t.Fatal("expected profile 'WORK' to be found")
		}
		if cfg.InstanceURL != "https://work.atlassian.net" {
			t.Errorf("expected URL https://work.atlassian.net, got %q", cfg.InstanceURL)
		}
		if cfg.Email != "work@example.com" {
			t.Errorf("expected email work@example.com, got %q", cfg.Email)
		}
		if cfg.APIToken != "token_work_456" {
			t.Errorf("expected token token_work_456, got %q", cfg.APIToken)
		}
		if cfg.ProjectKey != "WORK" {
			t.Errorf("expected project WORK, got %q", cfg.ProjectKey)
		}
	})

	t.Run("profile not found", func(t *testing.T) {
		_, found, err := ReadProfileConfig(configPath, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Fatal("expected profile to not be found")
		}
	})
}

func TestFindLocalProfile(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "src", "app")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}

	// Create .jira-profile in root
	profileFile := filepath.Join(tmpDir, ".jira-profile")
	if err := os.WriteFile(profileFile, []byte("# comment\nmy-project-profile\n"), 0644); err != nil {
		t.Fatalf("failed to write .jira-profile: %v", err)
	}

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	found := FindLocalProfile()
	if found != "my-project-profile" {
		t.Errorf("expected 'my-project-profile', got %q", found)
	}

	// Test GetActiveProfile resolution: local profile takes precedence over default
	os.Unsetenv("JIRA_PROFILE")
	SetActiveProfile("") // reset
	if prof := GetActiveProfile(); prof != "my-project-profile" {
		t.Errorf("expected GetActiveProfile() to return 'my-project-profile', got %q", prof)
	}

	// Environment variable takes precedence over local profile
	t.Setenv("JIRA_PROFILE", "env-profile")
	if prof := GetActiveProfile(); prof != "env-profile" {
		t.Errorf("expected GetActiveProfile() to return 'env-profile', got %q", prof)
	}

	// Explicitly set active profile takes top precedence
	SetActiveProfile("flag-profile")
	if prof := GetActiveProfile(); prof != "flag-profile" {
		t.Errorf("expected GetActiveProfile() to return 'flag-profile', got %q", prof)
	}
	SetActiveProfile("") // reset
}


func TestWriteProfileConfig_And_ListProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// 1. Write initial default profile
	err := WriteProfileConfig(configPath, "default", Config{
		InstanceURL: "https://default.atlassian.net",
		Email:       "default@example.com",
		APIToken:    "token1",
		ProjectKey:  "DEF",
	})
	if err != nil {
		t.Fatalf("failed to write default profile: %v", err)
	}

	// 2. Add second profile
	err = WriteProfileConfig(configPath, "cloit", Config{
		InstanceURL: "https://cloit.atlassian.net",
		Email:       "cloit@example.com",
		APIToken:    "token2",
		ProjectKey:  "CLOIT",
	})
	if err != nil {
		t.Fatalf("failed to write cloit profile: %v", err)
	}

	// Verify ListProfiles
	profiles, err := ListProfiles(configPath)
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 2 || profiles[0] != "default" || profiles[1] != "cloit" {
		t.Errorf("unexpected profiles: %v", profiles)
	}

	// 3. Update default profile
	err = WriteProfileConfig(configPath, "default", Config{
		InstanceURL: "https://new-default.atlassian.net",
		Email:       "new@example.com",
		APIToken:    "token_new",
		ProjectKey:  "NEWDEF",
	})
	if err != nil {
		t.Fatalf("failed to update default profile: %v", err)
	}

	// Check updated default
	cfgDef, found, err := ReadProfileConfig(configPath, "default")
	if err != nil || !found {
		t.Fatalf("failed to read default: %v, found: %v", err, found)
	}
	if cfgDef.InstanceURL != "https://new-default.atlassian.net" {
		t.Errorf("expected updated URL, got %s", cfgDef.InstanceURL)
	}

	// Check cloit is still preserved
	cfgCloit, found, err := ReadProfileConfig(configPath, "cloit")
	if err != nil || !found {
		t.Fatalf("failed to read cloit: %v, found: %v", err, found)
	}
	if cfgCloit.InstanceURL != "https://cloit.atlassian.net" {
		t.Errorf("expected cloit URL to be preserved, got %s", cfgCloit.InstanceURL)
	}
}

func TestLoadConfigFileForProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	err := WriteProfileConfig(configPath, "default", Config{
		InstanceURL: "https://myjira.atlassian.net",
		Email:       "my@example.com",
		APIToken:    "my_token",
		ProjectKey:  "PROJ",
	})
	if err != nil {
		t.Fatalf("WriteProfileConfig failed: %v", err)
	}

	t.Run("load valid profile", func(t *testing.T) {
		cfg, err := LoadConfigFileForProfile(configPath, "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.InstanceURL != "https://myjira.atlassian.net" {
			t.Errorf("expected instance URL https://myjira.atlassian.net, got %s", cfg.InstanceURL)
		}
	})

	t.Run("load missing profile", func(t *testing.T) {
		_, err := LoadConfigFileForProfile(configPath, "unknown")
		if err == nil {
			t.Fatal("expected error for unknown profile, got nil")
		}
		if !strings.Contains(err.Error(), "Jira 프로필 [unknown]을(를) 찾을 수 없습니다") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
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
