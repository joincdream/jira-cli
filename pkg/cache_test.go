package pkg

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestOpenCacheDB_And_Schema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	// Verify Space Save & Get
	now := time.Now().Truncate(time.Second)
	if err := cache.SaveSpace("JC", "Jira Client", now); err != nil {
		t.Fatalf("failed to save space: %v", err)
	}

	space, err := cache.GetSpace("JC")
	if err != nil {
		t.Fatalf("failed to get space: %v", err)
	}
	if space == nil || space.Key != "JC" || space.Name != "Jira Client" {
		t.Fatalf("unexpected space: %+v", space)
	}
}

func TestCacheDB_UpsertAndGetIssues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_issues_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	mockIssues := []Issue{
		{
			Key: "JC-10",
			Fields: IssueFields{
				Summary: "Open Task 10",
				Status:  Status{Name: "진행 중"},
				Updated: "2026-10-17T10:00:00.000+0900",
			},
		},
		{
			Key: "JC-11",
			Fields: IssueFields{
				Summary: "Done Task 11",
				Status:  Status{Name: "완료"},
				Updated: "2026-10-17T11:00:00.000+0900",
			},
		},
		{
			Key: "JC-12",
			Fields: IssueFields{
				Summary: "To Do Task 12",
				Status:  Status{Name: "해야 할 일"},
				Updated: "2026-10-17T09:00:00.000+0900",
			},
		},
	}

	if err := cache.UpsertIssues("JC", mockIssues); err != nil {
		t.Fatalf("failed to upsert issues: %v", err)
	}

	// 1. Get all issues for JC
	all, err := cache.GetIssues("JC", "all")
	if err != nil {
		t.Fatalf("failed to get all issues: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(all))
	}
	// Verify order by updated_at DESC (JC-11, JC-10, JC-12)
	if all[0].Key != "JC-11" || all[1].Key != "JC-10" || all[2].Key != "JC-12" {
		t.Fatalf("unexpected order: %s, %s, %s", all[0].Key, all[1].Key, all[2].Key)
	}

	// 2. Filter open issues (should exclude '완료')
	open, err := cache.GetIssues("JC", "open")
	if err != nil {
		t.Fatalf("failed to get open issues: %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("expected 2 open issues, got %d", len(open))
	}

	// 3. Filter done issues
	done, err := cache.GetIssues("JC", "done")
	if err != nil {
		t.Fatalf("failed to get done issues: %v", err)
	}
	if len(done) != 1 || done[0].Key != "JC-11" {
		t.Fatalf("expected 1 done issue (JC-11), got: %+v", done)
	}

	// 4. Single Issue lookup
	issue, err := cache.GetIssue("jc-10")
	if err != nil {
		t.Fatalf("failed to get single issue: %v", err)
	}
	if issue == nil || issue.Key != "JC-10" || issue.Fields.Summary != "Open Task 10" {
		t.Fatalf("unexpected single issue: %+v", issue)
	}

	// Non-existent issue
	missing, err := cache.GetIssue("JC-999")
	if err != nil {
		t.Fatalf("unexpected error for missing issue: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing issue, got: %+v", missing)
	}
}

func TestCacheDB_ActivitiesAndRolling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_activity_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	// Log some activities
	now := time.Now()
	if err := cache.LogActivity("JC", "JC-14", "윤상배", "STATUS_CHANGE", "진행 중 ➔ 완료", now); err != nil {
		t.Fatalf("failed to log activity: %v", err)
	}
	if err := cache.LogActivity("JC", "JC-17", "AI Agent", "COMMENT_ADDED", "설계 계획 완료", now.Add(time.Second)); err != nil {
		t.Fatalf("failed to log activity: %v", err)
	}

	acts, err := cache.GetRecentActivities(10, "JC")
	if err != nil {
		t.Fatalf("failed to get recent activities: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(acts))
	}
	if acts[0].IssueKey != "JC-17" || acts[1].IssueKey != "JC-14" {
		t.Fatalf("unexpected activity order: %+v", acts)
	}
}

func TestCacheDB_ActiveIssueContext(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_context_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	// Initially empty
	active, err := cache.GetActiveIssue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active != "" {
		t.Fatalf("expected empty active issue, got %q", active)
	}

	// Set active issue
	if err := cache.SetActiveIssue("jc-17"); err != nil {
		t.Fatalf("failed to set active issue: %v", err)
	}

	active, err = cache.GetActiveIssue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active != "JC-17" {
		t.Fatalf("expected JC-17, got %q", active)
	}
}

func TestCacheDB_PruneDeletedIssues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_prune_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	mockIssues := []Issue{
		{Key: "JC-1", Fields: IssueFields{Summary: "1", Status: Status{Name: "To Do"}}},
		{Key: "JC-2", Fields: IssueFields{Summary: "2", Status: Status{Name: "To Do"}}},
		{Key: "JC-3", Fields: IssueFields{Summary: "3", Status: Status{Name: "To Do"}}},
	}
	_ = cache.UpsertIssues("JC", mockIssues)

	// Prune, keeping only JC-1 and JC-3 (JC-2 should be removed)
	if err := cache.PruneDeletedIssues("JC", []string{"JC-1", "JC-3"}); err != nil {
		t.Fatalf("failed to prune: %v", err)
	}

	remaining, err := cache.GetIssues("JC", "all")
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 issues remaining, got %d", len(remaining))
	}
	for _, r := range remaining {
		if r.Key == "JC-2" {
			t.Fatalf("JC-2 should have been pruned")
		}
	}
}

func TestCacheDB_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "jira_concurrency_test.db")

	cache, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cache.Close()

	var wg sync.WaitGroup
	// 5 concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = cache.LogActivity("JC", "JC-17", "agent", "UPDATE", "concurrency test", time.Now())
				_ = cache.SetActiveIssue("JC-17")
			}
		}(i)
	}

	// 5 concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = cache.GetRecentActivities(10, "JC")
				_, _ = cache.GetActiveIssue()
			}
		}(i)
	}

	wg.Wait()
}
