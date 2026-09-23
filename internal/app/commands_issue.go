package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"tools/jira/internal/i18n"
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
func (c *ListCommand) Description() string { return i18n.T("cmd.list.desc") }

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
			return "", nil, fmt.Errorf("%s", i18n.T("cmd.common.err_output_flag_missing_format"))
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
		return "", nil, fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_unsupported_output_format", format))
	}

	return format, cleanArgs, nil
}

func (c *ListCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.list.usage"))
		return nil
	}

	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	jql := ""
	if len(cleanArgs) > 0 {
		jql = strings.Join(cleanArgs, " ")
	}

	issues, err := client.ListIssues(ctx, jql)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.list.err_fetch", err))
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
func (c *GetCommand) Description() string { return i18n.T("cmd.get.desc") }

func (c *GetCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.get.usage"))
		return nil
	}

	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	if len(cleanArgs) < 1 {
		return fmt.Errorf("%s", i18n.T("cmd.get.err_missing_key"))
	}

	key := strings.ToUpper(cleanArgs[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	issue, err := client.GetIssue(ctx, key)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.get.err_fetch", key, err))
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
func (c *CreateCommand) Description() string { return i18n.T("cmd.create.desc") }

func (c *CreateCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var desc, issueType, due, project, parent, labelsStr string
	var forceLabels, allowEmptyDesc bool

	fs.StringVar(&desc, "d", "", i18n.T("cmd.create.flag_desc"))
	fs.StringVar(&desc, "desc", "", i18n.T("cmd.create.flag_desc"))
	fs.StringVar(&desc, "description", "", i18n.T("cmd.create.flag_desc"))
	fs.StringVar(&issueType, "t", "작업", i18n.T("cmd.create.flag_type"))
	fs.StringVar(&issueType, "type", "작업", i18n.T("cmd.create.flag_type"))
	fs.StringVar(&labelsStr, "l", "", i18n.T("cmd.create.flag_labels"))
	fs.StringVar(&labelsStr, "labels", "", i18n.T("cmd.create.flag_labels"))
	fs.StringVar(&due, "due", "", i18n.T("cmd.create.flag_due"))
	fs.StringVar(&project, "p", "", i18n.T("cmd.create.flag_project"))
	fs.StringVar(&project, "project", "", i18n.T("cmd.create.flag_project"))
	fs.StringVar(&parent, "parent", "", i18n.T("cmd.create.flag_parent"))
	fs.BoolVar(&forceLabels, "force-labels", false, i18n.T("cmd.create.flag_force_labels"))
	fs.BoolVar(&allowEmptyDesc, "allow-empty-desc", false, i18n.T("cmd.create.flag_allow_empty_desc"))

	fs.Usage = func() {
		fmt.Fprintln(stdout, i18n.T("cmd.create.usage"))
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	posArgs, err := parseFlagsAndPositional(fs, args)
	if err != nil {
		return err
	}

	if len(posArgs) < 1 {
		return fmt.Errorf("%s", i18n.T("cmd.create.err_summary_required"))
	}

	argSummary := posArgs[0]
	summary := strings.TrimSpace(argSummary.Value)
	if utf8.RuneCountInString(summary) < 5 {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_summary_min_length", summary))
	}
	if !argSummary.Literal && strings.HasPrefix(summary, "-") {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_summary_flag_prefix", summary))
	}

	if strings.TrimSpace(desc) == "" && !allowEmptyDesc {
		return fmt.Errorf("%s", i18n.T("cmd.create.err_desc_required"))
	}

	if strings.TrimSpace(due) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(due)); err != nil {
			return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_invalid_due_date", due))
		}
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
				return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_label_validation", err, strings.Join(invalid, ", ")))
			}
			labels = normalized
		}
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	resp, err := client.CreateIssue(ctx, project, summary, desc, issueType, due, parent, labels)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_api_create", err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.create.success", resp.Key, client.GetConfig().InstanceURL, resp.Key))
	if len(labels) > 0 {
		fmt.Fprintf(stdout, i18n.Sprintf("cmd.create.applied_labels", strings.Join(labels, ", ")))
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
func (c *EditCommand) Description() string { return i18n.T("cmd.edit.desc") }

func (c *EditCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var labelsStr, dueStr, parentStr, summaryStr, descStr string
	var forceLabels bool
	fs.StringVar(&labelsStr, "l", "", i18n.T("cmd.create.flag_labels"))
	fs.StringVar(&labelsStr, "labels", "", i18n.T("cmd.create.flag_labels"))
	fs.StringVar(&dueStr, "due", "", i18n.T("cmd.edit.flag_due"))
	fs.StringVar(&parentStr, "parent", "", i18n.T("cmd.edit.flag_parent"))
	fs.StringVar(&summaryStr, "s", "", i18n.T("cmd.edit.flag_summary"))
	fs.StringVar(&summaryStr, "summary", "", i18n.T("cmd.edit.flag_summary"))
	fs.StringVar(&descStr, "d", "", i18n.T("cmd.create.flag_desc"))
	fs.StringVar(&descStr, "desc", "", i18n.T("cmd.create.flag_desc"))
	fs.StringVar(&descStr, "description", "", i18n.T("cmd.create.flag_desc"))
	fs.BoolVar(&forceLabels, "force-labels", false, i18n.T("cmd.create.flag_force_labels"))

	fs.Usage = func() {
		fmt.Fprintln(stdout, i18n.T("cmd.edit.usage"))
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	posArgs, err := parseFlagsAndPositional(fs, args)
	if err != nil {
		return err
	}

	if len(posArgs) < 1 {
		return fmt.Errorf("%s", i18n.T("cmd.edit.err_missing_key"))
	}

	key := strings.ToUpper(strings.TrimSpace(posArgs[0].Value))

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
		return fmt.Errorf("%s", i18n.T("cmd.edit.err_no_fields_to_update"))
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
					return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_label_validation", err, strings.Join(invalid, ", ")))
				}
				labels = normalized
			}
		}
		opts.Labels = labels
		opts.UpdateLabels = true
		if len(labels) == 0 {
			updatedItems = append(updatedItems, i18n.T("cmd.edit.item_labels_cleared"))
		} else {
			updatedItems = append(updatedItems, i18n.Sprintf("cmd.edit.item_labels_updated", strings.Join(labels, ", ")))
		}
	}

	if isSet("due") {
		trimmedDue := strings.TrimSpace(dueStr)
		if trimmedDue == "" || strings.ToLower(trimmedDue) == "none" || strings.ToLower(trimmedDue) == "null" {
			opts.DueDate = &trimmedDue
			updatedItems = append(updatedItems, i18n.T("cmd.edit.item_due_cleared"))
		} else {
			if _, err := time.Parse("2006-01-02", trimmedDue); err != nil {
				return fmt.Errorf("%s", i18n.Sprintf("cmd.create.err_invalid_due_date", trimmedDue))
			}
			opts.DueDate = &trimmedDue
			updatedItems = append(updatedItems, i18n.Sprintf("cmd.edit.item_due_updated", trimmedDue))
		}
	}

	if isSet("parent") {
		pUpper := strings.ToUpper(strings.TrimSpace(parentStr))
		opts.ParentKey = &pUpper
		if pUpper == "" || strings.ToLower(pUpper) == "none" || strings.ToLower(pUpper) == "null" {
			updatedItems = append(updatedItems, i18n.T("cmd.edit.item_parent_cleared"))
		} else {
			updatedItems = append(updatedItems, i18n.Sprintf("cmd.edit.item_parent_updated", pUpper))
		}
	}

	if isSet("s", "summary") {
		opts.Summary = &summaryStr
		updatedItems = append(updatedItems, i18n.Sprintf("cmd.edit.item_summary_updated", summaryStr))
	}

	if isSet("d", "desc", "description") {
		opts.Description = &descStr
		updatedItems = append(updatedItems, i18n.T("cmd.edit.item_desc_updated"))
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if err := client.UpdateIssue(ctx, key, opts); err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.edit.err_api_update", key, err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.edit.success", key, strings.Join(updatedItems, " | ")))
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
func (c *MoveCommand) Description() string { return i18n.T("cmd.move.desc") }

func (c *MoveCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.move.usage"))
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("%s", i18n.T("cmd.move.err_missing_args"))
	}

	key := strings.ToUpper(args[0])
	targetStatus := strings.Join(args[1:], " ")

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if err := client.TransitionIssue(ctx, key, targetStatus); err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.move.err_transition", err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.move.success", key, targetStatus))
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
func (c *CommentCommand) Description() string { return i18n.T("cmd.comment.desc") }

func (c *CommentCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.comment.usage"))
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("%s", i18n.T("cmd.comment.err_missing_args"))
	}

	key := strings.ToUpper(args[0])
	message := strings.Join(args[1:], " ")

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if err := client.AddComment(ctx, key, message); err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.comment.err_api_comment", err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.comment.success", key))
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
func (c *TransitionsCommand) Description() string { return i18n.T("cmd.transitions.desc") }

func (c *TransitionsCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.transitions.usage"))
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("%s", i18n.T("cmd.transitions.err_missing_key"))
	}

	key := strings.ToUpper(args[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	transitions, err := client.GetTransitions(ctx, key)
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.transitions.err_fetch", err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.transitions.header", key))
	for _, t := range transitions {
		fmt.Fprintf(stdout, i18n.Sprintf("cmd.transitions.item", t.ID, t.Name, t.To.Name))
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
func (c *DeleteCommand) Description() string { return i18n.T("cmd.delete.desc") }

func (c *DeleteCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if isHelpRequested(args) {
		fmt.Fprintln(stdout, i18n.T("cmd.delete.usage"))
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("%s", i18n.T("cmd.delete.err_missing_key"))
	}

	key := strings.ToUpper(args[0])
	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	if err := client.DeleteIssue(ctx, key, true); err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.delete.err_api_delete", key, err))
	}

	fmt.Fprintf(stdout, i18n.Sprintf("cmd.delete.success", key))
	return nil
}
