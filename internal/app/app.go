package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"tools/jira/internal/i18n"
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

// PrintUsage outputs the usage message to the specified writer based on active language.
func (a *App) PrintUsage(w io.Writer) {
	if i18n.CurrentLanguage() == "ko" {
		fmt.Fprintf(w, `Jira CLI - 개인 및 팀 업무 관리 도구 (v%s)

개요 (Overview for AI Agent & Users):
  Jira CLI는 터미널 및 AI Agent 환경에서 Atlassian Jira 작업을 수행하기 위한 경량 CLI 도구입니다.
  AI Agent(LLM)가 SKILL 또는 자동화 도구로 연동할 때 표준 출력(stdout/stderr), 종료 코드(0/1),
  그리고 머신 리더블 포맷(--json, --md)을 완벽히 지원합니다.

사용법 (Usage):
  jira [--profile <name>] [--lang <en|ko>] <command> [arguments] [flags]

명령어 목록 (Commands):
  list [JQL] [flags]             이슈 목록 조회 (기본: 최신순, 별칭: ls)
  get <KEY> [flags]              특정 이슈 상세, 설명, 하위작업, 코멘트 전체 맥락 조회 (별칭: view, show)
  create <SUMMARY> [flags]       신규 이슈(작업/스토리/에픽/버그/Subtask) 생성 (별칭: new, add)
  edit <KEY> [flags]             기존 이슈 정보(라벨, 마감일, 제목, 설명, 상위이슈) 수정 (별칭: update)
  move <KEY> <STATUS>            이슈 상태 전이 (예: '해야 할 일', '진행 중', '완료', 별칭: transition, status)
  comment <KEY> <MESSAGE>        이슈에 코멘트 등록
  transitions <KEY>              해당 이슈에서 전이 가능한 워크플로 상태 목록 조회
  delete <KEY>                   이슈 삭제 (별칭: rm)
  labels                         표준 라벨 카탈로그 및 가이드 조회 (별칭: tags)
  configure [--profile <name>]   대화형 Jira 프로필 설정 (~/.config/jira/config, 별칭: config)
  configure list                 등록된 Jira 프로필 목록 조회
  version                        CLI 버전 확인 (별칭: -v, --version)
  help                           도움말 출력 (별칭: -h, --help)

전역 옵션 (Global Options):
  --profile string               사용할 계정 프로필 (기본: default, 환경변수: JIRA_PROFILE)
  --lang string                  UI 언어 설정 (en, ko, 기본: en, 환경변수: JIRA_LANG)

출력 포맷 플래그 (Output Flags for list, get):
  -o, --output string            출력 형식 지정 (table [기본], json, md)
  --json                         JSON 구조체로 출력 (AI Agent 도구 호출 및 Function calling용)
  --md, --markdown               Markdown 문서/테이블로 출력 (LLM 프롬프트 주입 및 맥락 파악용)

생성 및 수정 플래그 (create / edit Flags):
  -d, --desc string              이슈 상세 설명 (Markdown 텍스트 지원, 자동 ADF 변환)
  -s, --summary string           이슈 요약/제목 (edit 전용)
  -t, --type string              이슈 유형 (기본: '작업', 선택: 작업, 스토리, 에픽, Subtask, Bug)
  -l, --labels string            라벨 목록 (쉼표 구분, 예: 'AI,Planning,Development')
  --due string                   마감일 (형식: YYYY-MM-DD, 해제: none)
  -p, --project string           프로젝트 키 (기본: 설정 파일의 project_key)
  --parent string                상위 이슈 키 (Subtask 생성 시 필수, 또는 에픽 연결)
  --force-labels                 표준 카탈로그 외 임의 라벨 강제 허용
  --allow-empty-desc             본문 설명 없이 티켓 생성 허용

종료 코드 (Exit Codes):
  0: 성공 (정상 수행 완료, stdout에 결과 반환)
  1: 실패 (인자 오류, API 인증 실패, 네트워크 오류 등, stderr에 에러 메시지 반환)

AI Agent / SKILL 개발 가이드 (Best Practices for AI Agents):
  1. 티켓 목록 조회 시:
     - 스크립트/도구 파싱용: 'jira list --json' 또는 'jira list "<JQL>" --json'
     - 사용자 브리핑/마크다운용: 'jira list --md'
  2. 티켓 전체 맥락 파악 시:
     - 'jira get <KEY> --md' 실행 시 설명(Description), 하위작업(Subtasks), 코멘트(Comments)가
       단일 마크다운 문서로 출력되어 별도 추가 질의 없이 전체 문맥을 즉시 파악 가능합니다.
  3. 티켓 생성 전:
     - 'jira labels'로 팀 표준 라벨을 사전에 확인하여 오타 및 비표준 라벨 방지 권장
  4. 상태 변경 전:
     - 'jira transitions <KEY>'로 전이 가능한 정확한 상태 이름을 확인 후 'jira move' 호출 권장

실행 예시 (Examples):
  jira list --md
  jira list "labels = AI and status = '진행 중'" --json
  jira get KAN-10 --md
  jira create "신규 기능 개발" -d "상세 요구사항" -l "AI,Development" --due 2026-10-15
  jira create "하위 작업" -t Subtask --parent KAN-10
  jira edit KAN-10 --due 2026-10-20 -l "AI,QA"
  jira transitions KAN-10
  jira move KAN-10 "진행 중"
  jira comment KAN-10 "검토 완료했습니다."
`, a.Version)
		return
	}

	fmt.Fprintf(w, `Jira CLI - Personal & Team Task Management Tool (v%s)

Overview (for AI Agents & Users):
  Jira CLI is a lightweight tool for interacting with Atlassian Jira in terminal and AI Agent environments.
  Fully supports stdout/stderr, standard exit codes (0/1), and machine-readable formats (--json, --md)
  for seamless integration with AI Agents (LLMs) via SKILLs or automation tools.

Usage:
  jira [--profile <name>] [--lang <en|ko>] <command> [arguments] [flags]

Commands:
  list [JQL] [flags]             List Jira issues (default: newest first, alias: ls)
  get <KEY> [flags]              View issue details, description, subtasks, and comments (alias: view, show)
  create <SUMMARY> [flags]       Create a new issue (Task/Story/Epic/Bug/Subtask, alias: new, add)
  edit <KEY> [flags]             Update existing issue fields (labels, due, summary, desc, parent, alias: update)
  move <KEY> <STATUS>            Transition issue status (e.g. 'In Progress', 'Done', alias: transition, status)
  comment <KEY> <MESSAGE>        Add a comment to an issue
  transitions <KEY>              List available workflow transitions for an issue
  delete <KEY>                   Delete an issue permanently (alias: rm)
  labels                         View standard labels catalog and guide (alias: tags)
  configure [--profile <name>]   Interactive configuration wizard (~/.config/jira/config, alias: config)
  configure list                 List configured Jira profiles
  version                        Display CLI version (alias: -v, --version)
  help                           Show help and usage information (alias: -h, --help)

Global Options:
  --profile string               Account profile to use (default: default, env: JIRA_PROFILE)
  --lang string                  Display language (en, ko, default: en, env: JIRA_LANG)

Output Format Flags (for list, get):
  -o, --output string            Specify output format (table [default], json, md)
  --json                         Output formatted JSON (for AI Agent tool calling)
  --md, --markdown               Output Markdown document/table (for LLM context injection)

Create and Edit Flags (create / edit):
  -d, --desc string              Issue description (Markdown supported, converted to ADF)
  -s, --summary string           Issue summary/title (edit only)
  -t, --type string              Issue type (default: 'Task', choices: Task, Story, Epic, Subtask, Bug)
  -l, --labels string            Labels list (comma-separated, e.g. 'AI,Development')
  --due string                   Due date (format: YYYY-MM-DD, clear: none)
  -p, --project string           Project key (default: config project_key)
  --parent string                Parent issue key (required for Subtask, or Epic linking)
  --force-labels                 Allow non-standard labels outside catalog
  --allow-empty-desc             Allow creating issue without description

Exit Codes:
  0: Success (output returned to stdout)
  1: Failure (argument error, API authentication error, network failure, etc., error returned to stderr)

AI Agent / SKILL Best Practices:
  1. For listing issues:
     - For scripts/tools: 'jira list --json' or 'jira list "<JQL>" --json'
     - For markdown summaries: 'jira list --md'
  2. For full issue context:
     - 'jira get <KEY> --md' outputs description, subtasks, and comments in a single markdown document.
  3. Before creating an issue:
     - Run 'jira labels' to inspect standard team labels and avoid typos.
  4. Before changing status:
     - Run 'jira transitions <KEY>' to verify exact available status names before calling 'jira move'.

Examples:
  jira list --md
  jira list "labels = AI and status = 'In Progress'" --json
  jira get KAN-10 --md
  jira create "Feature development" -d "Detailed specs" -l "AI,Development" --due 2026-10-15
  jira create "Subtask item" -t Subtask --parent KAN-10
  jira edit KAN-10 --due 2026-10-20 -l "AI,QA"
  jira transitions KAN-10
  jira move KAN-10 "In Progress"
  jira comment KAN-10 "Review completed."
`, a.Version)
}

// extractGlobalFlags extracts and removes --profile and --lang flags from args.
func extractGlobalFlags(args []string) (profile string, lang string, cleanArgs []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--profile" {
			if i+1 < len(args) {
				profile = args[i+1]
				i++
				continue
			}
		} else if strings.HasPrefix(arg, "--profile=") {
			profile = strings.TrimPrefix(arg, "--profile=")
			continue
		} else if arg == "--lang" {
			if i+1 < len(args) {
				lang = args[i+1]
				i++
				continue
			}
		} else if strings.HasPrefix(arg, "--lang=") {
			lang = strings.TrimPrefix(arg, "--lang=")
			continue
		}
		cleanArgs = append(cleanArgs, arg)
	}
	return profile, lang, cleanArgs
}

// extractProfileFlag extracts and removes --profile from args (compatibility wrapper).
func extractProfileFlag(args []string) (string, []string) {
	p, _, clean := extractGlobalFlags(args)
	return p, clean
}

// Run executes the application logic with arguments and I/O streams.
// Returns an integer exit code (0 for success, 1 for error).
func (a *App) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	profile, lang, cleanArgs := extractGlobalFlags(args)
	if profile != "" {
		pkg.SetActiveProfile(profile)
	}

	profileLang := ""
	if cfgPath, err := pkg.GetDefaultConfigPath(); err == nil {
		if pCfg, found, _ := pkg.ReadProfileConfig(cfgPath, pkg.GetActiveProfile()); found {
			profileLang = pCfg.Language
		}
	}
	i18n.Init(lang, profileLang)

	if len(cleanArgs) == 0 {
		a.PrintUsage(stdout)
		return 0
	}

	commandName := cleanArgs[0]
	cmd, exists := a.commands[commandName]
	if !exists {
		fmt.Fprintf(stderr, i18n.Sprintf("app.err_unknown_command", commandName))
		a.PrintUsage(stderr)
		return 1
	}

	if err := cmd.Execute(ctx, cleanArgs[1:], stdout, stderr); err != nil {
		errMsg := err.Error()
		if !strings.HasPrefix(errMsg, "❌") && !strings.HasPrefix(errMsg, "사용법:") && !strings.HasPrefix(errMsg, "Usage:") {
			fmt.Fprintf(stderr, "❌ %s\n", errMsg)
		} else {
			fmt.Fprintln(stderr, errMsg)
		}
		return 1
	}

	return 0
}
