package app

import (
	"context"
	"fmt"
	"io"

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
func (c *HelpCommand) Description() string { return "CLI 도움말 및 사용법을 출력합니다." }

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
func (c *VersionCommand) Description() string { return "CLI 버전을 출력합니다." }

func (c *VersionCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fmt.Fprintf(stdout, "jira version %s\n", c.version)
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
func (c *LabelsCommand) Description() string { return "표준 라벨 카탈로그를 조회합니다." }

func (c *LabelsCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	catalog, err := c.catalogLoader()
	if err != nil {
		return fmt.Errorf("라벨 카탈로그 로드 실패: %w", err)
	}
	pkg.PrintLabelsCatalog(stdout, catalog)
	return nil
}
