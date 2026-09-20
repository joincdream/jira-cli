package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"tools/jira/pkg"
)

// ListCommand lists issues based on JQL.
type ListCommand struct {
	clientProvider ClientProvider
}

func NewListCommand(cp ClientProvider) *ListCommand {
	return &ListCommand{clientProvider: cp}
}

func (c *ListCommand) Name() string        { return "list" }
func (c *ListCommand) Aliases() []string  { return []string{"ls"} }
func (c *ListCommand) Description() string { return "Jira 이슈 목록을 조회합니다." }

// parseOutputFormat extracts output format (--json, --md, -o, --output) from args.
// Supported formats: "table" (default), "json", "md" (or "markdown").
func parseOutputFormat(args []string) (string, []string, error) {
	var cleanArgs []string
	format := "table"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			format = "json"
			continue
		}
		if arg == "--md" || arg == "--markdown" {
			format = "md"
			continue
		}
		if arg == "-o" || arg == "--output" {
			if i+1 < len(args) {
				format = strings.ToLower(args[i+1])
				i++
				continue
			}
			return "", nil, fmt.Errorf("-o / --output 플래그에 포맷(table, json, md)을 지정하세요")
		}
		if strings.HasPrefix(arg, "-o=") {
			format = strings.ToLower(strings.TrimPrefix(arg, "-o="))
			continue
		}
		if strings.HasPrefix(arg, "--output=") {
			format = strings.ToLower(strings.TrimPrefix(arg, "--output="))
			continue
		}
		cleanArgs = append(cleanArgs, arg)
	}

	if format == "markdown" {
		format = "md"
	}

	if format != "table" && format != "json" && format != "md" {
		return "", nil, fmt.Errorf("지원하지 않는 출력 포맷: '%s' (지원 포맷: table, json, md)", format)
	}

	return format, cleanArgs, nil
}

func (c *ListCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	jql := ""
	if len(cleanArgs) > 0 {
		jql = strings.Join(cleanArgs, " ")
	}

	issues, err := client.ListIssues(ctx, jql)
	if err != nil {
		return fmt.Errorf("이슈 목록 조회 실패: %w", err)
	}

	switch format {
	case "json":
		return pkg.PrintIssuesJSON(stdout, issues)
	case "md":
		pkg.PrintIssuesMarkdown(stdout, issues)
		return nil
	default:
		pkg.PrintIssuesTable(stdout, issues)
		return nil
	}
}

// GetCommand displays detail and comments for a single issue.
type GetCommand struct {
	clientProvider ClientProvider
}

func NewGetCommand(cp ClientProvider) *GetCommand {
	return &GetCommand{clientProvider: cp}
}

func (c *GetCommand) Name() string        { return "get" }
func (c *GetCommand) Aliases() []string  { return []string{"view", "show"} }
func (c *GetCommand) Description() string { return "특정 이슈의 상세 정보를 조회합니다." }

func (c *GetCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	if len(cleanArgs) < 1 {
		return fmt.Errorf("사용법: jira get <KEY> [-o table|json|md] [--json] [--md] (예: jira get KAN-1 --md)")
	}

	key := strings.ToUpper(cleanArgs[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	issue, err := client.GetIssue(ctx, key)
	if err != nil {
		return fmt.Errorf("이슈(%s) 조회 실패: %w", key, err)
	}

	switch format {
	case "json":
		return pkg.PrintIssueJSON(stdout, issue)
	case "md":
		pkg.PrintIssueMarkdown(stdout, issue)
		return nil
	default:
		pkg.PrintIssueDetail(stdout, issue)
		return nil
	}
}

// CreateCommand creates a new Jira issue.
type CreateCommand struct {
	clientProvider ClientProvider
	catalogLoader  CatalogLoader
}

func NewCreateCommand(cp ClientProvider, cl CatalogLoader) *CreateCommand {
	return &CreateCommand{clientProvider: cp, catalogLoader: cl}
}

func (c *CreateCommand) Name() string        { return "create" }
func (c *CreateCommand) Aliases() []string  { return []string{"new", "add"} }
func (c *CreateCommand) Description() string { return "신규 이슈를 생성합니다." }

func (c *CreateCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("사용법: jira create <SUMMARY> [-d description] [-t type] [-l labels] [--due YYYY-MM-DD]")
	}

	summary := args[0]

	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var desc, issueType, due, project, parent, labelsStr string
	var forceLabels bool

	fs.StringVar(&desc, "d", "", "이슈 상세 설명")
	fs.StringVar(&desc, "desc", "", "이슈 상세 설명")
	fs.StringVar(&issueType, "t", "작업", "이슈 유형 (작업/스토리/에픽/Subtask)")
	fs.StringVar(&issueType, "type", "작업", "이슈 유형")
	fs.StringVar(&labelsStr, "l", "", "라벨 목록 (쉼표 구분)")
	fs.StringVar(&labelsStr, "labels", "", "라벨 목록 (쉼표 구분)")
	fs.StringVar(&due, "due", "", "마감일 (YYYY-MM-DD)")
	fs.StringVar(&project, "p", "", "프로젝트 키")
	fs.StringVar(&project, "project", "", "프로젝트 키")
	fs.StringVar(&parent, "parent", "", "상위 이슈 키 (Subtask인 경우)")
	fs.BoolVar(&forceLabels, "force-labels", false, "표준 카탈로그 외 임의 라벨 허용")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	var rawLabels []string
	if labelsStr != "" {
		for _, l := range strings.Split(labelsStr, ",") {
			trimmed := strings.TrimSpace(l)
			if trimmed != "" {
				rawLabels = append(rawLabels, trimmed)
			}
		}
	}

	// Validate & Normalize labels against catalog
	labels := rawLabels
	if len(rawLabels) > 0 && !forceLabels {
		catalog, _ := c.catalogLoader()
		if catalog != nil {
			normalized, invalid, err := catalog.ValidateAndNormalizeLabels(rawLabels)
			if err != nil {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("❌ 라벨 유효성 오류: %v\n", err))
				sb.WriteString(fmt.Sprintf("   정의되지 않은 라벨: %s\n\n", strings.Join(invalid, ", ")))
				sb.WriteString("💡 표준 라벨 목록은 'jira labels' 명령어로 확인하세요.\n")
				sb.WriteString("   (표준 외 라벨을 강제 등록하려면 --force-labels 옵션을 추가하세요)")
				return fmt.Errorf("%s", sb.String())
			}
			labels = normalized
		}
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	resp, err := client.CreateIssue(ctx, project, summary, desc, issueType, due, parent, labels)
	if err != nil {
		return fmt.Errorf("이슈 생성 실패: %w", err)
	}

	fmt.Fprintf(stdout, "✅ Jira 이슈 생성 완료! [Key: %s] (%s/browse/%s)\n", resp.Key, client.GetConfig().InstanceURL, resp.Key)
	if len(labels) > 0 {
		fmt.Fprintf(stdout, "   🏷️ 적용된 표준 라벨: %s\n", strings.Join(labels, ", "))
	}

	return nil
}

// EditCommand updates fields on an existing Jira issue.
type EditCommand struct {
	clientProvider ClientProvider
	catalogLoader  CatalogLoader
}

func NewEditCommand(cp ClientProvider, cl CatalogLoader) *EditCommand {
	return &EditCommand{clientProvider: cp, catalogLoader: cl}
}

func (c *EditCommand) Name() string        { return "edit" }
func (c *EditCommand) Aliases() []string  { return []string{"update"} }
func (c *EditCommand) Description() string { return "기존 이슈 정보를 수정합니다." }

func (c *EditCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("사용법: jira edit <KEY> [-l <LABELS>] [--due <YYYY-MM-DD>] [--parent <PARENT_KEY>] [-s <SUMMARY>] [-d <DESC>] [--force-labels]")
	}

	key := strings.ToUpper(args[0])
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var labelsStr, dueStr, parentStr, summaryStr, descStr string
	var forceLabels bool
	fs.StringVar(&labelsStr, "l", "", "라벨 목록 (쉼표 구분)")
	fs.StringVar(&labelsStr, "labels", "", "라벨 목록 (쉼표 구분)")
	fs.StringVar(&dueStr, "due", "", "마감일 (YYYY-MM-DD 또는 none)")
	fs.StringVar(&parentStr, "parent", "", "상위 이슈/에픽 키 (또는 none)")
	fs.StringVar(&summaryStr, "s", "", "이슈 요약/제목")
	fs.StringVar(&summaryStr, "summary", "", "이슈 요약/제목")
	fs.StringVar(&descStr, "d", "", "이슈 상세 설명")
	fs.StringVar(&descStr, "desc", "", "이슈 상세 설명")
	fs.StringVar(&descStr, "description", "", "이슈 상세 설명")
	fs.BoolVar(&forceLabels, "force-labels", false, "표준 카탈로그 외 임의 라벨 허용")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	isSet := func(names ...string) bool {
		found := false
		fs.Visit(func(f *flag.Flag) {
			for _, name := range names {
				if f.Name == name {
					found = true
				}
			}
		})
		return found
	}

	if !isSet("l", "labels", "due", "parent", "s", "summary", "d", "desc", "description") {
		return fmt.Errorf("수정할 항목을 하나 이상 지정하세요. (예: jira edit KAN-9 --due 2026-09-01 -l 'FDE,AI' --parent KAN-17)")
	}

	var opts pkg.UpdateIssueOptions
	var updatedItems []string

	if isSet("l", "labels") {
		var rawLabels []string
		if labelsStr != "" {
			for _, l := range strings.Split(labelsStr, ",") {
				trimmed := strings.TrimSpace(l)
				if trimmed != "" {
					rawLabels = append(rawLabels, trimmed)
				}
			}
		}
		labels := rawLabels
		if len(rawLabels) > 0 && !forceLabels {
			catalog, _ := c.catalogLoader()
			if catalog != nil {
				normalized, invalid, err := catalog.ValidateAndNormalizeLabels(rawLabels)
				if err != nil {
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("❌ 라벨 유효성 오류: %v\n", err))
					sb.WriteString(fmt.Sprintf("   정의되지 않은 라벨: %s\n\n", strings.Join(invalid, ", ")))
					sb.WriteString("💡 표준 라벨 목록은 'jira labels' 명령어로 확인하세요.\n")
					sb.WriteString("   (표준 외 라벨을 강제 등록하려면 --force-labels 옵션을 추가하세요)")
					return fmt.Errorf("%s", sb.String())
				}
				labels = normalized
			}
		}
		opts.Labels = labels
		opts.UpdateLabels = true
		if len(labels) == 0 {
			updatedItems = append(updatedItems, "라벨: (제거)")
		} else {
			updatedItems = append(updatedItems, fmt.Sprintf("라벨: [%s]", strings.Join(labels, ", ")))
		}
	}

	if isSet("due") {
		opts.DueDate = &dueStr
		if dueStr == "" || strings.ToLower(dueStr) == "none" || strings.ToLower(dueStr) == "null" {
			updatedItems = append(updatedItems, "마감일: (제거)")
		} else {
			updatedItems = append(updatedItems, fmt.Sprintf("마감일: %s", dueStr))
		}
	}

	if isSet("parent") {
		pUpper := strings.ToUpper(strings.TrimSpace(parentStr))
		opts.ParentKey = &pUpper
		if pUpper == "" || strings.ToLower(pUpper) == "none" || strings.ToLower(pUpper) == "null" {
			updatedItems = append(updatedItems, "상위 이슈: (제거)")
		} else {
			updatedItems = append(updatedItems, fmt.Sprintf("상위 이슈: %s", pUpper))
		}
	}

	if isSet("s", "summary") {
		opts.Summary = &summaryStr
		updatedItems = append(updatedItems, fmt.Sprintf("요약: %s", summaryStr))
	}

	if isSet("d", "desc", "description") {
		opts.Description = &descStr
		updatedItems = append(updatedItems, "설명: (수정됨)")
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	if err := client.UpdateIssue(ctx, key, opts); err != nil {
		return fmt.Errorf("이슈(%s) 수정 실패: %w", key, err)
	}

	fmt.Fprintf(stdout, "✅ [%s] 이슈가 성공적으로 업데이트되었습니다: %s\n", key, strings.Join(updatedItems, " | "))
	return nil
}

// MoveCommand transitions an issue to a new status.
type MoveCommand struct {
	clientProvider ClientProvider
}

func NewMoveCommand(cp ClientProvider) *MoveCommand {
	return &MoveCommand{clientProvider: cp}
}

func (c *MoveCommand) Name() string        { return "move" }
func (c *MoveCommand) Aliases() []string  { return []string{"transition", "status"} }
func (c *MoveCommand) Description() string { return "이슈의 상태를 변경(전이)합니다." }

func (c *MoveCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("사용법: jira move <KEY> <STATUS> (예: jira move KAN-1 '진행 중')")
	}

	key := strings.ToUpper(args[0])
	targetStatus := strings.Join(args[1:], " ")

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	if err := client.TransitionIssue(ctx, key, targetStatus); err != nil {
		return fmt.Errorf("상태 변경 실패: %w", err)
	}

	fmt.Fprintf(stdout, "✅ [%s] 이슈 상태가 '%s'로 성공적으로 변경되었습니다.\n", key, targetStatus)
	return nil
}

// CommentCommand adds a comment to an issue.
type CommentCommand struct {
	clientProvider ClientProvider
}

func NewCommentCommand(cp ClientProvider) *CommentCommand {
	return &CommentCommand{clientProvider: cp}
}

func (c *CommentCommand) Name() string        { return "comment" }
func (c *CommentCommand) Aliases() []string  { return []string{} }
func (c *CommentCommand) Description() string { return "이슈에 코멘트를 등록합니다." }

func (c *CommentCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("사용법: jira comment <KEY> <MESSAGE> (예: jira comment KAN-1 '코드 리뷰 완료')")
	}

	key := strings.ToUpper(args[0])
	message := strings.Join(args[1:], " ")

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	if err := client.AddComment(ctx, key, message); err != nil {
		return fmt.Errorf("코멘트 등록 실패: %w", err)
	}

	fmt.Fprintf(stdout, "✅ [%s] 코멘트가 등록되었습니다.\n", key)
	return nil
}

// TransitionsCommand lists possible workflow transitions for an issue.
type TransitionsCommand struct {
	clientProvider ClientProvider
}

func NewTransitionsCommand(cp ClientProvider) *TransitionsCommand {
	return &TransitionsCommand{clientProvider: cp}
}

func (c *TransitionsCommand) Name() string        { return "transitions" }
func (c *TransitionsCommand) Aliases() []string  { return []string{} }
func (c *TransitionsCommand) Description() string { return "전환 가능한 워크플로우 상태 목록을 조회합니다." }

func (c *TransitionsCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("사용법: jira transitions <KEY> (예: jira transitions KAN-1)")
	}

	key := strings.ToUpper(args[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	transitions, err := client.GetTransitions(ctx, key)
	if err != nil {
		return fmt.Errorf("전환 가능 목록 조회 실패: %w", err)
	}

	fmt.Fprintf(stdout, "📋 [%s] 전환 가능한 상태 목록:\n", key)
	for _, t := range transitions {
		fmt.Fprintf(stdout, "  • ID: %-5s ➔ 이름: %-15s (목적지: %s)\n", t.ID, t.Name, t.To.Name)
	}
	return nil
}

// DeleteCommand deletes an issue from Jira.
type DeleteCommand struct {
	clientProvider ClientProvider
}

func NewDeleteCommand(cp ClientProvider) *DeleteCommand {
	return &DeleteCommand{clientProvider: cp}
}

func (c *DeleteCommand) Name() string        { return "delete" }
func (c *DeleteCommand) Aliases() []string  { return []string{"rm"} }
func (c *DeleteCommand) Description() string { return "이슈를 삭제합니다." }

func (c *DeleteCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("사용법: jira delete <KEY> (예: jira delete KAN-1)")
	}

	key := strings.ToUpper(args[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("Jira 클라이언트 초기화 오류: %w", err)
	}

	if err := client.DeleteIssue(ctx, key, true); err != nil {
		return fmt.Errorf("이슈(%s) 삭제 실패: %w", key, err)
	}

	fmt.Fprintf(stdout, "🗑️  [%s] 이슈가 성공적으로 삭제되었습니다.\n", key)
	return nil
}
