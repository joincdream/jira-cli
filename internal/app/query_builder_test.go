package app

import (
	"strings"
	"testing"
)

func TestBuildFilterJQL(t *testing.T) {
	t.Run("default project with default status (open)", func(t *testing.T) {
		jql, err := BuildFilterJQL("JC", ListFilterOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPrefix := "project = JC AND statusCategory != Done ORDER BY"
		if !strings.HasPrefix(jql, expectedPrefix) {
			t.Errorf("expected JQL starting with %q, got %q", expectedPrefix, jql)
		}
	})

	t.Run("all projects scope (--all)", func(t *testing.T) {
		jql, err := BuildFilterJQL("JC", ListFilterOptions{All: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(jql, "project =") {
			t.Errorf("expected no project clause with --all, got %q", jql)
		}
		expectedPrefix := "statusCategory != Done ORDER BY"
		if !strings.HasPrefix(jql, expectedPrefix) {
			t.Errorf("expected JQL starting with %q, got %q", expectedPrefix, jql)
		}
	})

	t.Run("due filter relative keywords", func(t *testing.T) {
		tests := []struct {
			expr        string
			expectClause string
		}{
			{"today", "duedate = endOfDay()"},
			{"<=today", "duedate <= endOfDay()"},
			{">=today", "duedate >= startOfDay()"},
			{"tomorrow", "duedate = endOfDay('1d')"},
			{"<=tomorrow", "duedate <= endOfDay('1d')"},
			{"week", "duedate <= endOfWeek()"},
			{"<=week", "duedate <= endOfWeek()"},
			{"2026-10-15", "duedate = \"2026-10-15\""},
			{"<=2026-10-15", "duedate <= \"2026-10-15\""},
		}

		for _, tc := range tests {
			t.Run(tc.expr, func(t *testing.T) {
				jql, err := BuildFilterJQL("JC", ListFilterOptions{All: true, Due: tc.expr})
				if err != nil {
					t.Fatalf("unexpected error for %q: %v", tc.expr, err)
				}
				if !strings.Contains(jql, tc.expectClause) {
					t.Errorf("expected JQL to contain %q, got %q", tc.expectClause, jql)
				}
			})
		}
	})

	t.Run("invalid due filter format returns error", func(t *testing.T) {
		_, err := BuildFilterJQL("JC", ListFilterOptions{Due: "invalid-date"})
		if err == nil {
			t.Fatal("expected error for invalid due filter, got nil")
		}
		if !strings.Contains(err.Error(), "지원하지 않는 마감일(--due) 형식") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("status filter variants", func(t *testing.T) {
		// 1. done
		jqlDone, err := BuildFilterJQL("JC", ListFilterOptions{Status: "done"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(jqlDone, "statusCategory = Done") {
			t.Errorf("expected statusCategory = Done, got %q", jqlDone)
		}

		// 2. all (status skipped)
		jqlAllStatus, err := BuildFilterJQL("JC", ListFilterOptions{Status: "all"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(jqlAllStatus, "statusCategory") || strings.Contains(jqlAllStatus, "status =") {
			t.Errorf("expected no status filter with --status all, got %q", jqlAllStatus)
		}

		// 3. custom status name
		jqlCustom, err := BuildFilterJQL("JC", ListFilterOptions{Status: "검토 중"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(jqlCustom, `status = "검토 중"`) {
			t.Errorf("expected status = \"검토 중\", got %q", jqlCustom)
		}
	})

	t.Run("assignee filter variants", func(t *testing.T) {
		// 1. me
		jqlMe, err := BuildFilterJQL("JC", ListFilterOptions{Assignee: "me"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(jqlMe, "assignee = currentUser()") {
			t.Errorf("expected assignee = currentUser(), got %q", jqlMe)
		}

		// 2. none
		jqlNone, err := BuildFilterJQL("JC", ListFilterOptions{Assignee: "none"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(jqlNone, "assignee is EMPTY") {
			t.Errorf("expected assignee is EMPTY, got %q", jqlNone)
		}

		// 3. specific user
		jqlUser, err := BuildFilterJQL("JC", ListFilterOptions{Assignee: "test@example.com"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(jqlUser, `assignee = "test@example.com"`) {
			t.Errorf("expected assignee = \"test@example.com\", got %q", jqlUser)
		}
	})

	t.Run("combined complex query (all + <=today + me + open)", func(t *testing.T) {
		opts := ListFilterOptions{
			All:      true,
			Due:      "<=today",
			Status:   "open",
			Assignee: "me",
		}
		jql, err := BuildFilterJQL("JC", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedParts := []string{
			"statusCategory != Done",
			"duedate <= endOfDay()",
			"assignee = currentUser()",
			"ORDER BY duedate ASC, priority DESC, created DESC",
		}
		for _, part := range expectedParts {
			if !strings.Contains(jql, part) {
				t.Errorf("expected JQL to contain %q, got %q", part, jql)
			}
		}
		if strings.Contains(jql, "project =") {
			t.Errorf("expected no project clause with All=true, got %q", jql)
		}
	})
}
