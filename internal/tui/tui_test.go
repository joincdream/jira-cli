package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"tools/jira/pkg"
)

func TestTUI_Model_Navigation_And_View(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tui.db")
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

	model := NewModel("JC", "https://example.atlassian.net", nil, cache, nil)

	if len(model.issues) != 3 {
		t.Fatalf("expected 3 issues loaded from cache, got %d", len(model.issues))
	}
	if model.cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", model.cursor)
	}

	// Test Key Navigation
	// 1. Move Down ('j')
	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = m.(Model)
	if model.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", model.cursor)
	}

	// 2. Move Down ('j')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = m.(Model)
	if model.cursor != 2 {
		t.Errorf("expected cursor 2 after 'j', got %d", model.cursor)
	}

	// 3. Move Down again (clamped at end)
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = m.(Model)
	if model.cursor != 2 {
		t.Errorf("expected cursor clamped at 2, got %d", model.cursor)
	}

	// 4. Move Up ('k')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = m.(Model)
	if model.cursor != 1 {
		t.Errorf("expected cursor 1 after 'k', got %d", model.cursor)
	}

	// 5. Jump to Top ('g')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = m.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor 0 after 'g', got %d", model.cursor)
	}

	// 6. Jump to End ('G')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	model = m.(Model)
	if model.cursor != 2 {
		t.Errorf("expected cursor 2 after 'G', got %d", model.cursor)
	}

	// 7. Set Active ('c')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model = m.(Model)
	activeKey, err := cache.GetActiveIssue()
	if err != nil || activeKey != "JC-3" {
		t.Errorf("expected active issue 'JC-3', got '%s', err: %v", activeKey, err)
	}

	// 8. Window size update and rendering
	m, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = m.(Model)
	viewOutput := model.View()
	if !strings.Contains(viewOutput, "JIRA COCKPIT") {
		t.Errorf("expected view to contain 'JIRA COCKPIT', got:\n%s", viewOutput)
	}
	if !strings.Contains(viewOutput, "JC-1") || !strings.Contains(viewOutput, "JC-3") {
		t.Errorf("expected view to render issue keys, got:\n%s", viewOutput)
	}

	// 9. Digest Handling
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
	m, _ = model.Update(DigestMsg(digest))
	model = m.(Model)
	if !model.showDigest {
		t.Errorf("expected showDigest to be true")
	}

	digestView := model.View()
	if !strings.Contains(digestView, "Welcome Back") || !strings.Contains(digestView, "14일간") {
		t.Errorf("expected digest in view, got:\n%s", digestView)
	}

	// 10. Dismiss Digest ('esc')
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	model = m.(Model)
	if model.showDigest {
		t.Errorf("expected showDigest to be false after ESC")
	}

	// 11. Quit ('q')
	m, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model = m.(Model)
	if !model.quitting || cmd == nil {
		t.Errorf("expected quitting=true and non-nil cmd on 'q'")
	}
}
