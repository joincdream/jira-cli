# 📐 01. 시스템 아키텍처 및 설계 원칙 (Architecture & Design Principles)

이 문서는 `jira` CLI 도구의 전체적인 소프트웨어 구조, 설계 원칙, 데이터 흐름 및 컴포넌트 간 상호작용을 설명합니다. AI 에이전트(LLM)와 개발자는 본 문서를 통해 전체 시스템의 일관성을 유지하며 기능을 확장할 수 있습니다.

---

## 🎯 1. 핵심 설계 철학 (Core Philosophy)

1. **Zero External Dependencies (무의존성 원칙)**:
   - 외부 서드파티 라이브러리(예: spf13/cobra, urfave/cli 등)를 일체 사용하지 않고, 오직 Go 표준 라이브러리(`net/http`, `encoding/json`, `flag`, `text/tabwriter`, `context`)만으로 모든 기능을 구현합니다.
   - 단일 정적 바이너리로 즉시 배포 가능하며, Go 버전 호환성 문제와 공급망(Supply Chain) 보안 취약점을 원천 차단합니다.
2. **Command Pattern & Clean Router**:
   - 모든 명령어는 `app.Command` 인터페이스를 구현하며, `App` 인스턴스에 플러그인 형태로 등록됩니다.
   - 비즈니스 로직과 CLI 라우팅 계층이 명확히 분리되어 있어 신규 명령어 추가 및 수정이 직관적입니다.
3. **의존성 주입(DI)을 통한 테스트 용이성 (Testability)**:
   - 네트워크 통신(`ClientProvider`), 라벨 카탈로그 파일 로딩(`CatalogLoader`), 표준 I/O(`stdout`, `stderr`, `stdin`)가 인터페이스 및 함수 타입으로 추상화되어 있어, 단위 테스트 시 실제 API 호출 없이 100% Mocking이 가능합니다.
4. **AI Agent 친화적 I/O (Machine-Readable & Predictable)**:
   - 명확한 Exit Code (`0`: 성공, `1`: 실패)를 반환합니다.
   - 사용자/에이전트가 처리하기 쉬운 명확한 터미널 테이블 및 상태 배지 형식을 제공합니다.

---

## 🏗️ 2. 패키지 구조 및 책임 (Package Responsibilities)

```text
tools/jira/
├── cmd/
│   └── main.go                 # 어플리케이션 진입점 (OS Signal 제어, 프로세스 종료)
├── internal/
│   └── app/
│       ├── app.go              # App 구조체, CLI 라우팅, 커맨드 디스패칭
│       ├── command.go          # Command 인터페이스 규격
│       ├── commands_issue.go   # Jira Issue CRUD 명령군 (list, get, create, edit, move, comment, transitions, delete)
│       ├── commands_config.go  # 설정 마법사 명령군 (configure)
│       ├── commands_meta.go    # 메타 명령군 (help, version, labels)
│       └── app_test.go         # App 라우터 및 플래그 파싱 단위 테스트
└── pkg/
    ├── client.go               # Jira REST API v3 HTTP 클라이언트 및 계층적 설정 로더
    ├── client_test.go          # HTTP 클라이언트 및 요청 직렬화 테스트
    ├── format.go               # 터미널 테이블/상세 정보 텍스트 포맷터
    ├── format_test.go          # 출력 포맷팅 단위 테스트
    ├── labels.go               # 라벨 카탈로그 파싱, 유효성 검사, 대소문자 정규화
    ├── labels_test.go          # 라벨 카탈로그 로딩 및 검증 테스트
    ├── types.go                # Jira API DTO 모델 및 Markdown <-> ADF 변환 엔진
    └── types_test.go           # ADF 변환 양방향 단위 테스트
```

### 계층별 세부 역할

| 패키지 | 역할 및 설계 의도 |
| :--- | :--- |
| `cmd/main.go` | `context.WithCancel` 및 `signal.NotifyContext`를 통해 SIGINT, SIGTERM 시 안전한 Graceful Shutdown 지원. `app.NewDefaultApp().Run()` 호출 후 `os.Exit(exitCode)` 처리. |
| `internal/app` | 애플리케이션의 CLI 런타임 영역. 명령어 파싱, 플래그 검증, 에러 메시지 포맷팅, 표준 입출력 제어를 전담. |
| `pkg/client.go` | Jira Cloud REST API v3 (`/rest/api/3/...`)와의 직접적인 HTTP 통신. Basic Auth 인증, 엔드포인트 URL 조합, 공통 HTTP 요청 처리(`doRequest`), 에러 응답 파싱(`formatJiraAPIError`). |
| `pkg/types.go` | Jira의 복잡한 JSON 응답 구조체 및 ADF(Atlassian Document Format)를 트리 형태로 파싱/생성하는 내장 엔진. |
| `pkg/labels.go` | 표준 라벨 카탈로그 파일(`labels.json` / `.agents/labels.json`)을 상위 디렉토리로 순회 탐색하여 로딩하고, 입력 라벨을 검증/정규화. |
| `pkg/format.go` | `tabwriter.NewWriter` 기반의 고정폭 정렬 테이블 출력 및 상세 본문 렌더링. |

---

## 🔄 3. 전체 데이터 흐름 (Data Flow)

```mermaid
flowchart TD
    User([User or AI Agent]) -->|CLI Arguments & Flags| Main["cmd/main.go"]
    Main -->|context, args, stdout, stderr| App["internal/app.App"]
    
    App -->|Command Lookup| CmdRouter{"Command Exists?"}
    CmdRouter -->|No| Help["PrintUsage() -> exit(1)"]
    
    CmdRouter -->|Yes| Exec["Command.Execute()"]
    
    subgraph Execution ["Command Execution Flow"]
        Exec -->|Load Credentials| ConfigLoader["pkg.LoadConfig()"]
        ConfigLoader -->|Priority Scan| CredSources["OS Env / .jira.json / ~/.config/jira / .env"]
        
        Exec -->|Validate Labels| LabelValidator["pkg.LoadLabelCatalog()"]
        
        Exec -->|Convert Markdown to ADF| ADFEngine["pkg.BuildADFDocument()"]
        
        Exec -->|Invoke HTTP Request| JiraClient["pkg.Client (REST API v3)"]
    end
    
    JiraClient -->|Basic Auth HTTPS| JiraAPI[("Atlassian Jira Cloud REST API")]
    JiraAPI -->|JSON / ADF Response| JiraClient
    
    JiraClient -->|Raw DTO| Formatter["pkg.Formatters (Table / Detail)"]
    Formatter -->|Formatted Text| Stdout([stdout: Clean Terminal Output])
    
    Execution -.->|Error Encountered| Stderr([stderr: ❌ Error Message -> exit(1)])
```

---

## 🧩 4. 의존성 주입(DI) 및 확장 인터페이스

### `Command` 인터페이스 (`internal/app/command.go`)
모든 CLI 서브커맨드는 아래 인터페이스를 반드시 구현합니다.
```go
type Command interface {
    Name() string
    Aliases() []string
    Description() string
    Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error
}
```

### Provider 함수 주입 (`internal/app/app.go`)
```go
type ClientProvider func() (*pkg.Client, error)
type CatalogLoader func() (*pkg.LabelCatalog, error)
```
- 실제 프로덕션 환경에서는 `pkg.NewClient`, `pkg.LoadLabelCatalog`를 주입합니다.
- 테스트 환경에서는 Mock 인스턴스를 반환하는 클로저를 주입하여 네트워크나 디스크 I/O 없이 모든 로직을 검증합니다.
