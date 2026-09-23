package app

import (
	"context"
	"fmt"
	"io"

	"tools/jira/internal/i18n"
	"tools/jira/pkg"
)

// HelpCommand displays usage information.
type HelpCommand struct {
	app *App
}

func NewHelpCommand(app *App) *HelpCommand {
	return &HelpCommand{app: app}
}

func (c *HelpCommand) Name() string        { return "help" }
func (c *HelpCommand) Aliases() []string  { return []string{"-h", "--help"} }
func (c *HelpCommand) Description() string { return i18n.T("cmd.help.desc") }

func (c *HelpCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	c.app.PrintUsage(stdout)
	return nil
}

// VersionCommand displays the application version.
type VersionCommand struct {
	version string
}

func NewVersionCommand(version string) *VersionCommand {
	return &VersionCommand{version: version}
}

func (c *VersionCommand) Name() string        { return "version" }
func (c *VersionCommand) Aliases() []string  { return []string{"-v", "--version"} }
func (c *VersionCommand) Description() string { return i18n.T("cmd.version.desc") }

func (c *VersionCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fmt.Fprintf(stdout, i18n.Sprintf("cmd.version.output", c.version))
	return nil
}

// LabelsCommand displays the standard labels catalog.
type LabelsCommand struct {
	catalogLoader CatalogLoader
}

func NewLabelsCommand(loader CatalogLoader) *LabelsCommand {
	return &LabelsCommand{catalogLoader: loader}
}

func (c *LabelsCommand) Name() string        { return "labels" }
func (c *LabelsCommand) Aliases() []string  { return []string{"tags"} }
func (c *LabelsCommand) Description() string { return i18n.T("cmd.labels.desc") }

func (c *LabelsCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	catalog, err := c.catalogLoader()
	if err != nil {
		return fmt.Errorf("%s", i18n.Sprintf("cmd.labels.err_load", err))
	}
	pkg.PrintLabelsCatalog(stdout, catalog)
	return nil
}
