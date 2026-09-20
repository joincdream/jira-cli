package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// PrintIssuesTable prints a formatted table of issues to w.
func PrintIssuesTable(w io.Writer, issues []Issue) {
	if len(issues) == 0 {
		fmt.Fprintln(w, "등록된 Jira 티켓이 없습니다.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "KEY\tTYPE\tSTATUS\tLABELS\tASSIGNEE\tDUE DATE\tSUMMARY")
	fmt.Fprintln(tw, "---\t----\t------\t------\t--------\t--------\t-------")

	for _, issue := range issues {
		key := issue.Key
		issueType := issue.Fields.IssueType.Name
		status := formatStatusBadge(issue.Fields.Status.Name)
		labels := "-"
		if len(issue.Fields.Labels) > 0 {
			labels = strings.Join(issue.Fields.Labels, ",")
		}
		assignee := "미지정"
		if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
			assignee = issue.Fields.Assignee.DisplayName
		}
		dueDate := "-"
		if issue.Fields.DueDate != "" {
			dueDate = issue.Fields.DueDate
		}
		summary := issue.Fields.Summary

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", key, issueType, status, labels, assignee, dueDate, summary)
	}

	tw.Flush()
}

func formatStatusBadge(status string) string {
	switch strings.ToLower(status) {
	case "해야 할 일", "to do", "backlog", "백로그":
		return "[To Do]"
	case "진행 중", "in progress", "doing":
		return "[In Progress *]"
	case "완료", "done", "closed", "resolved":
		return "[Done OK]"
	default:
		return fmt.Sprintf("[%s]", status)
	}
}

// PrintIssueDetail prints a detailed view of a single issue.
func PrintIssueDetail(w io.Writer, issue *Issue) {
	fmt.Fprintf(w, "================================================================================\n")
	fmt.Fprintf(w, "🎫 [%s] %s\n", issue.Key, issue.Fields.Summary)
	fmt.Fprintf(w, "================================================================================\n")

	assignee := "미지정"
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		assignee = fmt.Sprintf("%s (%s)", issue.Fields.Assignee.DisplayName, issue.Fields.Assignee.EmailAddress)
	}

	reporter := "미지정"
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		reporter = fmt.Sprintf("%s (%s)", issue.Fields.Reporter.DisplayName, issue.Fields.Reporter.EmailAddress)
	}

	dueDate := "-"
	if issue.Fields.DueDate != "" {
		dueDate = issue.Fields.DueDate
	}

	labels := "(없음)"
	if len(issue.Fields.Labels) > 0 {
		labels = strings.Join(issue.Fields.Labels, ", ")
	}

	fmt.Fprintf(w, "• 프로젝트:    %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key)
	fmt.Fprintf(w, "• 이슈 유형:   %s\n", issue.Fields.IssueType.Name)
	fmt.Fprintf(w, "• 상태:        %s\n", issue.Fields.Status.Name)
	fmt.Fprintf(w, "• 라벨(Tags):  %s\n", labels)
	fmt.Fprintf(w, "• 담당자:      %s\n", assignee)
	fmt.Fprintf(w, "• 보고자:      %s\n", reporter)
	fmt.Fprintf(w, "• 마감일:      %s\n", dueDate)
	fmt.Fprintf(w, "• 생성일시:    %s\n", issue.Fields.Created)
	fmt.Fprintf(w, "• 수정일시:    %s\n", issue.Fields.Updated)

	// Description
	descText := ExtractTextFromADF(issue.Fields.Description)
	fmt.Fprintf(w, "\n📄 [상세 설명 (Description)]\n")
	if strings.TrimSpace(descText) == "" {
		fmt.Fprintf(w, "  (설명이 비어 있습니다)\n")
	} else {
		for _, line := range strings.Split(descText, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	// Subtasks
	if len(issue.Fields.Subtasks) > 0 {
		fmt.Fprintf(w, "\n🔹 [하위 작업 (Subtasks: %d개)]\n", len(issue.Fields.Subtasks))
		for _, st := range issue.Fields.Subtasks {
			fmt.Fprintf(w, "  - [%s] %s (%s)\n", st.Key, st.Fields.Summary, st.Fields.Status.Name)
		}
	}

	// Comments
	if issue.Fields.Comment != nil && len(issue.Fields.Comment.Comments) > 0 {
		fmt.Fprintf(w, "\n💬 [코멘트 (%d개)]\n", len(issue.Fields.Comment.Comments))
		for idx, c := range issue.Fields.Comment.Comments {
			cText := ExtractTextFromADF(c.Body)
			fmt.Fprintf(w, "  #%d [%s | %s]\n", idx+1, c.Author.DisplayName, c.Created)
			for _, line := range strings.Split(cText, "\n") {
				fmt.Fprintf(w, "    %s\n", line)
			}
		}
	}

	fmt.Fprintf(w, "--------------------------------------------------------------------------------\n")
}

// PrintIssuesJSON writes issues formatted as indented JSON.
func PrintIssuesJSON(w io.Writer, issues []Issue) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(issues)
}

// PrintIssuesMarkdown writes issues formatted as a Markdown table.
func PrintIssuesMarkdown(w io.Writer, issues []Issue) {
	if len(issues) == 0 {
		fmt.Fprintln(w, "_등록된 Jira 티켓이 없습니다._")
		return
	}

	fmt.Fprintln(w, "| Key | Type | Status | Labels | Assignee | Due Date | Summary |")
	fmt.Fprintln(w, "| :--- | :--- | :--- | :--- | :--- | :--- | :--- |")

	for _, issue := range issues {
		key := issue.Key
		issueType := issue.Fields.IssueType.Name
		status := issue.Fields.Status.Name
		labels := "-"
		if len(issue.Fields.Labels) > 0 {
			labels = strings.Join(issue.Fields.Labels, ", ")
		}
		assignee := "미지정"
		if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
			assignee = issue.Fields.Assignee.DisplayName
		}
		dueDate := "-"
		if issue.Fields.DueDate != "" {
			dueDate = issue.Fields.DueDate
		}
		summary := strings.ReplaceAll(issue.Fields.Summary, "|", "\\|")

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s |\n", key, issueType, status, labels, assignee, dueDate, summary)
	}
}

// PrintIssueJSON writes a single issue formatted as indented JSON.
func PrintIssueJSON(w io.Writer, issue *Issue) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(issue)
}

// PrintIssueMarkdown writes a single issue formatted as a rich Markdown document for LLM / humans.
func PrintIssueMarkdown(w io.Writer, issue *Issue) {
	fmt.Fprintf(w, "# [%s] %s\n\n", issue.Key, issue.Fields.Summary)

	assignee := "미지정"
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		assignee = fmt.Sprintf("%s (%s)", issue.Fields.Assignee.DisplayName, issue.Fields.Assignee.EmailAddress)
	}

	reporter := "미지정"
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		reporter = fmt.Sprintf("%s (%s)", issue.Fields.Reporter.DisplayName, issue.Fields.Reporter.EmailAddress)
	}

	dueDate := "-"
	if issue.Fields.DueDate != "" {
		dueDate = issue.Fields.DueDate
	}

	labels := "(없음)"
	if len(issue.Fields.Labels) > 0 {
		var wrapped []string
		for _, l := range issue.Fields.Labels {
			wrapped = append(wrapped, fmt.Sprintf("`%s`", l))
		}
		labels = strings.Join(wrapped, ", ")
	}

	fmt.Fprintf(w, "- **프로젝트**: %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key)
	fmt.Fprintf(w, "- **이슈 유형**: %s\n", issue.Fields.IssueType.Name)
	fmt.Fprintf(w, "- **상태**: %s\n", issue.Fields.Status.Name)
	fmt.Fprintf(w, "- **라벨**: %s\n", labels)
	fmt.Fprintf(w, "- **담당자**: %s\n", assignee)
	fmt.Fprintf(w, "- **보고자**: %s\n", reporter)
	fmt.Fprintf(w, "- **마감일**: %s\n", dueDate)
	fmt.Fprintf(w, "- **생성일시**: %s\n", issue.Fields.Created)
	fmt.Fprintf(w, "- **수정일시**: %s\n", issue.Fields.Updated)

	// Description
	descText := ExtractTextFromADF(issue.Fields.Description)
	fmt.Fprintf(w, "\n## 📄 상세 설명 (Description)\n\n")
	if strings.TrimSpace(descText) == "" {
		fmt.Fprintf(w, "_설명이 비어 있습니다._\n")
	} else {
		fmt.Fprintf(w, "%s\n", descText)
	}

	// Subtasks
	if len(issue.Fields.Subtasks) > 0 {
		fmt.Fprintf(w, "\n## 🔹 하위 작업 (Subtasks: %d개)\n\n", len(issue.Fields.Subtasks))
		for _, st := range issue.Fields.Subtasks {
			check := " "
			lowerStatus := strings.ToLower(st.Fields.Status.Name)
			if lowerStatus == "완료" || lowerStatus == "done" || lowerStatus == "closed" || lowerStatus == "resolved" {
				check = "x"
			}
			fmt.Fprintf(w, "- [%s] [%s] %s (%s)\n", check, st.Key, st.Fields.Summary, st.Fields.Status.Name)
		}
	}

	// Comments
	if issue.Fields.Comment != nil && len(issue.Fields.Comment.Comments) > 0 {
		fmt.Fprintf(w, "\n## 💬 코멘트 (%d개)\n\n", len(issue.Fields.Comment.Comments))
		for idx, c := range issue.Fields.Comment.Comments {
			cText := ExtractTextFromADF(c.Body)
			fmt.Fprintf(w, "### #%d %s (%s)\n\n", idx+1, c.Author.DisplayName, c.Created)
			for _, line := range strings.Split(cText, "\n") {
				fmt.Fprintf(w, "> %s\n", line)
			}
			fmt.Fprintln(w)
		}
	}
}

