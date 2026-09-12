package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tools/jira/pkg"
)

func TestApp_Run_NoArgs(t *testing.T) {
	app := NewDefaultApp()
	var stdout, stderr bytes.Buffer

	code := app.Run(context.Background(), []string{}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if !strings.Contains(stdout.String(), "Jira CLI - Personal & Team Task Management Tool") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}
}

func TestApp_Run_Help(t *testing.T) {
	app := NewDefaultApp()

	tests := []string{"help", "-h", "--help"}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := app.Run(context.Background(), []string{cmd}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if !strings.Contains(stdout.String(), "Jira CLI - Personal & Team Task Management Tool") {
				t.Errorf("expected usage in stdout, got: %s", stdout.String())
			}
		})
	}
}

func TestConfigureCommand(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalWd) }()

	input := "https://custom.atlassian.net\ntestuser@example.com\nsecret_token_abc\nMYPROJ\n"
	cmd := NewConfigureCommand(strings.NewReader(input))

	var stdout, stderr bytes.Buffer
	err := cmd.Execute(context.Background(), []string{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected configure to succeed, got error: %v", err)
	}

	savedData, err := os.ReadFile(".jira.json")
	if err != nil {
		t.Fatalf("failed to read created .jira.json: %v", err)
	}

	var saved configData
	if err := json.Unmarshal(savedData, &saved); err != nil {
		t.Fatalf("failed to parse created .jira.json: %v", err)
	}

	if saved.InstanceURL != "https://custom.atlassian.net" {
		t.Errorf("expected URL https://custom.atlassian.net, got %q", saved.InstanceURL)
	}
	if saved.Email != "testuser@example.com" {
		t.Errorf("expected Email testuser@example.com, got %q", saved.Email)
	}
	if saved.APIToken != "secret_token_abc" {
		t.Errorf("expected Token secret_token_abc, got %q", saved.APIToken)
	}
	if saved.ProjectKey != "MYPROJ" {
		t.Errorf("expected ProjectKey MYPROJ, got %q", saved.ProjectKey)
	}
}

func TestApp_Run_Version(t *testing.T) {
	app := NewDefaultApp()

	tests := []string{"version", "-v", "--version"}
	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := app.Run(context.Background(), []string{cmd}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if !strings.Contains(stdout.String(), "jira version") {
				t.Errorf("expected version output, got: %s", stdout.String())
			}
		})
	}
}

func TestApp_Run_UnknownCommand(t *testing.T) {
	app := NewDefaultApp()
	var stdout, stderr bytes.Buffer

	code := app.Run(context.Background(), []string{"unknown-cmd"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	if !strings.Contains(stderr.String(), "알 수 없는 명령어: unknown-cmd") {
		t.Errorf("expected unknown command error, got: %s", stderr.String())
	}
}

func TestApp_Run_Labels(t *testing.T) {
	mockLoader := func() (*pkg.LabelCatalog, error) {
		return &pkg.LabelCatalog{
			Version: "1.0.0",
			Categories: map[string]pkg.LabelCategory{
				"project": {
					Name: "프로젝트",
					Labels: []pkg.LabelItem{
						{Name: "DIVE", Description: "DIVE MCP 플랫폼"},
					},
				},
			},
		}, nil
	}

	app := NewApp("1.5.0", nil, mockLoader)
	var stdout, stderr bytes.Buffer

	code := app.Run(context.Background(), []string{"labels"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "DIVE") {
		t.Errorf("expected DIVE label in output, got: %s", stdout.String())
	}
}

func TestApp_Run_CommandMissingArgs(t *testing.T) {
	app := NewDefaultApp()

	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "get without key",
			args:        []string{"get"},
			expectedErr: "사용법: jira get <KEY>",
		},
		{
			name:        "create without summary",
			args:        []string{"create"},
			expectedErr: "사용법: jira create <SUMMARY>",
		},
		{
			name:        "edit without key",
			args:        []string{"edit"},
			expectedErr: "사용법: jira edit <KEY>",
		},
		{
			name:        "move without args",
			args:        []string{"move"},
			expectedErr: "사용법: jira move <KEY> <STATUS>",
		},
		{
			name:        "comment without args",
			args:        []string{"comment"},
			expectedErr: "사용법: jira comment <KEY> <MESSAGE>",
		},
		{
			name:        "transitions without key",
			args:        []string{"transitions"},
			expectedErr: "사용법: jira transitions <KEY>",
		},
		{
			name:        "delete without key",
			args:        []string{"delete"},
			expectedErr: "사용법: jira delete <KEY>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := app.Run(context.Background(), tt.args, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("expected exit code 1 for %v, got %d", tt.args, code)
			}
			if !strings.Contains(stderr.String(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got: %s", tt.expectedErr, stderr.String())
			}
		})
	}
}

func TestApp_Run_Aliases(t *testing.T) {
	mockLoader := func() (*pkg.LabelCatalog, error) {
		return &pkg.LabelCatalog{
			Version: "1.0.0",
			Categories: map[string]pkg.LabelCategory{},
		}, nil
	}
	app := NewApp("1.5.0", nil, mockLoader)

	aliases := []struct {
		alias       string
		expectedErr string
	}{
		{"view", "사용법: jira get <KEY>"},
		{"show", "사용법: jira get <KEY>"},
		{"new", "사용법: jira create <SUMMARY>"},
		{"add", "사용법: jira create <SUMMARY>"},
		{"update", "사용법: jira edit <KEY>"},
		{"transition", "사용법: jira move <KEY> <STATUS>"},
		{"status", "사용법: jira move <KEY> <STATUS>"},
		{"rm", "사용법: jira delete <KEY>"},
		{"tags", ""},
	}

	for _, a := range aliases {
		t.Run(a.alias, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			app.Run(context.Background(), []string{a.alias}, &stdout, &stderr)
			if strings.Contains(stderr.String(), "알 수 없는 명령어") {
				t.Errorf("alias %q was not recognized as a command", a.alias)
			}
			if a.expectedErr != "" && !strings.Contains(stderr.String(), a.expectedErr) {
				t.Errorf("expected error %q, got %s", a.expectedErr, stderr.String())
			}
		})
	}
}

func TestApp_Run_E2E_CommandsWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql"):
			resp := pkg.SearchResponse{
				Issues: []pkg.Issue{
					{
						Key: "KAN-1",
						Fields: pkg.IssueFields{
							Summary:   "Sample task",
							Status:    pkg.Status{Name: "To Do"},
							IssueType: pkg.IssueType{Name: "작업"},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/KAN-1/transitions"):
			if r.Method == "GET" {
				resp := pkg.TransitionsResponse{
					Transitions: []pkg.Transition{
						{ID: "31", Name: "완료", To: pkg.Status{Name: "완료"}},
					},
				}
				json.NewEncoder(w).Encode(resp)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}

		case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/KAN-1/comment"):
			w.WriteHeader(http.StatusCreated)

		case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/KAN-1"):
			if r.Method == "GET" {
				issue := pkg.Issue{
					Key: "KAN-1",
					Fields: pkg.IssueFields{
						Summary:   "Sample task",
						Status:    pkg.Status{Name: "To Do"},
						IssueType: pkg.IssueType{Name: "작업"},
					},
				}
				json.NewEncoder(w).Encode(issue)
			} else if r.Method == "PUT" || r.Method == "DELETE" {
				w.WriteHeader(http.StatusNoContent)
			}

		case r.URL.Path == "/rest/api/3/issue" && r.Method == "POST":
			resp := pkg.CreateIssueResponse{
				Key: "KAN-2",
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	clientProvider := func() (*pkg.Client, error) {
		return pkg.NewClientWithConfig(pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "token",
			ProjectKey:  "KAN",
		}), nil
	}

	mockCatalogLoader := func() (*pkg.LabelCatalog, error) {
		return &pkg.LabelCatalog{
			Version: "1.0.0",
			Categories: map[string]pkg.LabelCategory{
				"project": {
					Name: "프로젝트",
					Labels: []pkg.LabelItem{
						{Name: "DIVE", Description: "DIVE Platform"},
					},
				},
			},
		}, nil
	}

	app := NewApp("1.5.0", clientProvider, mockCatalogLoader)

	testCases := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
	}{
		{
			name:       "list command",
			args:       []string{"list"},
			wantExit:   0,
			wantStdout: "KAN-1",
		},
		{
			name:       "get command",
			args:       []string{"get", "KAN-1"},
			wantExit:   0,
			wantStdout: "Sample task",
		},
		{
			name:       "create command with standard label",
			args:       []string{"create", "New Issue", "-l", "DIVE", "--due", "2026-09-10"},
			wantExit:   0,
			wantStdout: "KAN-2",
		},
		{
			name:       "edit command",
			args:       []string{"edit", "KAN-1", "-s", "Updated title"},
			wantExit:   0,
			wantStdout: "성공적으로 업데이트되었습니다",
		},
		{
			name:       "move command",
			args:       []string{"move", "KAN-1", "완료"},
			wantExit:   0,
			wantStdout: "성공적으로 변경되었습니다",
		},
		{
			name:       "comment command",
			args:       []string{"comment", "KAN-1", "코멘트 내용"},
			wantExit:   0,
			wantStdout: "코멘트가 등록되었습니다",
		},
		{
			name:       "transitions command",
			args:       []string{"transitions", "KAN-1"},
			wantExit:   0,
			wantStdout: "완료",
		},
		{
			name:       "delete command",
			args:       []string{"delete", "KAN-1"},
			wantExit:   0,
			wantStdout: "성공적으로 삭제되었습니다",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := app.Run(context.Background(), tc.args, &stdout, &stderr)
			if exitCode != tc.wantExit {
				t.Fatalf("expected exit %d, got %d. stderr: %s", tc.wantExit, exitCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Errorf("expected stdout to contain %q, got:\n%s", tc.wantStdout, stdout.String())
			}
		})
	}
}
