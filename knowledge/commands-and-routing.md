---
title: "CLI Commands & Routing Engine"
type: api
description: "CLI 서브커맨드 라우팅, 플래그 파싱 및 머신 리더블(--md, --json) 출력 서식 엔진"
tags:
  - cli
  - routing
  - flags
  - formatting
  - markdown
  - json
status: stable
timestamp: 2026-09-20T18:00:00+09:00
sources:
  - internal/app/commands_issue.go
  - internal/app/commands_meta.go
  - pkg/format.go
  - pkg/format_test.go
verified: true
---

# 💻 CLI 명령어 라우팅 및 출력 포맷 엔진

본 문서는 CLI 서브커맨드 라우팅 메커니즘, 플래그 파싱 규칙 및 AI Agent와 터미널 환경을 위한 출력 포맷팅 체계를 설명합니다.

---

## 🧭 1. 명령어 및 별칭 명세 (Command Dispatcher)

모든 명령어는 [internal/app/app.go](file:///mnt/data/myjob/cloit/jira-cli/internal/app/app.go)에 등록되어 대소문자 무관 및 별칭(Aliases)으로 매핑됩니다.

| 기본 명령어 | 별칭 (Aliases) | 소스 위치 | 주요 기능 |
| :--- | :--- | :--- | :--- |
| `list` | `ls` | `commands_issue.go` | JQL 검색 및 이슈 목록 조회 (기본: 최근순) |
| `get` | `view`, `show` | `commands_issue.go` | 이슈 메타데이터, 설명, 하위작업, 코멘트 전체 맥락 조회 |
| `create` | `new`, `add` | `commands_issue.go` | 신규 티켓(작업, 스토리, 에픽, Subtask, Bug) 생성 |
| `edit` | `update` | `commands_issue.go` | 기존 티켓(제목, 설명, 라벨, 마감일, 상위키) 수정 |
| `move` | `transition`, `status` | `commands_issue.go` | 워크플로 상태 전이 (예: '해야 할 일' → '진행 중' → '완료') |
| `comment` | - | `commands_issue.go` | 티켓에 코멘트 추가 (ADF 변환 지원) |
| `transitions` | - | `commands_issue.go` | 해당 티켓에서 전이 가능한 워크플로 상태 ID 및 이름 조회 |
| `delete` | `rm` | `commands_issue.go` | 티켓 삭제 (단독 또는 하위작업 포함) |
| `labels` | `tags` | `commands_meta.go` | 프로젝트 표준 라벨 카탈로그 조회 |
| `configure` | `config` | `commands_config.go` | 대화형 프로필 생성/수정 및 목록 조회 |

---

## 🖨️ 2. 머신 리더블 출력 서식 엔진 (`pkg/format.go`)

AI Agent(LLM)와 자동화 스크립트 연동을 위해 세 가지 출력 모드를 지원합니다:

### 1) Markdown 출력 (`--md`, `-o md`)
- **이슈 목록 (`PrintIssuesMarkdown`)**:
  - GFM 파이프(`|`) 테이블 포맷으로 출력합니다.
  - 제목 내 파이프 기호는 `\|`로 자동 이스케이프 처리됩니다.
  - **요약문(Summary) 자르기 없이 원문 전체를 출력**하여 데이터 유실 및 UTF-8 멀티바이트 깨짐을 방지합니다.
- **이슈 상세 (`PrintIssueMarkdown`)**:
  - LLM 프롬프트에 바로 주입할 수 있는 단일 완성형 마크다운 문서로 서식화됩니다:
    - 상단 메타데이터 불릿 리스트
    - `## 📄 상세 설명 (Description)`: ADF 본문 텍스트
    - `## 🔹 하위 작업 (Subtasks)`: `- [ ]` 또는 `- [x]` 체크박스 리스트
    - `## 💬 코멘트 (Comments)`: `> Blockquote` 형태의 작성자 및 일시별 코멘트 블록

### 2) JSON 출력 (`--json`, `-o json`)
- **함수 (`PrintIssuesJSON`, `PrintIssueJSON`)**:
  - Go의 `json.Encoder`를 통해 들여쓰기 2칸(`"  "`)의 유효한 JSON으로 출력합니다.
  - LLM Tool Use / Function Calling에서 스키마 손실 없이 파싱할 때 기본 권장 포맷입니다.

### 3) 터미널 테이블 출력 (기본값)
- `text/tabwriter`를 이용한 아스키 기반 표 형식입니다.

---

## 🛡️ 3. 공통 플래그 추출 규칙 (`parseOutputFormat`)

[internal/app/commands_issue.go](file:///mnt/data/myjob/cloit/jira-cli/internal/app/commands_issue.go)의 `parseOutputFormat` 함수는 인자 리스트에서 포맷 플래그(`--json`, `--md`, `-o <fmt>`, `-o=<fmt>`)를 순서에 상관없이 감지하고 제거하여, 나머지 인자(JQL이나 이슈 키)와 충돌하지 않도록 보장합니다:

```bash
# 아래 호출들은 모두 동일하게 동작합니다
jira list "status = '진행 중'" --md
jira list --md "status = '진행 중'"
jira get KAN-10 --json
jira get --json KAN-10
```
