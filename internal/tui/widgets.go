package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"tools/jira/pkg"
)

var (
	colorJiraBlue = tcell.NewHexColor(0x2684FF)
	colorGreen    = tcell.NewHexColor(0x36B37E)
	colorAmber    = tcell.NewHexColor(0xFFAB00)
	colorRed      = tcell.NewHexColor(0xFF5630)
	colorDarkNavy = tcell.NewHexColor(0x172B4D)
	colorMuted    = tcell.NewHexColor(0x7A869A)
)

func formatStatusBadgeTview(status string) string {
	norm := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(norm, "완료") || strings.Contains(norm, "done") || strings.Contains(norm, "resolved"):
		return "[#36B37E:black:b][완료][-:-:-]"
	case strings.Contains(norm, "진행") || strings.Contains(norm, "progress") || strings.Contains(norm, "doing"):
		return "[#00B8D9:black:b][진행 중][-:-:-]"
	case strings.Contains(norm, "검토") || strings.Contains(norm, "review"):
		return "[#FFAB00:black:b][검토 중][-:-:-]"
	default:
		return fmt.Sprintf("[#7A869A:black][%s][-:-:-]", status)
	}
}

func formatActivityBadgeTview(eventType string) string {
	switch eventType {
	case "STATUS_CHANGE":
		return "[#00B8D9:black:b][STATUS][-:-:-]"
	case "COMMENT_ADDED":
		return "[#FFAB00:black:b][COMMENT][-:-:-]"
	case "CREATED":
		return "[#36B37E:black:b][CREATED][-:-:-]"
	default:
		return "[#7A869A:black][UPDATE][-:-:-]"
	}
}

func buildHeaderContent(space, syncState string, lastSync time.Time) string {
	syncBadge := "[#36B37E:black:b]🟢 " + syncState + "[-:-:-]"
	if strings.Contains(syncState, "SYNC") {
		syncBadge = "[#FFAB00:black:b]🔄 " + syncState + "[-:-:-]"
	}
	timeStr := fmt.Sprintf("[#7A869A]Last sync: %s[-]", lastSync.Format("15:04:05"))

	return fmt.Sprintf(" [#FFFFFF:#2684FF:b] ⚡ JIRA COCKPIT [-:-:-] [#FFFFFF:#172B4D:b] SPACE: %s [-:-:-]  %s  %s",
		space, syncBadge, timeStr)
}

func buildDetailCardText(issue *pkg.Issue) string {
	if issue == nil {
		return "[#7A869A]No issue selected[-]"
	}

	statusBadge := formatStatusBadgeTview(issue.Fields.Status.Name)
	typeStr := issue.Fields.IssueType.Name
	if typeStr == "" {
		typeStr = "Task"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[#FFFFFF:#2684FF:b] %s [-:-:-]  %s  [#7A869A][%s][-]\n", issue.Key, statusBadge, typeStr))
	sb.WriteString(fmt.Sprintf("[white:b]%s[-:-]\n\n", tview.Escape(issue.Fields.Summary)))

	assignee := "Unassigned"
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		assignee = issue.Fields.Assignee.DisplayName
	}
	reporter := "Unknown"
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		reporter = issue.Fields.Reporter.DisplayName
	}
	sb.WriteString(fmt.Sprintf("[#7A869A]👤 Assignee: [white]%s[#7A869A] | 📝 Reporter: [white]%s[-]\n", tview.Escape(assignee), tview.Escape(reporter)))

	if issue.Fields.Description != nil {
		descText := pkg.ExtractTextFromADF(issue.Fields.Description)
		descText = strings.ReplaceAll(descText, "\r\n", "\n")
		descText = strings.ReplaceAll(descText, "\r", "\n")
		descText = strings.TrimSpace(descText)
		if descText != "" {
			sb.WriteString("\n[yellow:b]Description:[-:-]\n")
			sb.WriteString(tview.Escape(descText))
		}
	}

	// Comments Section
	if issue.Fields.Comment != nil && len(issue.Fields.Comment.Comments) > 0 {
		comments := issue.Fields.Comment.Comments
		sb.WriteString(fmt.Sprintf("\n\n[yellow:b]💬 Comments (%d):[-:-]\n", len(comments)))
		for i, c := range comments {
			author := c.Author.DisplayName
			if author == "" {
				author = "Unknown"
			}
			createdTime := c.Created
			if t, err := time.Parse(time.RFC3339, c.Created); err == nil {
				createdTime = t.Format("2006-01-02 15:04")
			} else if len(createdTime) > 16 {
				createdTime = createdTime[:16]
			}

			bodyText := pkg.ExtractTextFromADF(c.Body)
			bodyText = strings.ReplaceAll(bodyText, "\r\n", "\n")
			bodyText = strings.ReplaceAll(bodyText, "\r", "\n")
			bodyText = strings.TrimSpace(bodyText)

			sb.WriteString(fmt.Sprintf("\n[#00B8D9:b]#%d %s[-] [#7A869A](%s)[-]\n", i+1, tview.Escape(author), createdTime))
			if bodyText != "" {
				bodyLines := strings.Split(bodyText, "\n")
				for _, bl := range bodyLines {
					sb.WriteString(fmt.Sprintf("  %s\n", tview.Escape(bl)))
				}
			}
		}
	}

	return sb.String()
}

func buildActivityFeedText(activities []pkg.ActivityEvent) string {
	if len(activities) == 0 {
		return " [#7A869A]No live events yet. Watching for changes...[-]"
	}

	var lines []string
	for _, act := range activities {
		timeStr := fmt.Sprintf("[#7A869A]%s[-]", act.CreatedAt.Format("15:04:05"))
		badge := formatActivityBadgeTview(act.EventType)
		keyStr := fmt.Sprintf("[white:b]%s[-:-]", act.IssueKey)
		detail := tview.Escape(act.Details)
		lines = append(lines, fmt.Sprintf(" %s %s %s: %s", timeStr, badge, keyStr, detail))
	}
	return strings.Join(lines, "\n")
}

func buildDigestText(digest *pkg.CatchupDigest) string {
	if digest == nil {
		return ""
	}
	days := int(digest.AbsenceDuration.Hours() / 24)
	if days < 1 {
		days = 1
	}

	var lines []string
	lines = append(lines, fmt.Sprintf(" [#FFAB00:black:b]🏖️ Welcome Back! 부재 중 업무 변경 브리핑 (최근 %d일간 %d건 변경 감지)[-:-:-]", days, digest.TotalChangedCount))

	if len(digest.NewlyAssigned) > 0 {
		var keys []string
		for _, iss := range digest.NewlyAssigned {
			keys = append(keys, iss.Key)
		}
		lines = append(lines, fmt.Sprintf("  ✨ 내게 새로 할당된 티켓: [white:b]%s[-:-]", strings.Join(keys, ", ")))
	}
	if len(digest.StatusChanges) > 0 {
		var changes []string
		for _, sc := range digest.StatusChanges {
			changes = append(changes, fmt.Sprintf("[%s] %s ➔ %s", sc.IssueKey, sc.OldStatus, sc.NewStatus))
		}
		lines = append(lines, fmt.Sprintf("  🔄 상태 변경된 내 티켓: [white:b]%s[-:-]", strings.Join(changes, ", ")))
	}
	if digest.NewCommentCount > 0 {
		lines = append(lines, fmt.Sprintf("  💬 확인해야 할 신규 코멘트: [white:b]%d건[-:-]", digest.NewCommentCount))
	}
	if len(digest.NewlyAssigned) == 0 && len(digest.StatusChanges) == 0 && digest.NewCommentCount == 0 {
		lines = append(lines, "  ✓ 부재 중 담당 티켓의 중대한 변경 사항이 없습니다.")
	}
	lines = append(lines, " [#7A869A][ESC 키를 눌러 브리핑 창을 닫습니다][-]")

	return strings.Join(lines, "\n")
}

func formatIssueURL(baseURL, key string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/browse/%s", baseURL, strings.ToUpper(key))
}

func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
