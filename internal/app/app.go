package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"tools/jira/pkg"
)

const DefaultVersion = "1.5.0"

// ClientProvider is a factory function for obtaining a Jira Client.
type ClientProvider func() (*pkg.Client, error)

// CatalogLoader is a function for loading the label catalog.
type CatalogLoader func() (*pkg.LabelCatalog, error)

// App is the CLI application router and runner.
type App struct {
	Version        string
	commands       map[string]Command
	commandList    []Command
	clientProvider ClientProvider
	catalogLoader  CatalogLoader
}

// NewApp creates a new App with custom providers.
func NewApp(version string, clientProvider ClientProvider, catalogLoader CatalogLoader) *App {
	if version == "" {
		version = DefaultVersion
	}
	if clientProvider == nil {
		clientProvider = pkg.NewClient
	}
	if catalogLoader == nil {
		catalogLoader = pkg.LoadLabelCatalog
	}

	app := &App{
		Version:        version,
		commands:       make(map[string]Command),
		clientProvider: clientProvider,
		catalogLoader:  catalogLoader,
	}

	app.registerBuiltinCommands()
	return app
}

// NewDefaultApp creates an App with standard configuration and default commands.
func NewDefaultApp() *App {
	return NewApp(DefaultVersion, pkg.NewClient, pkg.LoadLabelCatalog)
}

// Register registers a command and its aliases into the App.
func (a *App) Register(cmd Command) {
	a.commandList = append(a.commandList, cmd)
	a.commands[cmd.Name()] = cmd
	for _, alias := range cmd.Aliases() {
		a.commands[alias] = cmd
	}
}

func (a *App) registerBuiltinCommands() {
	// Meta / Information commands
	a.Register(NewHelpCommand(a))
	a.Register(NewVersionCommand(a.Version))
	a.Register(NewLabelsCommand(a.catalogLoader))
	a.Register(NewConfigureCommand(nil))

	// Jira Issue management commands
	a.Register(NewListCommand(a.clientProvider))
	a.Register(NewGetCommand(a.clientProvider))
	a.Register(NewCreateCommand(a.clientProvider, a.catalogLoader))
	a.Register(NewEditCommand(a.clientProvider, a.catalogLoader))
	a.Register(NewMoveCommand(a.clientProvider))
	a.Register(NewCommentCommand(a.clientProvider))
	a.Register(NewTransitionsCommand(a.clientProvider))
	a.Register(NewDeleteCommand(a.clientProvider))
}

// PrintUsage outputs the usage message to the specified writer.
func (a *App) PrintUsage(w io.Writer) {
	fmt.Fprintf(w, `Jira CLI - Personal & Team Task Management Tool (v%s)

사용법:
  jira <command> [arguments]

명령어 목록:
  configure [--global]           대화형 Jira 설정 마법사 (.jira.json 생성, 별칭: configuration, config)
  list [JQL]                     이슈 목록 조회 (기본: KAN 프로젝트 생성일 역순)
  get <KEY>                      특정 이슈의 상세 정보 및 코멘트 조회
  create <SUMMARY> [flags]       신규 이슈(작업/스토리/에픽/버그) 생성
  edit <KEY> [flags]             기존 이슈 정보(라벨, 마감일, 제목, 설명) 수정
  move <KEY> <STATUS>            이슈 상태 전이 (예: '해야 할 일', '진행 중', '완료')
  comment <KEY> <MESSAGE>        이슈에 코멘트 등록
  transitions <KEY>              해당 이슈에서 변경 가능한 상태 목록 조회
  delete <KEY>                   이슈 삭제
  labels                         표준 라벨 카탈로그 조회
  version                        CLI 버전 확인

create / edit 플래그 옵션:
  -d, --desc string              이슈 상세 설명
  -s, --summary string           이슈 요약/제목 (edit 전용)
  -t, --type string              이슈 유형 (기본: '작업', 선택: 스토리, 에픽, Subtask, Bug)
  -l, --labels string            라벨/태그 (쉼표로 구분, 예: 'DIVE,AI,Planning')
  --due string                   마감일 (예: 2026-09-05 또는 none)
  -p, --project string           프로젝트 키 (기본: .env의 JIRA_PROJECT_KEY 또는 KAN)
  --parent string                상위 이슈 키 (Subtask 생성 시 필수)
  --force-labels                 표준 카탈로그 외 임의 라벨 허용

예시:
  jira list
  jira labels
  jira create "DIVE MCP 플랫폼 구축" -d "상세 설명" -l "DIVE,AI,Planning" --due 2026-09-05
  jira edit KAN-9 --due 2026-09-01
  jira edit KAN-9 -l "FDE,AI,Planning,Report" --due 2026-09-01
  jira get KAN-1
  jira move KAN-1 "진행 중"
`, a.Version)
}

// Run executes the application logic with arguments and I/O streams.
// Returns an integer exit code (0 for success, 1 for error).
func (a *App) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		a.PrintUsage(stdout)
		return 0
	}

	commandName := args[0]
	cmd, exists := a.commands[commandName]
	if !exists {
		fmt.Fprintf(stderr, "알 수 없는 명령어: %s\n", commandName)
		a.PrintUsage(stderr)
		return 1
	}

	if err := cmd.Execute(ctx, args[1:], stdout, stderr); err != nil {
		if !strings.HasPrefix(err.Error(), "❌") && !strings.HasPrefix(err.Error(), "사용법:") {
			fmt.Fprintf(stderr, "❌ %s\n", err)
		} else {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}

	return 0
}
