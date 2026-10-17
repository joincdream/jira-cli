package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"tools/jira/pkg"
)

var (
	// Base Colors
	primaryColor    = lipgloss.Color("#2684FF") // Jira Blue
	accentColor     = lipgloss.Color("#00B8D9") // Cyan
	successColor    = lipgloss.Color("#36B37E") // Jira Green
	warningColor    = lipgloss.Color("#FFAB00") // Amber
	dangerColor     = lipgloss.Color("#FF5630") // Red
	mutedColor      = lipgloss.Color("#7A869A") // Gray
	darkBgColor     = lipgloss.Color("#172B4D") // Jira Dark Navy
	selectedBgColor = lipgloss.Color("#091E42") // Deep Navy

	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)

	// Header Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(darkBgColor).
			Padding(0, 1)

	statusLiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	statusSyncStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(warningColor)

	// Panel Styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	panelActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	panelHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#E6FCFF")).
				Background(darkBgColor).
				Padding(0, 1)

	// Selected Item Style
	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(selectedBgColor)

	// Status Badges
	badgeDoneStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	badgeInProgressStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)

	badgeTodoStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	badgeReviewStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor)

	// Footer Styles
	footerKeyStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	footerNoticeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#36B37E"))

	// Digest Box Style
	digestBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(warningColor).
			Background(lipgloss.Color("#201B15")).
			Padding(0, 1)
)

// View renders the TUI layout.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing Jira Cockpit..."
	}

	totalWidth := m.width
	totalHeight := m.height

	var sections []string

	// 1. Header (Height: ~1-2)
	header := m.renderHeader(totalWidth)
	sections = append(sections, header)

	// 2. Digest Banner (if vacation/long gap detected and not dismissed)
	digestHeight := 0
	if m.showDigest && m.digest != nil {
		digestView := m.renderDigest(totalWidth)
		sections = append(sections, digestView)
		digestHeight = lipgloss.Height(digestView)
	}

	// 3. Main Workspace: Split Left (List) & Right (Detail + Activity)
	footerHeight := 1
	headerHeight := lipgloss.Height(header)
	contentHeight := totalHeight - headerHeight - digestHeight - footerHeight
	if contentHeight < 10 {
		contentHeight = 10
	}

	mainWorkspace := m.renderMainWorkspace(totalWidth, contentHeight)
	sections = append(sections, mainWorkspace)

	// 4. Footer (Height: 1)
	footer := m.renderFooter(totalWidth)
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderHeader(width int) string {
	title := titleStyle.Render("⚡ JIRA COCKPIT")
	spaceBadge := badgeStyle.Render(fmt.Sprintf("SPACE: %s", m.space))

	syncIndicator := ""
	if strings.Contains(m.syncState, "SYNC") {
		syncIndicator = statusSyncStyle.Render("🔄 " + m.syncState)
	} else {
		syncIndicator = statusLiveStyle.Render("🟢 " + m.syncState)
	}

	syncTime := fmt.Sprintf("Last sync: %s", m.lastSyncTime.Format("15:04:05"))
	rightInfo := lipgloss.JoinHorizontal(lipgloss.Center, syncIndicator, "  ", mutedStyle.Render(syncTime))

	leftGroup := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", spaceBadge)
	gapWidth := width - runewidth.StringWidth(leftGroup) - runewidth.StringWidth(rightInfo)
	if gapWidth < 1 {
		gapWidth = 1
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftGroup, strings.Repeat(" ", gapWidth), rightInfo)
}

func (m Model) renderDigest(width int) string {
	if m.digest == nil {
		return ""
	}

	var lines []string
	days := int(m.digest.AbsenceDuration.Hours() / 24)
	if days < 1 {
		days = 1
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(warningColor).Render(
		fmt.Sprintf("🏖️ Welcome Back! 부재 중 업무 변경 브리핑 (최근 %d일간 %d건 변경 감지)", days, m.digest.TotalChangedCount),
	)
	lines = append(lines, title)

	if len(m.digest.NewlyAssigned) > 0 {
		var keys []string
		for _, iss := range m.digest.NewlyAssigned {
			keys = append(keys, iss.Key)
		}
		lines = append(lines, fmt.Sprintf("  ✨ 내게 새로 할당된 티켓: %s", strings.Join(keys, ", ")))
	}
	if len(m.digest.StatusChanges) > 0 {
		var changes []string
		for _, sc := range m.digest.StatusChanges {
			changes = append(changes, fmt.Sprintf("[%s] %s ➔ %s", sc.IssueKey, sc.OldStatus, sc.NewStatus))
		}
		lines = append(lines, fmt.Sprintf("  🔄 상태 변경된 내 티켓: %s", strings.Join(changes, ", ")))
	}
	if m.digest.NewCommentCount > 0 {
		lines = append(lines, fmt.Sprintf("  💬 확인해야 할 신규 코멘트: %d건", m.digest.NewCommentCount))
	}
	if len(m.digest.NewlyAssigned) == 0 && len(m.digest.StatusChanges) == 0 && m.digest.NewCommentCount == 0 {
		lines = append(lines, "  ✓ 부재 중 담당 티켓의 중대한 변경 사항이 없습니다.")
	}
	lines = append(lines, lipgloss.NewStyle().Faint(true).Render("  [ESC 키를 눌러 브리핑 창을 닫습니다]"))

	content := strings.Join(lines, "\n")
	boxWidth := width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	return digestBoxStyle.Width(boxWidth).Render(content)
}

func (m Model) renderMainWorkspace(width, height int) string {
	leftWidth := width * 42 / 100
	if leftWidth < 36 {
		leftWidth = 36
	}
	rightWidth := width - leftWidth - 2
	if rightWidth < 30 {
		rightWidth = 30
	}

	leftPanel := m.renderIssuesList(leftWidth, height)
	rightPanel := m.renderRightPane(rightWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderIssuesList(width, height int) string {
	boxWidth := width - 2
	boxHeight := height - 2

	headerText := fmt.Sprintf("📋 Active Issues (%d)", len(m.issues))
	header := panelHeaderStyle.Render(headerText)

	visibleLines := boxHeight - 2
	if visibleLines < 3 {
		visibleLines = 3
	}

	var rows []string
	if len(m.issues) == 0 {
		rows = append(rows, mutedStyle.Render("  No cached issues found."))
		rows = append(rows, mutedStyle.Render("  Waiting for background sync..."))
	} else {
		// Calculate scroll window
		startIndex := 0
		if m.cursor >= visibleLines {
			startIndex = m.cursor - visibleLines + 1
		}
		endIndex := startIndex + visibleLines
		if endIndex > len(m.issues) {
			endIndex = len(m.issues)
		}

		for i := startIndex; i < endIndex; i++ {
			issue := m.issues[i]
			isSelected := (i == m.cursor)

			prefix := "  "
			if isSelected {
				prefix = "> "
			}

			keyStr := fmt.Sprintf("%-7s", issue.Key)
			statusBadge := formatStatusBadgeTUI(issue.Fields.Status.Name)

			// Truncate summary to fit line
			prefixWidth := runewidth.StringWidth(prefix) + runewidth.StringWidth(keyStr) + runewidth.StringWidth(statusBadge) + 3
			maxSummaryLen := boxWidth - prefixWidth
			summary := issue.Fields.Summary
			if maxSummaryLen > 4 && runewidth.StringWidth(summary) > maxSummaryLen {
				summary = runewidth.Truncate(summary, maxSummaryLen-2, "…")
			}

			line := fmt.Sprintf("%s%s %s %s", prefix, keyStr, statusBadge, summary)
			if isSelected {
				pad := boxWidth - runewidth.StringWidth(line)
				if pad > 0 {
					line += strings.Repeat(" ", pad)
				}
				rows = append(rows, selectedRowStyle.Render(line))
			} else {
				rows = append(rows, line)
			}
		}
	}

	// Pad remaining vertical space
	for len(rows) < visibleLines {
		rows = append(rows, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(rows, "\n"))
	return panelActiveStyle.Width(boxWidth).Height(boxHeight).Render(content)
}

func (m Model) renderRightPane(width, height int) string {
	boxWidth := width - 2
	boxHeight := height - 2

	// Split height: 55% detail, 45% activities
	detailHeight := boxHeight * 55 / 100
	activityHeight := boxHeight - detailHeight - 1

	detailView := m.renderIssueDetail(boxWidth, detailHeight)
	activityView := m.renderActivityFeed(boxWidth, activityHeight)

	return lipgloss.JoinVertical(lipgloss.Left, detailView, activityView)
}

func (m Model) renderIssueDetail(width, height int) string {
	header := panelHeaderStyle.Render("📄 Issue Details")
	usableHeight := height - 2
	if usableHeight < 4 {
		usableHeight = 4
	}

	var lines []string

	if len(m.issues) == 0 || m.cursor < 0 || m.cursor >= len(m.issues) {
		lines = append(lines, mutedStyle.Render("No issue selected"))
	} else {
		cur := m.issues[m.cursor]
		keyBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(primaryColor).Render(" " + cur.Key + " ")
		statusBadge := formatStatusBadgeTUI(cur.Fields.Status.Name)
		typeStr := cur.Fields.IssueType.Name
		if typeStr == "" {
			typeStr = "Task"
		}

		lines = append(lines, fmt.Sprintf("%s  %s  [%s]", keyBadge, statusBadge, typeStr))
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render(cur.Fields.Summary))

		assignee := "Unassigned"
		if cur.Fields.Assignee != nil && cur.Fields.Assignee.DisplayName != "" {
			assignee = cur.Fields.Assignee.DisplayName
		}
		reporter := "Unknown"
		if cur.Fields.Reporter != nil && cur.Fields.Reporter.DisplayName != "" {
			reporter = cur.Fields.Reporter.DisplayName
		}

		meta := fmt.Sprintf("👤 Assignee: %s | ✍️ Reporter: %s", assignee, reporter)
		lines = append(lines, mutedStyle.Render(meta))

		if cur.Fields.Description != nil {
			descText := pkg.ExtractTextFromADF(cur.Fields.Description)
			if strings.TrimSpace(descText) != "" {
				descLines := strings.Split(descText, "\n")
				lines = append(lines, "")
				lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Description:"))
				for _, dl := range descLines {
					if len(lines) >= usableHeight-2 {
						lines = append(lines, mutedStyle.Render("... (more in browser)"))
						break
					}
					if strings.TrimSpace(dl) != "" {
						lines = append(lines, "  "+runewidth.Truncate(dl, width-6, "…"))
					}
				}
			}
		}
	}

	for len(lines) < usableHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(lines, "\n"))
	return panelStyle.Width(width).Height(height).Render(content)
}

func (m Model) renderActivityFeed(width, height int) string {
	header := panelHeaderStyle.Render("⚡ Live Activity Stream")
	usableHeight := height - 2
	if usableHeight < 3 {
		usableHeight = 3
	}

	var lines []string
	if len(m.activities) == 0 {
		lines = append(lines, mutedStyle.Render("  No live events yet. Watching for changes..."))
	} else {
		for i, act := range m.activities {
			if i >= usableHeight {
				break
			}
			timeStr := act.CreatedAt.Format("15:04:05")
			badge := formatActivityBadge(act.EventType)
			detail := act.Details

			maxLen := width - runewidth.StringWidth(timeStr) - runewidth.StringWidth(badge) - runewidth.StringWidth(act.IssueKey) - 8
			if maxLen > 4 && runewidth.StringWidth(detail) > maxLen {
				detail = runewidth.Truncate(detail, maxLen-2, "…")
			}

			line := fmt.Sprintf("  %s %s %s: %s", mutedStyle.Render(timeStr), badge, lipgloss.NewStyle().Bold(true).Render(act.IssueKey), detail)
			lines = append(lines, line)
		}
	}

	for len(lines) < usableHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(lines, "\n"))
	return panelStyle.Width(width).Height(height).Render(content)
}

func (m Model) renderFooter(width int) string {
	hints := footerKeyStyle.Render("[↑/↓, j/k] Navigate  [r] Sync Now  [o] Open Browser  [c] Set Active  [esc] Dismiss  [q] Quit")
	if m.statusMessage != "" {
		notice := footerNoticeStyle.Render(" " + m.statusMessage + " ")
		gap := width - runewidth.StringWidth(hints) - runewidth.StringWidth(notice)
		if gap < 1 {
			gap = 1
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, hints, strings.Repeat(" ", gap), notice)
	}
	return hints
}

func formatStatusBadgeTUI(status string) string {
	norm := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(norm, "완료") || strings.Contains(norm, "done") || strings.Contains(norm, "resolved"):
		return badgeDoneStyle.Render("[완료]")
	case strings.Contains(norm, "진행") || strings.Contains(norm, "progress") || strings.Contains(norm, "doing"):
		return badgeInProgressStyle.Render("[진행 중]")
	case strings.Contains(norm, "검토") || strings.Contains(norm, "review"):
		return badgeReviewStyle.Render("[검토 중]")
	default:
		return badgeTodoStyle.Render(fmt.Sprintf("[%s]", status))
	}
}

func formatActivityBadge(eventType string) string {
	switch eventType {
	case "STATUS_CHANGE":
		return badgeInProgressStyle.Render("[STATUS]")
	case "COMMENT_ADDED":
		return badgeReviewStyle.Render("[COMMENT]")
	case "CREATED":
		return badgeDoneStyle.Render("[CREATED]")
	default:
		return badgeTodoStyle.Render("[UPDATE]")
	}
}
