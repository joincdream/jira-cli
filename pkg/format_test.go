package pkg

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatStatusBadge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"해야 할 일", "[To Do]"},
		{"To Do", "[To Do]"},
		{"backlog", "[To Do]"},
		{"진행 중", "[In Progress *]"},
		{"In Progress", "[In Progress *]"},
		{"doing", "[In Progress *]"},
		{"완료", "[Done OK]"},
		{"done", "[Done OK]"},
		{"resolved", "[Done OK]"},
		{"검토 중", "[검토 중]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := formatStatusBadge(tt.input)
			if got != tt.want {
				t.Errorf("formatStatusBadge(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrintIssuesTable(t *testing.T) {
	t.Run("empty issues", func(t *testing.T) {
		var buf bytes.Buffer
		PrintIssuesTable(&buf, []Issue{})
		if !strings.Contains(buf.String(), "등록된 Jira 티켓이 없습니다.") {
			t.Errorf("expected empty message, got %q", buf.String())
		}
	})

	t.Run("valid issues table", func(t *testing.T) {
		var buf bytes.Buffer
		issues := []Issue{
			{
				Key: "KAN-1",
				Fields: IssueFields{
					Summary: "Test Summary 1",
					Status: Status{
						Name: "진행 중",
					},
					IssueType: IssueType{
						Name: "작업",
					},
					Labels:  []string{"AI", "Focus"},
					DueDate: "2026-09-01",
				},
			},
		}

		PrintIssuesTable(&buf, issues)
		out := buf.String()

		if !strings.Contains(out, "KAN-1") || !strings.Contains(out, "[In Progress *]") || !strings.Contains(out, "AI,Focus") {
			t.Errorf("table output missing expected values: %s", out)
		}
	})
}

func TestPrintIssuesMarkdown(t *testing.T) {
	var buf bytes.Buffer
	issues := []Issue{
		{
			Key: "KAN-1",
			Fields: IssueFields{
				Summary: "Test Summary with | pipe",
				Status: Status{
					Name: "진행 중",
				},
				IssueType: IssueType{
					Name: "작업",
				},
				Labels:  []string{"AI", "Focus"},
				DueDate: "2026-09-01",
			},
		},
	}

	PrintIssuesMarkdown(&buf, issues)
	out := buf.String()

	if !strings.Contains(out, "| Key | Type | Status | Labels | Assignee | Due Date | Summary |") {
		t.Errorf("expected markdown table header, got %s", out)
	}
	if !strings.Contains(out, "| KAN-1 | 작업 | 진행 중 | AI, Focus | 미지정 | 2026-09-01 | Test Summary with \\| pipe |") {
		t.Errorf("expected markdown row with escaped pipe, got %s", out)
	}
}

func TestPrintIssuesJSON(t *testing.T) {
	var buf bytes.Buffer
	issues := []Issue{
		{
			Key: "KAN-1",
			Fields: IssueFields{
				Summary: "JSON Test",
			},
		},
	}

	if err := PrintIssuesJSON(&buf, issues); err != nil {
		t.Fatalf("PrintIssuesJSON failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"key": "KAN-1"`) || !strings.Contains(out, `"summary": "JSON Test"`) {
		t.Errorf("json output missing expected fields: %s", out)
	}
}

func TestPrintIssueMarkdown(t *testing.T) {
	var buf bytes.Buffer
	issue := &Issue{
		Key: "KAN-1",
		Fields: IssueFields{
			Summary: "Issue Detail Markdown",
			Project: Project{Name: "Test Project", Key: "TEST"},
			Status:  Status{Name: "진행 중"},
			IssueType: IssueType{
				Name: "작업",
			},
			Labels:      []string{"AI", "Agent"},
			Description: "상세 설명 내용입니다.",
		},
	}

	PrintIssueMarkdown(&buf, issue)
	out := buf.String()

	if !strings.Contains(out, "# [KAN-1] Issue Detail Markdown") {
		t.Errorf("expected title, got: %s", out)
	}
	if !strings.Contains(out, "## 📄 상세 설명 (Description)") {
		t.Errorf("expected description heading, got: %s", out)
	}
	if !strings.Contains(out, "`AI`, `Agent`") {
		t.Errorf("expected wrapped labels, got: %s", out)
	}
}

func TestPrintIssueJSON(t *testing.T) {
	var buf bytes.Buffer
	issue := &Issue{
		Key: "KAN-10",
		Fields: IssueFields{
			Summary: "Single Issue JSON",
		},
	}

	if err := PrintIssueJSON(&buf, issue); err != nil {
		t.Fatalf("PrintIssueJSON failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"key": "KAN-10"`) || !strings.Contains(out, `"summary": "Single Issue JSON"`) {
		t.Errorf("json output missing expected fields: %s", out)
	}
}

func TestPrintIssuesMarkdownDetailed(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		var buf bytes.Buffer
		PrintIssuesMarkdownDetailed(&buf, []Issue{})
		if !strings.Contains(buf.String(), "등록된 Jira 티켓이 없습니다") {
			t.Errorf("expected empty message, got %q", buf.String())
		}
	})

	t.Run("multiple issues with details", func(t *testing.T) {
		var buf bytes.Buffer
		issues := []Issue{
			{
				Key: "KAN-1",
				Fields: IssueFields{
					Summary:     "First Task",
					Description: "First task description",
					Status:      Status{Name: "진행 중"},
					IssueType:   IssueType{Name: "작업"},
					Project:     Project{Key: "KAN", Name: "Kanban"},
				},
			},
			{
				Key: "KAN-2",
				Fields: IssueFields{
					Summary:     "Second Task",
					Description: "Second task description",
					Status:      Status{Name: "To Do"},
					IssueType:   IssueType{Name: "스토리"},
					Project:     Project{Key: "KAN", Name: "Kanban"},
				},
			},
		}

		PrintIssuesMarkdownDetailed(&buf, issues)
		out := buf.String()

		if !strings.Contains(out, "# [KAN-1] First Task") {
			t.Errorf("missing KAN-1 title in output: %s", out)
		}
		if !strings.Contains(out, "First task description") {
			t.Errorf("missing KAN-1 description in output: %s", out)
		}
		if !strings.Contains(out, "---") {
			t.Errorf("missing delimiter between issues in output: %s", out)
		}
		if !strings.Contains(out, "# [KAN-2] Second Task") {
			t.Errorf("missing KAN-2 title in output: %s", out)
		}
		if !strings.Contains(out, "Second task description") {
			t.Errorf("missing KAN-2 description in output: %s", out)
		}
	})
}

func TestPrintIssuesTableDetailed(t *testing.T) {
	var buf bytes.Buffer
	issues := []Issue{
		{
			Key: "KAN-1",
			Fields: IssueFields{
				Summary:     "First Task",
				Description: "First task description",
				Status:      Status{Name: "진행 중"},
				IssueType:   IssueType{Name: "작업"},
				Project:     Project{Key: "KAN", Name: "Kanban"},
			},
		},
	}
	PrintIssuesTableDetailed(&buf, issues)
	out := buf.String()
	if !strings.Contains(out, "🎫 [KAN-1] First Task") {
		t.Errorf("expected detailed issue header, got %s", out)
	}
	if !strings.Contains(out, "First task description") {
		t.Errorf("expected description, got %s", out)
	}
}

func TestPrintUsersFormatting(t *testing.T) {
	users := []User{
		{
			AccountID:    "acc-1",
			DisplayName:  "Alice Smith",
			EmailAddress: "alice@example.com",
			Active:       true,
		},
		{
			AccountID:    "acc-2",
			DisplayName:  "Bob Jones",
			EmailAddress: "bob@example.com",
			Active:       false,
		},
	}

	t.Run("table output", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUsersTable(&buf, users)
		out := buf.String()
		if !strings.Contains(out, "Alice Smith") || !strings.Contains(out, "alice@example.com") {
			t.Errorf("expected Alice in table output, got: %s", out)
		}
		if !strings.Contains(out, "Bob Jones") || !strings.Contains(out, "Inactive") {
			t.Errorf("expected Bob (Inactive) in table output, got: %s", out)
		}
	})

	t.Run("markdown output", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUsersMarkdown(&buf, users)
		out := buf.String()
		if !strings.Contains(out, "| Display Name | Email | Status | Account ID |") {
			t.Errorf("expected markdown header, got: %s", out)
		}
		if !strings.Contains(out, "| Alice Smith | alice@example.com | Active | `acc-1` |") {
			t.Errorf("expected Alice markdown row, got: %s", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		var buf bytes.Buffer
		err := PrintUsersJSON(&buf, users)
		if err != nil {
			t.Fatalf("PrintUsersJSON failed: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, `"accountId": "acc-1"`) {
			t.Errorf("expected acc-1 in JSON output, got: %s", out)
		}
	})

	t.Run("empty users", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUsersTable(&buf, []User{})
		if !strings.Contains(buf.String(), "조회된 Jira 사용자가 없습니다.") {
			t.Errorf("expected empty message, got %q", buf.String())
		}
	})
}

