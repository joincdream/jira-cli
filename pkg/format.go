package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"tools/jira/internal/i18n"
)

// PrintIssuesTable prints a formatted table of issues to w.
func PrintIssuesTable(w io.Writer, issues []Issue) {
	if len(issues) == 0 {
		fmt.Fprintln(w, i18n.T("fmt.table.no_issues"))
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, i18n.T("fmt.table.header"))
	fmt.Fprintln(tw, "---\t----\t------\t------\t--------\t--------\t-------")

	for _, issue := range issues {
		key := issue.Key
		issueType := issue.Fields.IssueType.Name
		status := formatStatusBadge(issue.Fields.Status.Name)
		labels := "-"
		if len(issue.Fields.Labels) > 0 {
			labels = strings.Join(issue.Fields.Labels, ",")
		}
		assignee := i18n.T("fmt.common.unassigned")
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

	assignee := i18n.T("fmt.common.unassigned")
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		assignee = fmt.Sprintf("%s (%s)", issue.Fields.Assignee.DisplayName, issue.Fields.Assignee.EmailAddress)
	}

	reporter := i18n.T("fmt.common.unassigned")
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		reporter = fmt.Sprintf("%s (%s)", issue.Fields.Reporter.DisplayName, issue.Fields.Reporter.EmailAddress)
	}

	dueDate := "-"
	if issue.Fields.DueDate != "" {
		dueDate = issue.Fields.DueDate
	}

	labels := i18n.T("fmt.common.none")
	if len(issue.Fields.Labels) > 0 {
		labels = strings.Join(issue.Fields.Labels, ", ")
	}

	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_project", issue.Fields.Project.Name, issue.Fields.Project.Key))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_type", issue.Fields.IssueType.Name))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_status", issue.Fields.Status.Name))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_labels", labels))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_assignee", assignee))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_reporter", reporter))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_due", dueDate))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_created", issue.Fields.Created))
	fmt.Fprintf(w, i18n.Sprintf("fmt.detail.field_updated", issue.Fields.Updated))

	// Description
	descText := ExtractTextFromADF(issue.Fields.Description)
	fmt.Fprintf(w, i18n.T("fmt.detail.header_desc"))
	if strings.TrimSpace(descText) == "" {
		fmt.Fprintf(w, i18n.T("fmt.detail.desc_empty"))
	} else {
		for _, line := range strings.Split(descText, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}

	// Subtasks
	if len(issue.Fields.Subtasks) > 0 {
		fmt.Fprintf(w, i18n.Sprintf("fmt.detail.header_subtasks", len(issue.Fields.Subtasks)))
		for _, st := range issue.Fields.Subtasks {
			fmt.Fprintf(w, "  - [%s] %s (%s)\n", st.Key, st.Fields.Summary, st.Fields.Status.Name)
		}
	}

	// Comments
	if issue.Fields.Comment != nil && len(issue.Fields.Comment.Comments) > 0 {
		fmt.Fprintf(w, i18n.Sprintf("fmt.detail.header_comments", len(issue.Fields.Comment.Comments)))
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
		fmt.Fprintln(w, i18n.T("fmt.md.no_issues"))
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
		assignee := i18n.T("fmt.common.unassigned")
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

	assignee := i18n.T("fmt.common.unassigned")
	if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
		assignee = fmt.Sprintf("%s (%s)", issue.Fields.Assignee.DisplayName, issue.Fields.Assignee.EmailAddress)
	}

	reporter := i18n.T("fmt.common.unassigned")
	if issue.Fields.Reporter != nil && issue.Fields.Reporter.DisplayName != "" {
		reporter = fmt.Sprintf("%s (%s)", issue.Fields.Reporter.DisplayName, issue.Fields.Reporter.EmailAddress)
	}

	dueDate := "-"
	if issue.Fields.DueDate != "" {
		dueDate = issue.Fields.DueDate
	}

	labels := i18n.T("fmt.common.none")
	if len(issue.Fields.Labels) > 0 {
		var wrapped []string
		for _, l := range issue.Fields.Labels {
			wrapped = append(wrapped, fmt.Sprintf("`%s`", l))
		}
		labels = strings.Join(wrapped, ", ")
	}

	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_project", issue.Fields.Project.Name, issue.Fields.Project.Key))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_type", issue.Fields.IssueType.Name))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_status", issue.Fields.Status.Name))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_labels", labels))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_assignee", assignee))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_reporter", reporter))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_due", dueDate))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_created", issue.Fields.Created))
	fmt.Fprintf(w, i18n.Sprintf("fmt.md.field_updated", issue.Fields.Updated))

	// Description
	descText := ExtractTextFromADF(issue.Fields.Description)
	fmt.Fprintf(w, i18n.T("fmt.md.header_desc"))
	if strings.TrimSpace(descText) == "" {
		fmt.Fprintf(w, i18n.T("fmt.md.desc_empty"))
	} else {
		fmt.Fprintf(w, "%s\n", descText)
	}

	// Subtasks
	if len(issue.Fields.Subtasks) > 0 {
		fmt.Fprintf(w, i18n.Sprintf("fmt.md.header_subtasks", len(issue.Fields.Subtasks)))
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
		fmt.Fprintf(w, i18n.Sprintf("fmt.md.header_comments", len(issue.Fields.Comment.Comments)))
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

