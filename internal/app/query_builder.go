package app

import (
	"fmt"
	"strings"
	"time"
)

// ListFilterOptions holds CLI flags used to construct a JQL query.
type ListFilterOptions struct {
	All      bool
	Due      string
	Status   string
	Assignee string
}

// parseDueFilter parses a due date expression into a JQL clause.
// Supports: today, <=today, >=today, tomorrow, <=tomorrow, week, YYYY-MM-DD, <=YYYY-MM-DD, etc.
func parseDueFilter(expr string) (string, error) {
	s := strings.TrimSpace(expr)
	if s == "" {
		return "", nil
	}

	op := "="
	val := s
	if strings.HasPrefix(s, "<=") {
		op = "<="
		val = strings.TrimSpace(strings.TrimPrefix(s, "<="))
	} else if strings.HasPrefix(s, ">=") {
		op = ">="
		val = strings.TrimSpace(strings.TrimPrefix(s, ">="))
	} else if strings.HasPrefix(s, "<") {
		op = "<"
		val = strings.TrimSpace(strings.TrimPrefix(s, "<"))
	} else if strings.HasPrefix(s, ">") {
		op = ">"
		val = strings.TrimSpace(strings.TrimPrefix(s, ">"))
	} else if strings.HasPrefix(s, "=") {
		op = "="
		val = strings.TrimSpace(strings.TrimPrefix(s, "="))
	}

	lower := strings.ToLower(val)
	switch lower {
	case "today":
		if op == "<=" {
			return "duedate <= endOfDay()", nil
		} else if op == ">=" {
			return "duedate >= startOfDay()", nil
		} else if op == "<" {
			return "duedate < startOfDay()", nil
		} else if op == ">" {
			return "duedate > endOfDay()", nil
		}
		return "duedate = endOfDay()", nil
	case "tomorrow":
		if op == "<=" {
			return "duedate <= endOfDay('1d')", nil
		} else if op == ">=" {
			return "duedate >= startOfDay('1d')", nil
		}
		return "duedate = endOfDay('1d')", nil
	case "week":
		if op == "<=" || op == "=" {
			return "duedate <= endOfWeek()", nil
		} else if op == ">=" {
			return "duedate >= startOfWeek()", nil
		}
		return fmt.Sprintf("duedate %s endOfWeek()", op), nil
	default:
		// Attempt YYYY-MM-DD parsing
		if _, err := time.Parse("2006-01-02", val); err == nil {
			return fmt.Sprintf("duedate %s \"%s\"", op, val), nil
		}
		return "", fmt.Errorf("지원하지 않는 마감일(--due) 형식입니다: '%s' (지원: today, <=today, tomorrow, <=tomorrow, week, YYYY-MM-DD, <=YYYY-MM-DD)", expr)
	}
}

// parseStatusFilter maps a status option to a JQL clause.
// Default (empty) or "open" maps to statusCategory != Done.
// "done" maps to statusCategory = Done.
// "all" or "*" skips the status filter.
func parseStatusFilter(status string) string {
	s := strings.TrimSpace(status)
	if s == "" || strings.EqualFold(s, "open") {
		return "statusCategory != Done"
	}
	if strings.EqualFold(s, "done") {
		return "statusCategory = Done"
	}
	if strings.EqualFold(s, "all") || s == "*" {
		return ""
	}
	return fmt.Sprintf("status = \"%s\"", s)
}

// parseAssigneeFilter maps an assignee option to a JQL clause.
// "me", "@me", "currentUser()" maps to assignee = currentUser().
// "none", "unassigned", "empty" maps to assignee is EMPTY.
// Other values map to assignee = "<value>".
func parseAssigneeFilter(assignee string) string {
	s := strings.TrimSpace(assignee)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if lower == "me" || lower == "@me" || lower == "currentuser()" {
		return "assignee = currentUser()"
	}
	if lower == "none" || lower == "unassigned" || lower == "empty" {
		return "assignee is EMPTY"
	}
	return fmt.Sprintf("assignee = \"%s\"", s)
}

// BuildFilterJQL constructs a JQL query string based on the given project and options.
func BuildFilterJQL(defaultProject string, opts ListFilterOptions) (string, error) {
	var clauses []string

	// 1. Project scope
	trimmedProject := strings.TrimSpace(defaultProject)
	if !opts.All && trimmedProject != "" && trimmedProject != "*" && !strings.EqualFold(trimmedProject, "all") {
		clauses = append(clauses, fmt.Sprintf("project = %s", trimmedProject))
	}

	// 2. Status
	if statusClause := parseStatusFilter(opts.Status); statusClause != "" {
		clauses = append(clauses, statusClause)
	}

	// 3. Due date
	if opts.Due != "" {
		dueClause, err := parseDueFilter(opts.Due)
		if err != nil {
			return "", err
		}
		if dueClause != "" {
			clauses = append(clauses, dueClause)
		}
	}

	// 4. Assignee
	if assigneeClause := parseAssigneeFilter(opts.Assignee); assigneeClause != "" {
		clauses = append(clauses, assigneeClause)
	}

	jql := strings.Join(clauses, " AND ")
	orderClause := "ORDER BY duedate ASC, priority DESC, created DESC"
	if jql != "" {
		jql += " " + orderClause
	} else {
		jql = orderClause
	}

	return jql, nil
}
