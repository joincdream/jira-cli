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

func TestTUI_Filter_Utilities(t *testing.T) {
	// 1. Test IsUnresolvedStatus
	if !IsUnresolvedStatus("To Do") {
		t.Errorf("expected 'To Do' to be unresolved")
	}
	if !IsUnresolvedStatus("해야 할 일") {
		t.Errorf("expected '해야 할 일' to be unresolved")
	}
	if !IsUnresolvedStatus("진행 중") {
		t.Errorf("expected '진행 중' to be unresolved")
	}
	if !IsUnresolvedStatus("In Review") {
		t.Errorf("expected 'In Review' to be unresolved")
	}
	if IsUnresolvedStatus("완료") {
		t.Errorf("expected '완료' to be resolved")
	}
	if IsUnresolvedStatus("Done") {
		t.Errorf("expected 'Done' to be resolved")
	}
	if IsUnresolvedStatus("Closed") {
		t.Errorf("expected 'Closed' to be resolved")
	}

	// 2. Test GetUniqueStatuses
	issues := []pkg.Issue{
		{Key: "JC-1", Fields: pkg.IssueFields{Status: pkg.Status{Name: "To Do"}}},
		{Key: "JC-2", Fields: pkg.IssueFields{Status: pkg.Status{Name: "In Progress"}}},
		{Key: "JC-3", Fields: pkg.IssueFields{Status: pkg.Status{Name: "To Do"}}},
		{Key: "JC-4", Fields: pkg.IssueFields{Status: pkg.Status{Name: "Done"}}},
		{Key: "JC-5", Fields: pkg.IssueFields{Status: pkg.Status{Name: "In Progress"}}},
	}

	unique := GetUniqueStatuses(issues)
	expected := []string{"To Do", "In Progress", "Done"}
	if len(unique) != len(expected) {
		t.Fatalf("expected %d unique statuses, got %d", len(expected), len(unique))
	}
	for i, s := range expected {
		if unique[i] != s {
			t.Errorf("expected status %d to be '%s', got '%s'", i, s, unique[i])
		}
	}

	// 3. Test FilterIssuesByStatus
	filterSelection := map[string]bool{
		"To Do":       true,
		"In Progress": true,
	}
	filtered := FilterIssuesByStatus(issues, filterSelection)
	if len(filtered) != 4 {
		t.Fatalf("expected 4 issues after filter, got %d", len(filtered))
	}
	for _, iss := range filtered {
		if iss.Fields.Status.Name == "Done" {
			t.Errorf("unexpected issue with status 'Done' in filtered result: %s", iss.Key)
		}
	}

	// Empty filter returns all issues
	allFiltered := FilterIssuesByStatus(issues, nil)
	if len(allFiltered) != len(issues) {
		t.Errorf("expected nil filter to return all %d issues, got %d", len(issues), len(allFiltered))
	}
}

func TestCockpit_StatusFilter_Modal_And_Filtering(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tui_filter.db")
	cache, err := pkg.OpenCacheDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open cache: %v", err)
	}
	defer cache.Close()

	sampleIssues := []pkg.Issue{
		{Key: "JC-1", Fields: pkg.IssueFields{Summary: "Task 1", Status: pkg.Status{Name: "할 일"}}},
		{Key: "JC-2", Fields: pkg.IssueFields{Summary: "Task 2", Status: pkg.Status{Name: "진행 중"}}},
		{Key: "JC-3", Fields: pkg.IssueFields{Summary: "Task 3", Status: pkg.Status{Name: "완료"}}},
		{Key: "JC-4", Fields: pkg.IssueFields{Summary: "Task 4", Status: pkg.Status{Name: "검토 중"}}},
	}
	_ = cache.UpsertIssues("JC", sampleIssues)

	cockpit := NewCockpit("JC", "https://example.atlassian.net", nil, cache, nil)

	if len(cockpit.issues) != 4 {
		t.Fatalf("expected 4 issues, got %d", len(cockpit.issues))
	}
	if cockpit.table.GetRowCount() != 4 {
		t.Fatalf("expected 4 table rows, got %d", cockpit.table.GetRowCount())
	}

	// 1. Initial State: Filter is not active
	if cockpit.filterActive {
		t.Errorf("expected filterActive to be false initially")
	}

	// 2. Open Modal
	cockpit.showStatusFilterModal()
	if !cockpit.pages.HasPage("filterModal") {
		t.Fatalf("expected filterModal page to be active")
	}

	// 3. Test Apply Filter programmatically
	cockpit.filterActive = true
	cockpit.selectedStatuses = map[string]bool{
		"할 일":  true,
		"진행 중": true,
	}
	cockpit.applyFilter()
	cockpit.refreshIssuesTable()

	if len(cockpit.filteredIssues) != 2 {
		t.Fatalf("expected 2 filtered issues, got %d", len(cockpit.filteredIssues))
	}
	if cockpit.table.GetRowCount() != 2 {
		t.Fatalf("expected 2 table rows after filter, got %d", cockpit.table.GetRowCount())
	}

	// Check table title reflects filter
	title := cockpit.table.GetTitle()
	if !strings.Contains(title, "2/4") || !strings.Contains(title, "Filter") {
		t.Errorf("expected table title to show (2/4) and Filter, got: %s", title)
	}

	// 4. Test Navigation and detail view with filtered issues
	cockpit.table.Select(0, 0)
	cockpit.onIssueSelectionChanged(0)
	detailText := cockpit.detailView.GetText(true)
	expectedKey0 := cockpit.filteredIssues[0].Key
	if !strings.Contains(detailText, expectedKey0) {
		t.Errorf("expected row 0 to contain %s, got:\n%s", expectedKey0, detailText)
	}

	cockpit.table.Select(1, 0)
	cockpit.onIssueSelectionChanged(1)
	detailText2 := cockpit.detailView.GetText(true)
	expectedKey1 := cockpit.filteredIssues[1].Key
	if !strings.Contains(detailText2, expectedKey1) {
		t.Errorf("expected row 1 to contain %s, got:\n%s", expectedKey1, detailText2)
	}

	// 5. Test Clear Filter
	cockpit.filterActive = false
	cockpit.selectedStatuses = make(map[string]bool)
	cockpit.applyFilter()
	cockpit.refreshIssuesTable()

	if len(cockpit.getDisplayIssues()) != 4 {
		t.Fatalf("expected 4 display issues after clearing filter, got %d", len(cockpit.getDisplayIssues()))
	}
	if cockpit.table.GetRowCount() != 4 {
		t.Fatalf("expected 4 table rows after clearing filter, got %d", cockpit.table.GetRowCount())
	}
}
