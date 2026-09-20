---
title: "System Architecture & Design Principles"
type: architecture
description: "Go 표준 라이브러리 기반 Zero-Dependency 원칙, 3-Tier 계층 아키텍처 및 의존성 주입 설계"
tags:
  - architecture
  - zero-dependency
  - dependency-injection
  - layered-design
status: stable
timestamp: 2026-09-20T18:00:00+09:00
sources:
  - cmd/main.go
  - internal/app/app.go
  - internal/app/command.go
verified: true
---

# 🏛️ 시스템 아키텍처 및 설계 원칙

Jira CLI는 개발자와 AI 에이전트(LLM) 환경에서 즉각적으로 구동될 수 있도록 **초경량, 고성능, 무의존성(Zero-Dependency)**을 핵심 원칙으로 설계되었습니다.

---

## 🎯 1. 핵심 설계 원칙 (Core Principles)

1. **Zero Third-Party Dependencies (공급망 보안 및 빌드 속도)**:
   - `go.mod`에 외부 서드파티 라이브러리를 일절 포함하지 않습니다.
   - HTTP 통신(`net/http`), JSON 처리(`encoding/json`), 플래그 파싱(`flag`), 터미널 서식(`text/tabwriter`) 등 모든 기능을 Go 표준 라이브러리만으로 구현합니다.
2. **IoC (Inversion of Control) 및 테스트 용이성**:
   - `internal/app` 계층은 구체적인 네트워크 클라이언트에 직접 의존하지 않고, `ClientProvider` 및 `CatalogLoader` 함수 팩토리를 통해 의존성을 주입받습니다.
   - `httptest.Server` 기반의 E2E Mock 테스트를 실제 네트워크 없이 10ms 이내에 수행할 수 있습니다.
3. **명확한 I/O 스트림 및 종료 코드 (AI Agent Friendly)**:
   - 정상 출력은 `stdout`, 에러/진단 메시지는 `stderr`로 엄격히 분리합니다.
   - 에이전트 파이프라인에서 성공(`0`)과 실패(`1`)를 명확히 판별할 수 있도록 결정론적 Exit Code를 반환합니다.

---

## 🏗️ 2. 3-Tier 계층 구조 (Layered Architecture)

```mermaid
flowchart TD
    OS[OS / Terminal / AI Agent] -->|Args & Context| Main["cmd/main.go<br>(Entrypoint & Signal Handler)"]
    Main -->|Execute Run()| App["internal/app/app.go<br>(App Router & Command Dispatcher)"]
    
    subgraph Commands ["internal/app/ (CLI Presentation Layer)"]
        App --> ListCmd["commands_issue.go (ListCommand)"]
        App --> GetCmd["commands_issue.go (GetCommand)"]
        App --> CreateCmd["commands_issue.go (CreateCommand)"]
        App --> EditCmd["commands_issue.go (EditCommand)"]
        App --> MoveCmd["commands_issue.go (MoveCommand)"]
        App --> ConfigCmd["commands_config.go (ConfigureCommand)"]
        App --> MetaCmd["commands_meta.go (Help/Labels/Version)"]
    end
    
    subgraph Core ["pkg/ (Domain & Infrastructure Layer)"]
        ListCmd & GetCmd & CreateCmd --> Client["pkg/client.go (Jira REST API v3 Client)"]
        CreateCmd & EditCmd --> Labels["pkg/labels.go (Labels Catalog & Validator)"]
        ListCmd & GetCmd --> Format["pkg/format.go (Markdown & JSON Formatter)"]
        Client --> Types["pkg/types.go (ADF Engine & Data Models)"]
    end

    Client -->|HTTPS Basic Auth| JiraAPI["Atlassian Jira Cloud REST API v3"]
```

### 계층별 역할 정의
- **`cmd/`**: 애플리케이션 진입점. `os.Interrupt`, `SIGTERM` 시그널을 수신하여 `context.WithCancel`을 트리거하고 `app.Run()`의 종료 코드로 프로세스를 종료합니다.
- **`internal/app/`**: 외부 패키지에서 임포트할 수 없는 내부 비즈니스 라우터. `Command` 인터페이스를 기반으로 명령어를 등록하고, 플래그 파싱 및 출력 스트림(`stdout`, `stderr`)을 제어합니다.
- **`pkg/`**: 재사용 가능한 핵심 라이브러리. Atlassian REST API v3 통신, ADF 마크다운 변환 엔진, INI 프로필 파서, 터미널/마크다운 테이블 포맷터를 포함합니다.

---

## 🧩 3. Command 인터페이스 및 등록 패턴

모든 CLI 명령어는 [internal/app/command.go](file:///mnt/data/myjob/cloit/jira-cli/internal/app/command.go)에 정의된 인터페이스를 구현합니다:

```go
type Command interface {
    Name() string
    Aliases() []string
    Description() string
    Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error
}
```

- 신규 명령어를 추가할 때는 `Command` 인터페이스를 만족하는 구조체를 작성하고 [internal/app/app.go](file:///mnt/data/myjob/cloit/jira-cli/internal/app/app.go)의 `registerBuiltinCommands()`에 `a.Register(NewYourCommand(...))`로 등록하기만 하면 라우팅, 도움말 및 별칭 처리가 자동화됩니다.
