package pkg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiffIssues(t *testing.T) {
	oldMap := map[string]Issue{
		"JC-1": {
			Key: "JC-1",
			Fields: IssueFields{
				Summary: "Old Task 1",
				Status:  Status{Name: "To Do"},
				Comment: &CommentBlock{Comments: []Comment{{Body: "c1"}}},
			},
		},
		"JC-2": {
			Key: "JC-2",
			Fields: IssueFields{
				Summary:  "Old Task 2",
				Status:   Status{Name: "진행 중"},
				Assignee: &User{DisplayName: "Bob", EmailAddress: "bob@example.com"},
			},
		},
	}

	newIssues := []Issue{
		{
			// JC-1: status changed, 1 new comment
			Key: "JC-1",
			Fields: IssueFields{
				Summary: "Old Task 1",
				Status:  Status{Name: "진행 중"},
				Comment: &CommentBlock{
					Comments: []Comment{
						{Body: "c1", Author: User{DisplayName: "Alice"}},
						{Body: "c2", Author: User{DisplayName: "Bob"}},
					},
				},
			},
		},
		{
			// JC-2: newly assigned to me
			Key: "JC-2",
			Fields: IssueFields{
				Summary:  "Old Task 2",
				Status:   Status{Name: "진행 중"},
				Assignee: &User{DisplayName: "Me", EmailAddress: "me@example.com"},
			},
		},
		{
			// JC-3: newly created
			Key: "JC-3",
			Fields: IssueFields{
				Summary:   "Brand New Task 3",
				Status:    Status{Name: "To Do"},
				IssueType: IssueType{Name: "Task"},
				Assignee:  &User{DisplayName: "Me", EmailAddress: "me@example.com"},
			},
		},
	}

	events, statusChanges, newlyAssigned, newComments := DiffIssues("JC", oldMap, newIssues, "me@example.com")

	// 1. Status changes
	if len(statusChanges) != 1 || statusChanges[0].IssueKey != "JC-1" || statusChanges[0].NewStatus != "진행 중" {
		t.Fatalf("expected 1 status change for JC-1, got: %+v", statusChanges)
	}

	// 2. Newly assigned to me (JC-2 and JC-3)
	if len(newlyAssigned) != 2 {
		t.Fatalf("expected 2 newly assigned issues, got %d", len(newlyAssigned))
	}

	// 3. New comments
	if newComments != 1 {
		t.Fatalf("expected 1 new comment, got %d", newComments)
	}

	// 4. Activity events (JC-1 status, JC-1 comment, JC-3 created = 3 events)
	if len(events) != 3 {
		t.Fatalf("expected 3 activity events, got %d: %+v", len(events), events)
	}
}

func TestCatchupDigest_Format(t *testing.T) {
	digest := &CatchupDigest{
		IsLongAbsence:   true,
		AbsenceDuration: 14 * 24 * time.Hour,
		NewlyAssigned: []Issue{
			{Key: "JC-20", Fields: IssueFields{Summary: "결제 모듈 긴급 패치"}},
		},
		StatusChanges: []StatusChange{
			{IssueKey: "JC-14", OldStatus: "진행 중", NewStatus: "완료"},
		},
		NewCommentCount:   5,
		TotalChangedCount: 8,
	}

	report := digest.FormatSummary()
	if !strings.Contains(report, "Welcome Back!") {
		t.Fatalf("expected Welcome Back in report, got: %s", report)
	}
	if !strings.Contains(report, "14일 전") {
		t.Fatalf("expected 14일 전 in report, got: %s", report)
	}
	if !strings.Contains(report, "[JC-20]") || !strings.Contains(report, "[JC-14]") {
		t.Fatalf("expected JC-20 and JC-14 in report, got: %s", report)
	}
}

func TestWatchEngine_InitState_And_PollOnce(t *testing.T) {
	var searchCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			searchCalls++
			w.WriteHeader(http.StatusOK)
			mockIssues := []Issue{
				{
					Key: "JC-17",
					Fields: IssueFields{
						Summary: "Watch Engine Test",
						Status:  Status{Name: "진행 중"},
						Updated: "2026-10-17T12:00:00.000+0900",
					},
				},
			}
			json.NewEncoder(w).Encode(SearchResponse{Issues: mockIssues})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithConfig(Config{
		InstanceURL: server.URL,
		Email:       "me@example.com",
		APIToken:    "dummy",
		ProjectKey:  "JC",
	})

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "watch_test.db")
	cacheDB, err := OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache db: %v", err)
	}
	defer cacheDB.Close()

	engine := NewWatchEngine(client, cacheDB, "JC", 15*time.Second)

	// 1. Initial State Sync
	digest, err := engine.InitState(context.Background())
	if err != nil {
		t.Fatalf("InitState error: %v", err)
	}
	// Initial sync is not long absence
	if digest != nil && digest.IsLongAbsence {
		t.Fatalf("expected nil or non-long-absence digest on fresh db")
	}

	// Verify in-memory issues loaded
	memIssues := engine.GetMemoryIssues()
	if len(memIssues) != 1 || memIssues[0].Key != "JC-17" {
		t.Fatalf("expected JC-17 in memory, got: %+v", memIssues)
	}

	// Verify event published
	select {
	case evt := <-engine.EventChannel():
		if evt.IssueKey != "JC-17" || evt.EventType != "CREATED" {
			t.Fatalf("unexpected event: %+v", evt)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected event in channel, got timeout")
	}

	// Verify saved to SQLite
	cached, err := cacheDB.GetIssues("JC", "all")
	if err != nil || len(cached) != 1 || cached[0].Key != "JC-17" {
		t.Fatalf("expected JC-17 in SQLite cache, got: %+v", cached)
	}

	// 2. Second poll: JC-17 transitions to "완료" (Done)
	// Even though it is now Done, watch engine should detect the STATUS_CHANGE event,
	// publish it to event channel, and prune it from active memory & cache.
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/search/jql") {
			w.WriteHeader(http.StatusOK)
			doneIssues := []Issue{
				{
					Key: "JC-17",
					Fields: IssueFields{
						Summary: "Watch Engine Test",
						Status: Status{
							Name:           "완료",
							StatusCategory: &StatusCategory{Key: "done", Name: "Done"},
						},
						Updated: "2026-10-18T15:15:00.000+0900",
					},
				},
			}
			json.NewEncoder(w).Encode(SearchResponse{Issues: doneIssues})
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	events, err := engine.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != "STATUS_CHANGE" {
		t.Fatalf("expected 1 STATUS_CHANGE event, got: %+v", events)
	}
	if !strings.Contains(events[0].Details, "진행 중 ➔ 완료") {
		t.Errorf("expected transition details to contain '진행 중 ➔ 완료', got: %s", events[0].Details)
	}

	// Active issues in memory should now be empty (pruned because it's Done)
	if len(engine.GetMemoryIssues()) != 0 {
		t.Errorf("expected 0 active issues in memory after completion, got %d", len(engine.GetMemoryIssues()))
	}
	// Active issues in SQLite should also be 0
	cachedAfterDone, _ := cacheDB.GetIssues("JC", "all")
	if len(cachedAfterDone) != 0 {
		t.Errorf("expected 0 active issues in SQLite cache after completion, got %d", len(cachedAfterDone))
	}
}
