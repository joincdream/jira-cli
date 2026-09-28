package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tools/jira/pkg"
)

func TestBuildReportJQL(t *testing.T) {
	tests := []struct {
		name           string
		all            bool
		project        string
		defaultProject string
		assignee       string
		period         string
		expectedJQL    string
		expectedScope  string
	}{
		{
			name:           "default project and default week period",
			all:            false,
			defaultProject: "JC",
			period:         "week",
			expectedJQL:    "project = JC AND ((statusCategory != Done) OR (resolved >= startOfWeek() OR (updated >= startOfWeek() AND statusCategory = Done))) ORDER BY duedate ASC, updated DESC",
			expectedScope:  "Project JC",
		},
		{
			name:          "all projects and month period",
			all:           true,
			period:        "month",
			expectedJQL:   "((statusCategory != Done) OR (resolved >= startOfMonth() OR (updated >= startOfMonth() AND statusCategory = Done))) ORDER BY duedate ASC, updated DESC",
			expectedScope: "All Projects",
		},
		{
			name:           "specific project and period all with assignee me",
			all:            false,
			project:        "KAN",
			assignee:       "me",
			period:         "all",
			expectedJQL:    "project = KAN AND assignee = currentUser() ORDER BY duedate ASC, updated DESC",
			expectedScope:  "Project KAN",
		},
		{
			name:           "assignee none",
			all:            false,
			defaultProject: "JC",
			assignee:       "none",
			period:         "week",
			expectedJQL:    "project = JC AND ((statusCategory != Done) OR (resolved >= startOfWeek() OR (updated >= startOfWeek() AND statusCategory = Done))) AND assignee is EMPTY ORDER BY duedate ASC, updated DESC",
			expectedScope:  "Project JC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jql, scope := buildReportJQL(tt.all, tt.project, tt.defaultProject, tt.assignee, tt.period)
			if jql != tt.expectedJQL {
				t.Errorf("expected JQL %q, got %q", tt.expectedJQL, jql)
			}
			if scope != tt.expectedScope {
				t.Errorf("expected scope %q, got %q", tt.expectedScope, scope)
			}
		})
	}
}

func TestAggregateReport(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	issues := []pkg.Issue{
		{
			Key: "JC-1",
			Fields: pkg.IssueFields{
				Summary: "완료된 작업 1",
				Status: pkg.Status{
					Name: "완료",
					StatusCategory: &pkg.StatusCategory{
						Key: "done",
					},
				},
				Assignee: &pkg.User{DisplayName: "윤상배"},
				Project:  pkg.Project{Key: "JC"},
			},
		},
		{
			Key: "JC-2",
			Fields: pkg.IssueFields{
				Summary: "지연된 작업 (2일 전 마감)",
				Status: pkg.Status{
					Name: "진행 중",
					StatusCategory: &pkg.StatusCategory{
						Key: "indeterminate",
					},
				},
				Assignee: &pkg.User{DisplayName: "윤상배"},
				DueDate:  "2026-09-26",
				Project:  pkg.Project{Key: "JC"},
			},
		},
		{
			Key: "JC-3",
			Fields: pkg.IssueFields{
				Summary: "마감 임박 작업 (내일 마감)",
				Status: pkg.Status{
					Name: "해야 할 일",
					StatusCategory: &pkg.StatusCategory{
						Key: "new",
					},
				},
				DueDate: "2026-09-29",
				Project: pkg.Project{Key: "JC"},
			},
		},
		{
			Key: "KAN-1",
			Fields: pkg.IssueFields{
				Summary: "다른 프로젝트 작업",
				Status: pkg.Status{
					Name: "진행 중",
					StatusCategory: &pkg.StatusCategory{
						Key: "indeterminate",
					},
				},
				Assignee: &pkg.User{DisplayName: "홍길동"},
				Project:  pkg.Project{Key: "KAN"},
			},
		},
	}

	data := aggregateReport(issues, "All Projects", "week", true, now)

	// Summary checks
	if data.Summary.Total != 4 {
		t.Errorf("expected total 4, got %d", data.Summary.Total)
	}
	if data.Summary.Done != 1 {
		t.Errorf("expected done 1, got %d", data.Summary.Done)
	}
	if data.Summary.InProgress != 2 {
		t.Errorf("expected in_progress 2, got %d", data.Summary.InProgress)
	}
	if data.Summary.ToDo != 1 {
		t.Errorf("expected to_do 1, got %d", data.Summary.ToDo)
	}
	if data.Summary.Overdue != 1 {
		t.Errorf("expected overdue 1, got %d", data.Summary.Overdue)
	}
	if data.Summary.DueSoon != 1 {
		t.Errorf("expected due_soon 1, got %d", data.Summary.DueSoon)
	}
	if data.Summary.DoneRate != 25.0 {
		t.Errorf("expected done_rate 25.0, got %.1f", data.Summary.DoneRate)
	}

	// Health check (1 overdue -> WARNING)
	if data.Health != "WARNING" {
		t.Errorf("expected Health 'WARNING', got %s", data.Health)
	}

	// Overdue issue check
	if len(data.Overdue) != 1 || data.Overdue[0].Key != "JC-2" {
		t.Errorf("expected Overdue issue JC-2, got %+v", data.Overdue)
	}
	if data.Overdue[0].DiffLabel != "+2일 지연" {
		t.Errorf("expected diff label '+2일 지연', got %s", data.Overdue[0].DiffLabel)
	}

	// DueSoon issue check
	if len(data.DueSoon) != 1 || data.DueSoon[0].Key != "JC-3" {
		t.Errorf("expected DueSoon issue JC-3, got %+v", data.DueSoon)
	}

	// Assignee workload check
	if len(data.Assignees) != 3 {
		t.Errorf("expected 3 assignees, got %d", len(data.Assignees))
	}

	// Project breakdown check
	if len(data.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(data.Projects))
	}
}

func TestReportCommand_Execute(t *testing.T) {
	mockIssues := []pkg.Issue{
		{
			Key: "JC-10",
			Fields: pkg.IssueFields{
				Summary: "완료된 티켓",
				Status: pkg.Status{
					Name: "Done",
					StatusCategory: &pkg.StatusCategory{
						Key: "done",
					},
				},
				Assignee: &pkg.User{DisplayName: "윤상배"},
				Project:  pkg.Project{Key: "JC"},
			},
		},
		{
			Key: "JC-11",
			Fields: pkg.IssueFields{
				Summary: "진행 중 티켓",
				Status: pkg.Status{
					Name: "In Progress",
					StatusCategory: &pkg.StatusCategory{
						Key: "indeterminate",
					},
				},
				Assignee: &pkg.User{DisplayName: "윤상배"},
				DueDate:  "2026-10-01",
				Project:  pkg.Project{Key: "JC"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(pkg.SearchResponse{
				Issues: mockIssues,
				IsLast: true,
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
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

	cmd := NewReportCommand(clientProvider)

	t.Run("execute terminal format", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := stdout.String()
		if !strings.Contains(out, "주간 업무 브리핑") {
			t.Errorf("expected terminal output to contain title, got: %s", out)
		}
		if !strings.Contains(out, "JC-10") || !strings.Contains(out, "JC-11") {
			t.Errorf("expected issues in terminal output, got: %s", out)
		}
	})

	t.Run("execute markdown format", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--md"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := stdout.String()
		if !strings.Contains(out, "# 주간 업무 브리핑 리포트") {
			t.Errorf("expected markdown title, got: %s", out)
		}
		if !strings.Contains(out, "Executive Summary") {
			t.Errorf("expected Executive Summary, got: %s", out)
		}
		if !strings.Contains(out, "| 전체 이슈 |") {
			t.Errorf("expected Executive Summary Table header, got: %s", out)
		}
		if !strings.Contains(out, "JC-10") {
			t.Errorf("expected JC-10 in completed, got: %s", out)
		}
	})

	t.Run("execute json format", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--json"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed ReportData
		if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
			t.Fatalf("failed to parse json output: %v", err)
		}

		if parsed.Summary.Total != 2 {
			t.Errorf("expected total 2 in json, got %d", parsed.Summary.Total)
		}
	})

	t.Run("execute help format", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := cmd.Execute(context.Background(), []string{"--help"}, &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "jira report [options]") {
			t.Errorf("expected help output, got: %s", stdout.String())
		}
	})
}
