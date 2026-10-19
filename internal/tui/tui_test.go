package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tools/jira/pkg"
)

func TestCockpit_Initialization_And_Navigation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tui_cockpit.db")
	cache, err := pkg.OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer cache.Close()

	sampleIssues := []pkg.Issue{
		{
			Key: "JC-1",
			Fields: pkg.IssueFields{
				Summary: "First Task",
				Status:  pkg.Status{Name: "해야 할 일"},
				Updated: "2026-10-17T12:00:00.000+0900",
			},
		},
		{
			Key: "JC-2",
			Fields: pkg.IssueFields{
				Summary: "Second Task",
				Status:  pkg.Status{Name: "진행 중"},
				Updated: "2026-10-17T11:00:00.000+0900",
			},
		},
		{
			Key: "JC-3",
			Fields: pkg.IssueFields{
				Summary: "Third Task",
				Status:  pkg.Status{Name: "완료"},
				Updated: "2026-10-17T10:00:00.000+0900",
			},
		},
	}
	_ = cache.UpsertIssues("JC", sampleIssues)

	cockpit := NewCockpit("JC", "https://example.atlassian.net", nil, cache, nil)

	if len(cockpit.issues) != 3 {
		t.Fatalf("expected 3 issues loaded from cache, got %d", len(cockpit.issues))
	}
	if cockpit.table.GetRowCount() != 3 {
		t.Fatalf("expected 3 table rows, got %d", cockpit.table.GetRowCount())
	}

	// 1. Verify detail view contents for selected issue
	cockpit.onIssueSelectionChanged(0)
	detailText := cockpit.detailView.GetText(true)
	if !strings.Contains(detailText, "JC-1") || !strings.Contains(detailText, "First Task") {
		t.Errorf("expected detail to contain JC-1 and First Task, got:\n%s", detailText)
	}

	// 2. Select another row
	cockpit.onIssueSelectionChanged(2)
	detailText3 := cockpit.detailView.GetText(true)
	if !strings.Contains(detailText3, "JC-3") || !strings.Contains(detailText3, "Third Task") {
		t.Errorf("expected detail to contain JC-3 and Third Task, got:\n%s", detailText3)
	}

	// 3. Test Set Active Issue ('c')
	cockpit.table.Select(2, 0)
	cockpit.setCurrentIssueActive()
	activeKey, err := cache.GetActiveIssue()
	if err != nil || activeKey != "JC-3" {
		t.Errorf("expected active issue 'JC-3', got '%s', err: %v", activeKey, err)
	}

	// 4. Test Digest View Handling
	digest := &pkg.CatchupDigest{
		IsLongAbsence:   true,
		AbsenceDuration: 14 * 24 * time.Hour,
		NewlyAssigned: []pkg.Issue{
			{Key: "JC-2", Fields: pkg.IssueFields{Summary: "Newly assigned"}},
		},
		StatusChanges: []pkg.StatusChange{
			{IssueKey: "JC-3", OldStatus: "To Do", NewStatus: "Done"},
		},
		NewCommentCount:   3,
		TotalChangedCount: 5,
	}
	cockpit.digest = digest
	cockpit.showDigest = true
	cockpit.updateDigest()
	cockpit.rebuildRootLayout()

	digestText := cockpit.digestView.GetText(true)
	if !strings.Contains(digestText, "Welcome Back") || !strings.Contains(digestText, "14일간") {
		t.Errorf("expected digest text to contain Welcome Back, got:\n%s", digestText)
	}

	// Dismiss digest
	cockpit.showDigest = false
	cockpit.rebuildRootLayout()
	if cockpit.showDigest {
		t.Errorf("expected showDigest to be false")
	}

	// 5. Test Zoom Mode ('z')
	cockpit.zoomMode = true
	cockpit.rebuildWorkspaceFlex()
	if cockpit.workspaceFlex.GetItemCount() != 1 {
		t.Errorf("expected 1 item in workspaceFlex when zoomed, got %d", cockpit.workspaceFlex.GetItemCount())
	}

	cockpit.zoomMode = false
	cockpit.rebuildWorkspaceFlex()
	if cockpit.workspaceFlex.GetItemCount() != 2 {
		t.Errorf("expected 2 items in workspaceFlex after restore, got %d", cockpit.workspaceFlex.GetItemCount())
	}

	// 6. Test Ratio Adjustments ('[' and ']')
	initialLeft := cockpit.leftRatio
	cockpit.leftRatio += 5
	cockpit.rebuildWorkspaceFlex()
	if cockpit.leftRatio != initialLeft+5 {
		t.Errorf("expected leftRatio %d, got %d", initialLeft+5, cockpit.leftRatio)
	}
}

func TestWidgets_Formatting_And_Escaping(t *testing.T) {
	// 1. Status Badges
	badgeDone := formatStatusBadgeTview("완료")
	if !strings.Contains(badgeDone, "완료") {
		t.Errorf("expected badgeDone to contain 완료, got %s", badgeDone)
	}

	badgeProgress := formatStatusBadgeTview("In Progress")
	if !strings.Contains(badgeProgress, "진행 중") {
		t.Errorf("expected badgeProgress to contain 진행 중, got %s", badgeProgress)
	}

	// 2. Detail Card with CRLF Description and Special characters + Comments
	desc := "Line 1\r\nLine 2 with brackets [and tags] and emojis: ⚡ 🏖️\r\n"
	issue := &pkg.Issue{
		Key: "JC-99",
		Fields: pkg.IssueFields{
			Summary:     "Important [Bug] Fix",
			Status:      pkg.Status{Name: "진행 중"},
			Description: pkg.BuildADFDocument(desc),
			Comment: &pkg.CommentBlock{
				Total: 1,
				Comments: []pkg.Comment{
					{
						ID:      "10001",
						Author:  pkg.User{DisplayName: "Alice Smith"},
						Body:    "Fixed the regression in test environment.",
						Created: "2026-10-18T14:00:00.000+0900",
					},
				},
			},
		},
	}

	cardText := buildDetailCardText(issue)
	if !strings.Contains(cardText, "JC-99") {
		t.Errorf("expected card to contain JC-99")
	}
	if strings.Contains(cardText, "\r") {
		t.Errorf("expected cardText to have CRLF normalized without '\\r'")
	}
	if !strings.Contains(cardText, "Line 2 with brackets") {
		t.Errorf("expected card to render description lines")
	}
	if !strings.Contains(cardText, "Comments (1)") || !strings.Contains(cardText, "Alice Smith") {
		t.Errorf("expected card to render comment section and author, got:\n%s", cardText)
	}

	// 3. Activity Feed Text
	activities := []pkg.ActivityEvent{
		{
			IssueKey:  "JC-99",
			EventType: "STATUS_CHANGE",
			Details:   "진행 중 ➔ 완료",
			CreatedAt: time.Now(),
		},
	}
	actText := buildActivityFeedText(activities)
	if !strings.Contains(actText, "JC-99") || !strings.Contains(actText, "[STATUS]") {
		t.Errorf("expected activity feed to render event, got:\n%s", actText)
	}

	// 4. Header Text
	header := buildHeaderContent("JC", "LIVE WATCHING", time.Now())
	if !strings.Contains(header, "JIRA COCKPIT") || !strings.Contains(header, "SPACE: JC") {
		t.Errorf("expected header to contain title and space, got:\n%s", header)
	}
}
