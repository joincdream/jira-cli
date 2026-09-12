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
