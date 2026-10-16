package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"tools/jira/internal/i18n"
	"tools/jira/pkg"
)

// UsersCommand searches and lists Jira users.
type UsersCommand struct {
	clientProvider ClientProvider
}

// NewUsersCommand creates a new UsersCommand.
func NewUsersCommand(cp ClientProvider) *UsersCommand {
	return &UsersCommand{clientProvider: cp}
}

func (c *UsersCommand) Name() string        { return "users" }
func (c *UsersCommand) Aliases() []string  { return []string{"user"} }
func (c *UsersCommand) Description() string { return i18n.T("cmd.users.desc") }

func (c *UsersCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("users", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var assignable, includeInactive bool
	var project string

	fs.BoolVar(&assignable, "assignable", false, i18n.T("cmd.users.flag_assignable"))
	fs.BoolVar(&includeInactive, "include-inactive", false, i18n.T("cmd.users.flag_include_inactive"))
	fs.StringVar(&project, "p", "", i18n.T("cmd.users.flag_project"))
	fs.StringVar(&project, "project", "", i18n.T("cmd.users.flag_project"))

	fs.Usage = func() {
		fmt.Fprintln(stdout, i18n.T("cmd.users.usage"))
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		fs.SetOutput(stderr)
	}

	if isHelpRequested(args) {
		fs.Usage()
		return nil
	}

	format, cleanArgs, err := parseOutputFormat(args)
	if err != nil {
		return err
	}

	posArgs, err := parseFlagsAndPositional(fs, cleanArgs)
	if err != nil {
		return err
	}

	var query string
	if len(posArgs) > 0 {
		var parts []string
		for _, pa := range posArgs {
			parts = append(parts, pa.Value)
		}
		query = strings.Join(parts, " ")
	}

	client, err := c.clientProvider()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.common.err_client_init", err))
	}

	var users []pkg.User
	if assignable {
		users, err = client.SearchAssignableUsers(ctx, project, query)
	} else {
		users, err = client.SearchGlobalUsers(ctx, query, includeInactive)
	}
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.users.err_fetch", err))
	}

	switch format {
	case "json":
		return pkg.PrintUsersJSON(stdout, users)
	case "md":
		pkg.PrintUsersMarkdown(stdout, users)
		return nil
	default:
		pkg.PrintUsersTable(stdout, users)
		return nil
	}
}
