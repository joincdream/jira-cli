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
timestamp: 2026-10-17T12:45:00+09:00
sources:
  - internal/app/commands_issue.go
  - internal/app/commands_user.go
  - internal/app/commands_open.go
  - internal/app/commands_agent.go
  - internal/app/commands_report.go
  - internal/app/commands_meta.go
  - internal/app/app.go
  - internal/templates/templates.go
  - internal/i18n/i18n.go
  - pkg/format.go
  - pkg/format_test.go
  - pkg/git.go
  - pkg/git_test.go
verified: true
---

# 💻 CLI 명령어 라우팅 및 출력 포맷 엔진

본 문서는 CLI 서브커맨드 라우팅 메커니즘, 플래그 파싱 규칙 및 AI Agent와 터미널 환경을 위한 출력 포맷팅 체계를 설명합니다.

---

## 🧭 1. 명령어 및 별칭 명세 (Command Dispatcher)

모든 명령어는 [internal/app/app.go](file:///mnt/data/myjob/cloit/jira-cli/internal/app/app.go)에 등록되어 대소문자 무관 및 별칭(Aliases)으로 매핑됩니다.

| 기본 명령어 | 별칭 (Aliases) | 소스 위치 | 주요 기능 |
| :--- | :--- | :--- | :--- |
| `list` | `ls` | `commands_issue.go` | JQL 검색 및 이슈 목록 조회 (기본: 최근순, `--mine`, `--detail` 지원) |
| `get` | `view`, `show` | `commands_issue.go` | 이슈 메타데이터, 설명, 하위작업, 코멘트 전체 맥락 조회 |
| `create` | `new`, `add` | `commands_issue.go` | 신규 티켓 생성 (`-a/--assignee` 지원) |
| `edit` | `update` | `commands_issue.go` | 기존 티켓 수정 (`-a/--assignee` 지원) |
| `assign` | - | `commands_issue.go` | 티켓 담당자 할당 및 해제 (`me`, `none`, 이메일, 이름) |
| `users` | - | `commands_user.go` | Jira 사용자 검색 (기본 인스턴스 전역, `--assignable` 프로젝트 한정) |
| `move` | `transition`, `status` | `commands_issue.go` | 워크플로 상태 전이 (예: '해야 할 일' → '진행 중' → '완료') |
| `comment` | - | `commands_issue.go` | 티켓에 코멘트 추가 (ADF 변환 지원) |
| `transitions` | - | `commands_issue.go` | 해당 티켓에서 전이 가능한 워크플로 상태 ID 및 이름 조회 |
| `delete` | `rm` | `commands_issue.go` | 티켓 삭제 (단독 또는 하위작업 포함) |
| `open` | `browse` | `commands_open.go` | 티켓 상세 페이지 또는 프로젝트 보드를 브라우저에서 열기 |
| `tui` | `cockpit`, `dashboard` | `commands_tui.go` | Bubble Tea 기반 실시간 대시보드 및 터미널 콕핏 화면 실행 |
| `watch` | `tail` | `commands_tui.go` | 실시간 이슈 변경 이벤트 스트리밍 출력 (`--json` 지원) |
| `use` | `set-active`, `current` | `commands_tui.go` | 로컬 캐시에 전역 활성 이슈 키 설정/조회 (`--clear` 지원) |
| `cache` | - | `commands_tui.go` | 로컬 SQLite 캐시 데이터베이스 관리 (`clean`, `status`) |
| `init-agent` | - | `commands_agent.go` | AI 에이전트(Cursor, Claude Code, Antigravity) 연동 룰 및 SKILL 초기화 |
| `report` | `digest`, `summary` | `commands_report.go` | 주간 업무 브리핑 및 프로젝트 진척 현황 요약 리포트 |
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

---

## ⚙️ 4. 인자 정규화 및 사전 검증 가드레일 (`internal/app/flags.go`)

### 1) 순서 독립성 (Order Independence) & `--help` 우선 처리
- 모든 서브커맨드는 `isHelpRequested`를 통해 `--help` 및 `-h` 플래그를 최우선 감지하여, API 요청을 발생시키지 않고 안전하게 도움말(Usage)을 `stdout`으로 출력하고 `Exit 0`으로 종료합니다.
- `parseFlagsAndPositional` 함수를 통해 위치 인자와 플래그의 배치 순서가 바뀌어도(예: `jira create -d "설명" "제목"`) 동일하게 파싱됩니다.
- Double Dash(`--`) 이후의 모든 토큰은 순수 위치 인자로 보호되어 `-`로 시작하는 제목도 안전하게 전달 가능합니다.

### 2) 티켓 생성(Create) 데이터 무결성 가드레일
- **제목(Summary) 검증**: 공백 제외 최소 5자 이상 필수(`utf8.RuneCountInString`), `--` 없이 `-`로 시작하는 플래그 오인 문자열 차단.
- **상세 설명(Description) 필수화**: 본문 없는 빈 티켓 양산을 방지하기 위해 기본 필수 요구 (설명 없는 간이 생성이 필요한 경우 `--allow-empty-desc` 명시 필요).
- **마감일(Due Date) 사전 검증**: Jira 서버 전송 전 로컬에서 `YYYY-MM-DD` 포맷 유효성 사전 검증 (Fail-Fast).

---

## 🌐 5. i18n 다국어 지원 및 로케일 해석 엔진 (`internal/i18n`)

CLI 전반의 모든 도움말, 에러 메시지, 터미널 포맷터 라벨은 다국어 지원(`ko`, `en`)을 제공합니다.

### 1) 5단계 로케일 결정 우선순위
```text
1. CLI 플래그:      --lang ko  (또는 --lang=en)
       ↓ (미지정 시)
2. 환경 변수:        JIRA_LANG=ko
       ↓ (미지정 시)
3. 프로필 설정:      ~/.config/jira/config 내 [profile] 섹션의 language = ko
       ↓ (미지정 시)
4. 시스템 OS 로케일:  $LC_ALL 또는 $LANG (예: ko_KR.UTF-8 ➔ ko)
       ↓ (미지정 시)
5. 기본값 (Fallback): "en"
```

### 2) 사용법 및 예시
```bash
# 강제 영문 모드로 실행
jira --lang en create "New Task" -d "Details"

# 강제 한국어 모드로 실행
jira --lang ko list --md

# 환경 변수로 기본 언어 고정
export JIRA_LANG=ko
```

---

## 🔍 6. 이슈 목록 범용 필터 플래그 (`internal/app/query_builder.go`)

Jira 관리자 및 실무자가 복잡한 JQL을 직접 작성하지 않고도 원하는 조건의 티켓을 빠르게 검색할 수 있도록 직교적(Orthogonal)인 범용 필터 플래그를 제공합니다:

| 플래그 | 형태 | 허용 값 예시 | 변환되는 JQL 표현식 |
| :--- | :--- | :--- | :--- |
| **`--all`, `-A`** | bool | `true` / `false` | 기본 `project = ...` 제한을 해제하고 전체 테넌트 범위로 조회 |
| **`--due`** | string | `today`<br>`<=today`<br>`tomorrow`<br>`<=tomorrow`<br>`week`<br>`YYYY-MM-DD`<br>`<=YYYY-MM-DD` | `duedate = endOfDay()`<br>`duedate <= endOfDay()`<br>`duedate = endOfDay('1d')`<br>`duedate <= endOfDay('1d')`<br>`duedate <= endOfWeek()`<br>`duedate = "2026-10-15"`<br>`duedate <= "2026-10-15"` |
| **`--status`** | string | `open` (기본값)<br>`done`<br>`all` (또는 `*`)<br>특정 상태명 | `statusCategory != Done`<br>`statusCategory = Done`<br>*(상태 조건 생략)*<br>`status = "<value>"` |
| **`--assignee`, `-a`** | string | `me`<br>`none`<br>특정 사용자/이메일 | `assignee = currentUser()`<br>`assignee is EMPTY`<br>`assignee = "<value>"` |
| **`--detail`, `-d`** | bool | `true` / `false` | 이슈 본문 설명(`description`) 및 최신 코멘트(`comment`) 포함 (AI 1-Shot 브리핑 지원) |

### 사용 예시
```bash
# 전체 프로젝트를 통틀어 오늘까지 해야 할 일 (미완료, 연체 포함)
jira list --all --due '<=today'

# 전체 프로젝트에서 내게 할당된 열린 티켓 목록
jira list --all --assignee me

# 현재 프로젝트에서 이번 주 마감인 티켓
jira list --due week

# 머신 리더블 마크다운 포맷과 결합
jira list --all --due '<=today' --md

# AI 에이전트용 원샷 활성 작업 풀 일괄 조회 (설명 및 코멘트 포함)
jira list --assignee me --status open --detail --md
```

---

## 📊 7. 주간 업무 브리핑 및 프로젝트 진척 리포트 (`jira report`)

관리자(PM/PdM) 및 개발 리드가 전체 프로젝트 진척률, 지연/위험 티켓, 담당자별 워크로드를 한눈에 파악하고 회의록이나 협업 툴(Slack/Confluence)에 즉시 공유할 수 있도록 지원하는 집계 리포팅 엔진입니다:

* **실행 모드**:
  * 기본 뷰: 깔끔한 텍스트 카드 포맷 (이모지 최소화, 전문적 가독성 확보)
  * `--md`: 모든 섹션(종합 요약, 위험/지연 작업, 완료 성과, 진행 중 작업, 담당자 워크로드, 프로젝트별 현황)을 **구조화된 Markdown Table**로 출력
  * `--json`: AI Agent 및 파이프라인 연계용 정형 JSON 데이터
* **지원 플래그**:
  * `--all`, `-A`: 전사 모든 프로젝트 대상 집계
  * `-p`, `--project <KEY>`: 특정 프로젝트 지정
  * `-a`, `--assignee <USER>`: 특정 담당자 필터 (`me`, `none`, 또는 사용자명)
  * `--period <week|month|all>`: 집계 대상 기간 (기본값: `week`)

---

## 🌿 8. Git 브랜치 컨텍스트 자동 바인딩 (Zero-Argument Ticket Binding)

개발자가 특정 이슈 브랜치(예: `feature/JC-14-git-binding`, `fix/JC-6-transition-error`)에서 작업 중일 때, 티켓 키 인수를 생략해도 CLI가 현재 Git 브랜치에서 이슈 키를 자동으로 추출하여 실행하는 무마찰(Zero-Argument) 인터페이스입니다.

### 1) 동작 규칙 및 인수 3단계 우선순위 (3-Tier Resolution)
1. **1순위 (명시적 인자)**: 사용자가 명령어 인자로 티켓 키를 명시하면(예: `jira get JC-3`) 최우선 적용합니다.
2. **2순위 (Git 브랜치 자동 감지)**:
   - 현재 브랜치 이름(예: `feature/JC-17-tui`)에서 이슈 키(`JC-17`)를 자동 추출합니다.
3. **3순위 (로컬 캐시 활성 컨텍스트 Fallback)**:
   - `main` 브랜치나 비-Git 디렉토리 환경이라도 `jira use JC-17`로 설정된 활성 티켓 키(`active_context` 테이블)가 존재하면 이를 자동으로 바인딩합니다.
4. **키 생략 지원 명령어**:
   - `jira get`: 활성/브랜치 키의 상세 정보 조회 (`jira get --md`)
   - `jira transitions`: 활성/브랜치 키의 워크플로 전이 목록 조회
   - `jira move <STATUS>`: 첫 번째 인자가 이슈 키 형식이 아니면 바인딩된 키를 타깃으로 상태 전이
   - `jira comment <MSG>`: 첫 번째 인자가 이슈 키 형식이 아니면 바인딩된 키에 코멘트 등록
   - `jira assign <USER>`: 인자가 1개일 때 바인딩된 키를 대상으로 담당자 지정 (`jira assign me`)
   - `jira edit [flags]`: 키 생략 시 바인딩된 키의 필드 수정
   - `jira open`: 키 생략 시 바인딩된 키의 이슈 웹페이지 오픈
5. **가이드 에러 안내**:
   - 인자, 브랜치, 캐시 활성 컨텍스트 3단계 모두 키가 없을 경우 직관적인 액션 가이드 에러를 출력하고 Exit code 1을 반환합니다.

---

## 🖥️ 9. 실시간 대시보드 및 감시 엔진 (`jira tui`, `jira watch`)

### 1) Bubble Tea TUI 콕핏 (`jira tui`)
- Go TUI 업계 표준인 `bubbletea` + `lipgloss` 기반 터미널 전용 대시보드입니다.
- **주요 레이아웃**:
  - 좌측 패널 (42%): 활성 이슈 목록 (커서 이동, 상태별 배지, 제목 말줄임)
  - 우측 상단 (55%): 선택된 티켓 상세 카드 (키, 상태, 담당자, 설명 ADF 파싱 텍스트)
  - 우측 하단 (45%): 백그라운드 Diffing 엔진이 수집한 실시간 변경 피드 (Activity Stream)
  - 상단/하단: 연결 스페이스 배지, 실시간 동기화 상태 (`LIVE WATCHING`), 단축키 레퍼런스
- **단축키**: `↑/↓`, `j/k`(이동), `r`(즉시 동기화), `o`(브라우저 열기), `c`(활성 티켓 설정), `ESC`(브리핑 닫기), `q`(종료).
- **AltScreen 버퍼**: 종료 시 화면이 원래 터미널로 완전히 복원됩니다.

### 2) CLI 실시간 스트리머 (`jira watch`)
- TUI 없이 터미널 표준 출력으로 변경 이벤트를 tail -f 형태로 실시간 스트리밍합니다.
- `--json` 플래그 사용 시 AI Agent나 파이프라인에서 수신하기 쉬운 스트림 JSON 라인으로 출력합니다.
- 적응형 백오프(15s ➔ 30s ➔ 60s) 및 HTTP 429 Retry-After를 자체적으로 방어합니다.

---

## ⚡ 10. 로컬 공유 SQLite 캐시 엔진 (`~/.cache/jira/state.db`)

Pure Go SQLite(`modernc.org/sqlite`, CGO_ENABLED=0)를 활용하여 0ms 기동(Stale-While-Revalidate)과 오프라인 조회를 지원합니다:
- **`jira list --cached`**: 네트워크 호출 없이 로컬 SQLite DB에서 이슈 목록 즉시 반환.
- **`jira get --cached <KEY>`**: 로컬 캐시에 저장된 티켓 상세 정보 즉시 반환.
- **`jira cache status`**: 캐시 파일 경로, 파일 크기, 저장된 이슈 수, 실시간 이벤트 수, 활성 티켓 표시.
- **`jira cache clean`**: 캐시 DB 및 WAL 저널 파일을 안전하게 삭제하고 초기화.
- **장기 부재 브리핑 (Welcome Back Digest)**: 7일 이상 동기화가 중단된 후 재기동 시 부재 중 변경된 내 티켓과 신규 할당 작업을 요약 브리핑 카드로 출력.




