package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"tools/jira/internal/i18n"
	"tools/jira/pkg"
)

// ReportCommand generates a project progress and weekly briefing report.
type ReportCommand struct {
	clientProvider ClientProvider
}

// NewReportCommand creates a new ReportCommand instance.
func NewReportCommand(cp ClientProvider) *ReportCommand {
	return &ReportCommand{clientProvider: cp}
}

func (c *ReportCommand) Name() string        { return "report" }
func (c *ReportCommand) Aliases() []string  { return []string{"digest", "briefing", "summary"} }
func (c *ReportCommand) Description() string { return i18n.T("cmd.report.desc") }

// ReportData represents the full aggregated metrics for a report.
type ReportData struct {
	Scope       string            `json:"scope"`
	Period      string            `json:"period"`
	GeneratedAt string            `json:"generated_at"`
	Health      string            `json:"health"`
	HealthEmoji string            `json:"health_emoji"`
	Summary     ReportSummary     `json:"summary"`
	Overdue     []ReportIssueItem `json:"overdue"`
	DueSoon     []ReportIssueItem `json:"due_soon"`
	Completed   []ReportIssueItem `json:"completed"`
	InProgress  []ReportIssueItem `json:"in_progress"`
	Assignees   []AssigneeSummary `json:"assignees"`
	Projects    []ProjectSummary  `json:"projects,omitempty"`
}

// ReportSummary holds high-level counts and ratios.
type ReportSummary struct {
	Total      int     `json:"total"`
	Done       int     `json:"done"`
	InProgress int     `json:"in_progress"`
	ToDo       int     `json:"to_do"`
	DoneRate   float64 `json:"done_rate"`
	Overdue    int     `json:"overdue"`
	DueSoon    int     `json:"due_soon"`
}

// ReportIssueItem represents an issue highlighted in the report.
type ReportIssueItem struct {
	Key       string `json:"key"`
	Project   string `json:"project"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	Assignee  string `json:"assignee"`
	DueDate   string `json:"due_date,omitempty"`
	DiffDays  int    `json:"diff_days"`
	DiffLabel string `json:"diff_label"`
}

// AssigneeSummary holds workload statistics for an individual assignee.
type AssigneeSummary struct {
	Name       string `json:"name"`
	Total      int    `json:"total"`
	InProgress int    `json:"in_progress"`
	ToDo       int    `json:"to_do"`
	Done       int    `json:"done"`
	StatusNote string `json:"status_note"`
}

// ProjectSummary holds breakdown statistics for each project in multi-project scope.
type ProjectSummary struct {
	Key        string  `json:"key"`
	Total      int     `json:"total"`
	Done       int     `json:"done"`
	InProgress int     `json:"in_progress"`
	Overdue    int     `json:"overdue"`
	DoneRate   float64 `json:"done_rate"`
	Bar        string  `json:"bar"`
}

func (c *ReportCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		c.printHelp(stdout)
		return nil
	}

	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var all bool
	var project, assignee, period string

	fs.BoolVar(&all, "all", false, i18n.T("cmd.report.flag_all"))
	fs.BoolVar(&all, "A", false, i18n.T("cmd.report.flag_all"))
	fs.StringVar(&project, "project", "", i18n.T("cmd.report.flag_project"))
	fs.StringVar(&project, "p", "", i18n.T("cmd.report.flag_project"))
	fs.StringVar(&assignee, "assignee", "", i18n.T("cmd.report.flag_assignee"))
	fs.StringVar(&assignee, "a", "", i18n.T("cmd.report.flag_assignee"))
	fs.StringVar(&period, "period", "week", i18n.T("cmd.report.flag_period"))

	if err := fs.Parse(cleanArgs); err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	cfg := client.GetConfig()
	jql, scopeLabel := buildReportJQL(all, project, cfg.ProjectKey, assignee, period)

	issues, err := client.ListIssues(ctx, jql)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.list.err_fetch_issues", err))
	}

	now := time.Now()
	data := aggregateReport(issues, scopeLabel, period, all, now)

	switch format {
	case "json":
		return printReportJSON(stdout, data)
	case "md":
		return printReportMarkdown(stdout, data)
	default:
		return printReportTerminal(stdout, data)
	}
}

func (c *ReportCommand) printHelp(w io.Writer) {
	fmt.Fprintln(w, i18n.T("cmd.report.usage"))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  -all, -A\t%s\n", i18n.T("cmd.report.flag_all"))
	fmt.Fprintf(tw, "  -p, --project\t%s\n", i18n.T("cmd.report.flag_project"))
	fmt.Fprintf(tw, "  -a, --assignee\t%s\n", i18n.T("cmd.report.flag_assignee"))
	fmt.Fprintf(tw, "  --period\t%s\n", i18n.T("cmd.report.flag_period"))
	fmt.Fprintf(tw, "  --md\t%s\n", i18n.T("cmd.common.flag_md"))
	fmt.Fprintf(tw, "  --json\t%s\n", i18n.T("cmd.common.flag_json"))
	tw.Flush()
}

func buildReportJQL(all bool, project, defaultProject, assignee, period string) (string, string) {
	var conditions []string
	var scopeLabel string

	if all {
		scopeLabel = "All Projects"
	} else if project != "" {
		conditions = append(conditions, fmt.Sprintf("project = %s", strings.ToUpper(project)))
		scopeLabel = fmt.Sprintf("Project %s", strings.ToUpper(project))
	} else if defaultProject != "" {
		conditions = append(conditions, fmt.Sprintf("project = %s", strings.ToUpper(defaultProject)))
		scopeLabel = fmt.Sprintf("Project %s", strings.ToUpper(defaultProject))
	}

	switch strings.ToLower(period) {
	case "month":
		conditions = append(conditions, "((statusCategory != Done) OR (resolved >= startOfMonth() OR (updated >= startOfMonth() AND statusCategory = Done)))")
	case "all":
		// No date filter for all
	default: // week
		conditions = append(conditions, "((statusCategory != Done) OR (resolved >= startOfWeek() OR (updated >= startOfWeek() AND statusCategory = Done)))")
	}

	if assignee != "" {
		switch strings.ToLower(assignee) {
		case "me":
			conditions = append(conditions, "assignee = currentUser()")
		case "none", "null", "empty":
			conditions = append(conditions, "assignee is EMPTY")
		default:
			conditions = append(conditions, fmt.Sprintf("assignee = %q", assignee))
		}
	}

	jql := strings.Join(conditions, " AND ")
	if jql != "" {
		jql += " ORDER BY duedate ASC, updated DESC"
	} else {
		jql = "ORDER BY duedate ASC, updated DESC"
	}

	return jql, scopeLabel
}

func aggregateReport(issues []pkg.Issue, scopeLabel, period string, multiProject bool, now time.Time) ReportData {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	data := ReportData{
		Scope:       scopeLabel,
		Period:      period,
		GeneratedAt: now.Format("2006-01-02 15:04"),
	}

	assigneeMap := make(map[string]*AssigneeSummary)
	projectMap := make(map[string]*ProjectSummary)

	for _, issue := range issues {
		data.Summary.Total++

		cat := getStatusCategory(issue)
		isDone := cat == "done"
		isInProgress := cat == "indeterminate"

		assigneeName := i18n.T("fmt.common.unassigned")
		if issue.Fields.Assignee != nil && issue.Fields.Assignee.DisplayName != "" {
			assigneeName = issue.Fields.Assignee.DisplayName
		}

		projKey := issue.Fields.Project.Key
		if projKey == "" && strings.Contains(issue.Key, "-") {
			projKey = strings.Split(issue.Key, "-")[0]
		}

		// Ensure map entries
		if _, ok := assigneeMap[assigneeName]; !ok {
			assigneeMap[assigneeName] = &AssigneeSummary{Name: assigneeName}
		}
		as := assigneeMap[assigneeName]
		as.Total++

		if _, ok := projectMap[projKey]; !ok {
			projectMap[projKey] = &ProjectSummary{Key: projKey}
		}
		ps := projectMap[projKey]
		ps.Total++

		// Due date evaluation
		diffDays := 0
		hasDue := false
		diffLabel := ""
		if issue.Fields.DueDate != "" {
			if dueT, err := time.Parse("2006-01-02", issue.Fields.DueDate); err == nil {
				dueDay := time.Date(dueT.Year(), dueT.Month(), dueT.Day(), 0, 0, 0, 0, now.Location())
				diffDays = int(dueDay.Sub(today).Hours() / 24)
				hasDue = true
				if diffDays < 0 {
					diffLabel = fmt.Sprintf("+%d일 지연", -diffDays)
				} else if diffDays == 0 {
					diffLabel = "오늘 마감 (D-Day)"
				} else {
					diffLabel = fmt.Sprintf("D-%d", diffDays)
				}
			}
		}

		item := ReportIssueItem{
			Key:       issue.Key,
			Project:   projKey,
			Summary:   issue.Fields.Summary,
			Status:    issue.Fields.Status.Name,
			Assignee:  assigneeName,
			DueDate:   issue.Fields.DueDate,
			DiffDays:  diffDays,
			DiffLabel: diffLabel,
		}

		if isDone {
			data.Summary.Done++
			as.Done++
			ps.Done++
			data.Completed = append(data.Completed, item)
		} else {
			if isInProgress {
				data.Summary.InProgress++
				as.InProgress++
				ps.InProgress++
				data.InProgress = append(data.InProgress, item)
			} else {
				data.Summary.ToDo++
				as.ToDo++
			}

			if hasDue {
				if diffDays < 0 {
					data.Summary.Overdue++
					ps.Overdue++
					data.Overdue = append(data.Overdue, item)
				} else if diffDays <= 3 {
					data.Summary.DueSoon++
					data.DueSoon = append(data.DueSoon, item)
				}
			}
		}
	}

	if data.Summary.Total > 0 {
		data.Summary.DoneRate = float64(data.Summary.Done) / float64(data.Summary.Total) * 100.0
	}

	// Calculate Health
	if data.Summary.Overdue == 0 {
		data.Health = "NORMAL"
	} else if data.Summary.Overdue <= 2 {
		data.Health = "WARNING"
	} else {
		data.Health = "CRITICAL"
	}

	// Sort and format Assignees
	for _, as := range assigneeMap {
		if as.Name == i18n.T("fmt.common.unassigned") && as.InProgress+as.ToDo > 0 {
			as.StatusNote = "할당 필요"
		} else if as.InProgress >= 4 {
			as.StatusNote = "과부하 주의"
		} else {
			as.StatusNote = "정상"
		}
		data.Assignees = append(data.Assignees, *as)
	}
	sort.Slice(data.Assignees, func(i, j int) bool {
		remI := data.Assignees[i].InProgress + data.Assignees[i].ToDo
		remJ := data.Assignees[j].InProgress + data.Assignees[j].ToDo
		if remI != remJ {
			return remI > remJ
		}
		return data.Assignees[i].Name < data.Assignees[j].Name
	})

	// Sort and format Projects if multiProject
	if multiProject || len(projectMap) > 1 {
		for _, ps := range projectMap {
			if ps.Total > 0 {
				ps.DoneRate = float64(ps.Done) / float64(ps.Total) * 100.0
			}
			ps.Bar = makeProgressBar(ps.DoneRate)
			data.Projects = append(data.Projects, *ps)
		}
		sort.Slice(data.Projects, func(i, j int) bool {
			return data.Projects[i].Key < data.Projects[j].Key
		})
	}

	return data
}

func getStatusCategory(issue pkg.Issue) string {
	if issue.Fields.Status.StatusCategory != nil && issue.Fields.Status.StatusCategory.Key != "" {
		return strings.ToLower(issue.Fields.Status.StatusCategory.Key)
	}
	name := strings.ToLower(issue.Fields.Status.Name)
	switch name {
	case "done", "완료", "closed", "resolved":
		return "done"
	case "in progress", "진행 중", "doing", "검토 중":
		return "indeterminate"
	default:
		return "new"
	}
}

func makeProgressBar(rate float64) string {
	if rate < 0 {
		rate = 0
	}
	if rate > 100 {
		rate = 100
	}
	totalBlocks := 10
	filled := int((rate + 5) / 10)
	if filled > totalBlocks {
		filled = totalBlocks
	}
	empty := totalBlocks - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s] %.0f%%", bar, rate)
}

func printReportTerminal(w io.Writer, data ReportData) error {
	fmt.Fprintf(w, "[REPORT] 주간 업무 브리핑 [%s] (%s 기준)\n", data.Scope, data.GeneratedAt)
	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintf(w, "상태: %s  | 총 %d건 중 %d건 완료 (%.1f%%) | 지연: %d건 | 마감임박: %d건\n",
		data.Health, data.Summary.Total, data.Summary.Done, data.Summary.DoneRate, data.Summary.Overdue, data.Summary.DueSoon)
	fmt.Fprintln(w, "--------------------------------------------------------------------------------")

	// 1. Risks & Overdue
	if len(data.Overdue) > 0 {
		fmt.Fprintf(w, "\n[주의 및 지연 작업 (%d건)]\n", len(data.Overdue))
		for _, item := range data.Overdue {
			fmt.Fprintf(w, "  • [%s] %s (담당: %s, 마감: %s, %s)\n", item.Key, item.Summary, item.Assignee, item.DueDate, item.DiffLabel)
		}
	}

	// 2. Due Soon
	if len(data.DueSoon) > 0 {
		fmt.Fprintf(w, "\n[마감 임박 작업 (%d건)]\n", len(data.DueSoon))
		for _, item := range data.DueSoon {
			fmt.Fprintf(w, "  • [%s] %s (담당: %s, 마감: %s, %s)\n", item.Key, item.Summary, item.Assignee, item.DueDate, item.DiffLabel)
		}
	}

	// 3. Completed
	if len(data.Completed) > 0 {
		fmt.Fprintf(w, "\n[완료된 작업 (%d건)]\n", len(data.Completed))
		limit := len(data.Completed)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			item := data.Completed[i]
			fmt.Fprintf(w, "  • [%s] %s (담당: %s)\n", item.Key, item.Summary, item.Assignee)
		}
		if len(data.Completed) > 5 {
			fmt.Fprintf(w, "  ... 외 %d건 완료\n", len(data.Completed)-5)
		}
	}

	// 4. In Progress
	if len(data.InProgress) > 0 {
		fmt.Fprintf(w, "\n[현재 진행 중인 작업 (%d건)]\n", len(data.InProgress))
		for _, item := range data.InProgress {
			dueInfo := ""
			if item.DueDate != "" {
				dueInfo = fmt.Sprintf(", %s", item.DiffLabel)
			}
			fmt.Fprintf(w, "  • [%s] %s (담당: %s%s)\n", item.Key, item.Summary, item.Assignee, dueInfo)
		}
	}

	// 5. Assignees
	if len(data.Assignees) > 0 {
		fmt.Fprintln(w, "\n[담당자별 워크로드]")
		for _, as := range data.Assignees {
			rem := as.InProgress + as.ToDo
			fmt.Fprintf(w, "  • %s: 잔여 %d건 (진행 %d, 대기 %d) - %s\n", as.Name, rem, as.InProgress, as.ToDo, as.StatusNote)
		}
	}

	// 6. Projects Breakdown
	if len(data.Projects) > 0 {
		fmt.Fprintln(w, "\n[프로젝트별 진척 현황]")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  프로젝트\t전체\t완료\t진행\t지연\t진척도")
		for _, ps := range data.Projects {
			fmt.Fprintf(tw, "  %s\t%d\t%d\t%d\t%d\t%s\n", ps.Key, ps.Total, ps.Done, ps.InProgress, ps.Overdue, ps.Bar)
		}
		tw.Flush()
	}

	fmt.Fprintln(w, "================================================================================")
	fmt.Fprintln(w, "팁: Markdown 출력 'jira report --md' | JSON 출력 'jira report --json'")
	return nil
}

func printReportMarkdown(w io.Writer, data ReportData) error {
	fmt.Fprintf(w, "# 주간 업무 브리핑 리포트 (%s 기준)\n", data.GeneratedAt)
	fmt.Fprintf(w, "> **대상**: %s | **상태**: %s (지연: %d건, 마감임박: %d건)\n\n",
		data.Scope, data.Health, data.Summary.Overdue, data.Summary.DueSoon)
	fmt.Fprintln(w, "---")

	// 1. Executive Summary Table
	fmt.Fprintln(w, "\n### 1. 종합 진척 현황 (Executive Summary)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "| 전체 이슈 | 완료 (Done) | 진행 중 (In Progress) | 대기 (To Do) | 지연 (Overdue) | 마감 임박 (Due Soon) | 완료율 |")
	fmt.Fprintln(w, "| :---: | :---: | :---: | :---: | :---: | :---: | :---: |")
	fmt.Fprintf(w, "| %d | %d | %d | %d | **%d** | %d | **%.1f%%** |\n",
		data.Summary.Total, data.Summary.Done, data.Summary.InProgress, data.Summary.ToDo, data.Summary.Overdue, data.Summary.DueSoon, data.Summary.DoneRate)

	// 2. Risks & Overdue Table
	fmt.Fprintln(w, "\n---")
	fmt.Fprintln(w, "\n### 2. 위험 및 주의 작업 (Risks & Overdue)")
	fmt.Fprintln(w, "")
	if len(data.Overdue) == 0 && len(data.DueSoon) == 0 {
		fmt.Fprintln(w, "_현재 지연되거나 마감 임박한 위험 이슈가 없습니다._")
	} else {
		fmt.Fprintln(w, "| 이슈 키 | 프로젝트 | 요약 | 담당자 | 마감일 | 구분 |")
		fmt.Fprintln(w, "| :--- | :---: | :--- | :--- | :---: | :--- |")
		for _, item := range data.Overdue {
			fmt.Fprintf(w, "| **%s** | %s | %s | %s | %s | **지연 (%s)** |\n",
				item.Key, item.Project, item.Summary, item.Assignee, item.DueDate, item.DiffLabel)
		}
		for _, item := range data.DueSoon {
			fmt.Fprintf(w, "| **%s** | %s | %s | %s | %s | 임박 (%s) |\n",
				item.Key, item.Project, item.Summary, item.Assignee, item.DueDate, item.DiffLabel)
		}
	}

	// 3. Completed Table
	fmt.Fprintln(w, "\n---")
	fmt.Fprintln(w, "\n### 3. 이번 주 완료 성과 (Completed This Week)")
	fmt.Fprintln(w, "")
	if len(data.Completed) == 0 {
		fmt.Fprintln(w, "_이번 기간에 완료된 티켓이 없습니다._")
	} else {
		fmt.Fprintln(w, "| 이슈 키 | 프로젝트 | 요약 | 담당자 | 상태 |")
		fmt.Fprintln(w, "| :--- | :---: | :--- | :--- | :---: |")
		for _, item := range data.Completed {
			fmt.Fprintf(w, "| **%s** | %s | %s | %s | 완료 |\n",
				item.Key, item.Project, item.Summary, item.Assignee)
		}
	}

	// 4. In Progress Table
	fmt.Fprintln(w, "\n---")
	fmt.Fprintln(w, "\n### 4. 현재 진행 중인 작업 (In Progress)")
	fmt.Fprintln(w, "")
	if len(data.InProgress) == 0 {
		fmt.Fprintln(w, "_현재 진행 중인 작업이 없습니다._")
	} else {
		fmt.Fprintln(w, "| 이슈 키 | 프로젝트 | 요약 | 담당자 | 마감일 |")
		fmt.Fprintln(w, "| :--- | :---: | :--- | :--- | :---: |")
		for _, item := range data.InProgress {
			due := item.DueDate
			if due != "" && item.DiffLabel != "" {
				due = fmt.Sprintf("%s (%s)", item.DueDate, item.DiffLabel)
			} else if due == "" {
				due = "-"
			}
			fmt.Fprintf(w, "| **%s** | %s | %s | %s | %s |\n",
				item.Key, item.Project, item.Summary, item.Assignee, due)
		}
	}

	// 5. Workload by Assignee Table
	if len(data.Assignees) > 0 {
		fmt.Fprintln(w, "\n---")
		fmt.Fprintln(w, "\n### 5. 담당자별 워크로드 (Resource Allocation)")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "| 담당자 | 진행 중 | 대기 | 합계 | 상태 |")
		fmt.Fprintln(w, "| :--- | :---: | :---: | :---: | :--- |")
		for _, as := range data.Assignees {
			fmt.Fprintf(w, "| **%s** | %d | %d | %d | %s |\n", as.Name, as.InProgress, as.ToDo, as.InProgress+as.ToDo, as.StatusNote)
		}
	}

	// 6. Cross-Project Breakdown Table
	if len(data.Projects) > 0 {
		fmt.Fprintln(w, "\n---")
		fmt.Fprintln(w, "\n### 6. 프로젝트별 현황 (Project Breakdown)")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "| 프로젝트 키 | 전체 | 완료 | 진행 중 | 지연 | 진행률 |")
		fmt.Fprintln(w, "| :--- | :---: | :---: | :---: | :---: | :--- |")
		for _, ps := range data.Projects {
			fmt.Fprintf(w, "| **%s** | %d | %d | %d | %d | `%s` |\n", ps.Key, ps.Total, ps.Done, ps.InProgress, ps.Overdue, ps.Bar)
		}
	}

	return nil
}

func printReportJSON(w io.Writer, data ReportData) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(bytes))
	return err
}
