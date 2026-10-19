---
title: "Jira CLI Knowledge Log"
type: log
description: "Jira CLI Open Knowledge Format(OKF) 지식 베이스 생성 및 갱신 이력"
tags:
  - okf
  - log
  - changelog
status: stable
timestamp: 2026-10-16T20:53:00+09:00
verified: true
---

# 📜 Jira CLI Knowledge Log

본 문서는 `knowledge/` 번들 내 지식 자산의 생성, 코드 변경에 따른 동기화 및 검증 이력을 기록합니다.

## 2026-10-18 (JC-17: 백그라운드 Watch 완료 상태 전이 이벤트 감지 픽스)
- **작업 티켓**: [JC-17](https://joincdream.atlassian.net/browse/JC-17)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 감시 JQL 쿼리 확장 (`pkg/watch.go`):
     - `statusCategory != Done` 조건으로 인해 방금 완료된 티켓이 응답에서 제외되어 `STATUS_CHANGE` 이벤트를 발행하지 못하던 문제를 해결.
     - JQL을 `project = '%s' AND (statusCategory != Done OR updated >= -1d) ORDER BY updated DESC`로 수정하여 최근 완료된 티켓도 Diff 대상에 포함.
  2. 활성/완료 티켓 분리 처리:
     - `DiffIssues`로 `STATUS_CHANGE`(진행 중 ➔ 완료) 이벤트를 정상 발행한 후, 활성 캐시/메모리(`inMemoryIssues`, `issues` 테이블)에는 `statusCategory != Done`인 미완료 티켓만 보관하고 완료된 티켓은 `PruneDeletedIssues`로 자동 정리.
  3. 빈 키 슬라이스 Prune 버그 픽스 (`pkg/cache.go`):
     - 모든 티켓이 완료되어 활성 티켓이 0개가 되었을 때 `PruneDeletedIssues`가 조기 리턴하던 버그 수정.
  4. 테스트 검증:
     - `pkg/watch_test.go`: 완료 상태 전이 이벤트 발행 및 활성 메모리/캐시 프루닝 단위 테스트 추가 (100% Pass).

## 2026-10-18 (JC-17: TUI rivo/tview 2D 위젯 대시보드 엔진 전면 전환)
- **작업 티켓**: [JC-17](https://joincdream.atlassian.net/browse/JC-17)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. `rivo/tview` 및 `gdamore/tcell/v2` 기반 2D 위젯 아키텍처 전환:
     - 1차원 문자열 결합(Bubble Tea/Lip Gloss)의 근본적 한계(한글/이모지 폭 오차, 창 리사이즈 시 강제 줄바꿈 및 화면 상단 밀림)를 해소하기 위해 2차원 셀 캔버스 버퍼 엔진으로 전환.
     - `internal/tui/app.go`: `tview.Application` 메인 루프, `tview.Flex` 기반 반응형 듀얼 패널 레이아웃 조립.
     - `internal/tui/widgets.go`: 좌측 이슈 목록 `tview.Table` (헤더, 선택 하이라이트, `j/k` 이동), 우측 상단 `tview.TextView` (상세 카드, 동적 색상, 자동 스크롤), 우측 하단 `tview.TextView` (실시간 피드), 상단/하단 바 서식화.
     - `internal/tui/tui_test.go`: 단위 및 네비게이션 테스트 100% 통과.
     - 패널 크기 조절 및 전체화면(Zoom) 기능: `z` (상세 카드 전체화면 토글), `[`/`]` (좌우 분할 폭 5% 조절), `{`/`}` (상하 분할 높이 5% 조절) 단축키 지원.
  2. 동시성 및 스레드 세이프 UI 업데이트:
     - 백그라운드 `pkg.WatchEngine` 이벤트 채널과 `app.QueueUpdateDraw()` 연동으로 화면 깜빡임 없는 실시간 스트리밍 갱신.
  3. 검증:
     - `go test -v ./internal/tui ./pkg` 100% Pass.
     - `CGO_ENABLED=0` Linux 및 macOS 크로스 컴파일 성공.
     - `make install` 배포 완료.

## 2026-10-18 (JC-17: TUI Viewport 도입 및 테두리 깨짐 방지 픽스)
- **작업 티켓**: [JC-17](https://joincdream.atlassian.net/browse/JC-17)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. `github.com/charmbracelet/bubbles/viewport` 도입:
     - 티켓 상세(Detail) 및 실시간 활동 피드(Activity Feed)에 뷰포트 위젯 적용.
     - 긴 본문(Description) 및 개행문자가 포함되어도 고정된 가상 사각형 밖으로 넘치지 않도록 자동 클리핑(`overflow: hidden`) 및 스크롤(`ctrl+d`, `ctrl+u`, 마우스 휠) 지원.
  2. Lip Gloss 가로/세로 프레임 오차 해결:
     - Border(2) 및 Padding(2)을 고려한 내부 가용 폭(`innerWidth = width - 4`)으로 텍스트 및 선택 행 말줄임/패딩 기준 통일.
     - 좌우 패널 합산 너비(`leftWidth + rightWidth == width`) 및 우측 상하 패널 합산 높이(`detailHeight + activityHeight == height`) 엄밀 일치로 터미널 강제 줄바꿈(Auto-Wrap) 완전 방지.
  3. 개행문자 정제:
     - Jira ADF 본문 텍스트 내 CRLF(`\r\n`)를 LF(`\n`)로 정규화하여 캐리지 리턴 커서 덮어쓰기 방지.
  4. Makefile 인스톨 보강:
     - `install -m 755`를 적용하여 바이너리 실행 중에도 `text file busy` 에러 없이 교체 설치 가능하도록 개선.

## 2026-10-17 (JC-17: TUI Real-time Dashboard & Shared SQLite Cache Engine)
- **작업 티켓**: [JC-17](https://joincdream.atlassian.net/browse/JC-17)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. Pure Go SQLite 공유 로컬 캐시 엔진 구현 (`pkg/cache.go`, `pkg/cache_test.go`):
     - `modernc.org/sqlite` 드라이버 채택 (`CGO_ENABLED=0` 100% 순수 Go 호환).
     - `~/.cache/jira/state.db` 단일 통합 DB 내 멀티 스페이스 인덱싱 (`spaces`, `issues`, `activities`, `active_context`).
     - WAL 모드 및 5000ms busy_timeout 활성화로 고성능 읽기/쓰기 동시성 보장.
     - `UpsertIssues`, `GetIssues`, `LogActivity`, `SetActiveIssue`, `PruneDeletedIssues` CRUD 지원.
  2. SWR 증분 감시 및 In-Memory Diffing 엔진 구현 (`pkg/watch.go`, `pkg/watch_test.go`):
     - Stale-While-Revalidate 패턴: 기동 시 0ms 즉시 이전 로컬 스냅샷 메모리 로드.
     - 백그라운드 증분 폴링 및 In-Memory Diffing: `old.Status != new.Status` (상태 전이), `len(new.Comments) > len(old.Comments)` (코멘트 추가), `new.Assignee == me` (할당) 정밀 감지.
     - 적응형 백오프(15s ➔ 30s ➔ 60s) 및 HTTP 429 방어.
     - 장기 부재(7일 이상 휴가 등) 감지 시 Welcome Back 복귀 브리핑 카드 (`CatchupDigest`) 생성.
  3. Bubble Tea + Lip Gloss TUI 콕핏 구현 (`internal/tui/model.go`, `internal/tui/view.go`, `internal/tui/tui_test.go`):
     - Elm MUV 아키텍처, 더블 버퍼링, UTF-8 2칸 한글 줄맞춤 및 터미널 반응형 리사이즈 지원.
     - 좌측 활성 이슈 목록 (42%), 우측 상단 티켓 상세 (55%), 우측 하단 실시간 변경 피드 (45%).
     - 단축키: `↑/↓`, `j/k`(이동), `r`(즉시 동기화), `o`(브라우저 열기), `c`(활성 티켓 설정), `ESC`(브리핑 닫기), `q`(종료).
     - `tea.WithAltScreen()` 적용으로 종료 시 터미널 화면 완벽 원복.
  4. CLI 서브커맨드 연동 (`internal/app/commands_tui.go`, `internal/app/commands_issue.go`):
     - `jira tui`: TUI 콕핏 실행.
     - `jira watch`: 터미널 실시간 스트리밍 (`--json` 지원).
     - `jira use <KEY>`: 활성 티켓 컨텍스트 설정 (`main` 브랜치에서도 Zero-Argument 타깃 바인딩 3순위 Fallback 연계).
     - `jira list --cached`, `jira get --cached`: 오프라인 초고속 0ms 로컬 캐시 조회.
     - `jira cache status`, `jira cache clean`: 캐시 관리.
  5. 테스트 및 검증:
     - `go test -v ./...` 100% Pass (단위 테스트 및 통합 Mock 테스트).
     - `CGO_ENABLED=0 GOOS=linux go build ./...` 및 `CGO_ENABLED=0 GOOS=darwin go build ./...` 크로스 컴파일 성공.
     - `make install` 배포 완료.
- **검증 소스**:
  - `pkg/cache.go`, `pkg/watch.go`, `internal/tui/*`, `internal/app/commands_tui.go`, `internal/app/commands_issue.go`

---

## 2026-10-17 (JC-14: Git Branch Context Auto-Binding / Zero-Argument Ticket Binding)
- **작업 티켓**: [JC-14](https://joincdream.atlassian.net/browse/JC-14)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. Git 브랜치 탐색 및 티켓 키 자동 추출 엔진 구현 (`pkg/git.go`, `pkg/git_test.go`):
     - `GetCurrentBranch`: 1차 `git rev-parse --abbrev-ref HEAD` (1초 타임아웃), 2차 `.git/HEAD` 직접 파싱 (Zero Subprocess).
     - `ExtractIssueKey`: 브랜치명에서 대소문자 무관 Jira 키 추출 및 대문자 정규화.
     - `ResolveIssueKey`: 명시적 키 우선, 생략 시 현재 Git 브랜치에서 키 자동 감지.
  2. 주요 서브커맨드 Zero-Argument 바인딩 (`internal/app/commands_issue.go`, `internal/app/commands_open.go`):
     - `jira get`: 키 생략 시 브랜치 키 상세 정보 조회
     - `jira transitions`: 키 생략 시 브랜치 키 워크플로 전이 목록 조회
     - `jira move <STATUS>`: 첫 번째 인자가 키가 아니면 브랜치 키를 타깃으로 상태 전이
     - `jira comment <MSG>`: 첫 번째 인자가 키가 아니면 브랜치 키에 코멘트 등록
     - `jira assign <USER>`: 인자 1개 시 브랜치 키에 담당자 지정
     - `jira edit [flags]`: 키 생략 시 브랜치 키 수정
     - `jira open`: 키 생략 시 브랜치 키가 있으면 이슈 웹페이지 오픈, 없으면 프로젝트 보드 오픈
  3. 다국어 및 에러 가이드 강화 (`internal/i18n/locales/{ko,en}.json`):
     - `cmd.common.err_git_context_required`, `cmd.common.err_no_issue_key_in_branch` 추가.
  4. 테스트 검증:
     - `pkg/git_test.go`, `internal/app/commands_issue_test.go` 내 `TestJC14_GitBranchContextBinding` 작성.
     - `go test -v ./...` 100% Pass 및 `make install` 배포 완료.
- **검증 소스**:
  - `pkg/git.go`, `internal/app/commands_issue.go`, `internal/app/commands_open.go`, `knowledge/commands-and-routing.md`

---

## 2026-10-16 (Agent Task Lifecycle Protocol: Automatic In-Progress Transition)
- **작업 티켓**: [JC-14](https://joincdream.atlassian.net/browse/JC-14) / Agent Guidelines Refinement
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 에이전트 작업 수명주기(Lifecycle) 프로토콜 정립:
     - 기존의 '임의 마감 금지(Human Gatekeeper)' 가드레일이 지나치게 방어적으로 작용하여 작업 착수 시에도 상태 전이를 누락하던 문제 해결.
     - **작업 착수 시 (Task Inception)**: 사용자가 특정 티켓의 작업 착수나 계획 수립을 지시하면 즉시 해당 티켓을 **'진행 중(In Progress)'**으로 자동 전이(`jira move <KEY> "진행 중"`)하여 실시간 가시화 보장.
     - **작업 마감 시 (Human Gatekeeper)**: 자율 마감 금지 유지, 최종 마감은 인간의 명시적 지시 시에만 완료 처리.
  2. 에이전트 룰 및 SKILL 동기화:
     - `.agents/skills/jira/SKILL.md`
     - `internal/templates/skill.md`
     - `internal/templates/cursor.mdc`
     - `internal/templates/claude.md`
  3. 티켓 반영:
     - `JC-14` 티켓 상태를 '진행 중'으로 성공적으로 전이.
- **검증 소스**:
  - `.agents/skills/jira/SKILL.md`, `internal/templates/`

---

## 2026-10-16 (JC-16: User Search, Assignee Management & Filtering)
- **작업 티켓**: [JC-16](https://joincdream.atlassian.net/browse/JC-16)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 사용자 조회 및 담당자 지정/해제 기능 추가:
     - `jira assign <KEY> <USER>`: 이슈 담당자 할당 (`me`, `none`/`unassigned`, 이메일, 이름 매핑 지원). Jira Cloud REST API v3 GDPR 제약에 따라 `accountId` 기반 자동 변환.
     - `jira users [flags] [query]`: 인스턴스 전역 사용자 검색 (기본) 및 `--assignable` 플래그를 통한 프로젝트 단위 할당 가능 팀원 검색 지원.
     - `jira edit <KEY> --assignee <USER>` (`-a`) 및 `jira create ... --assignee <USER>` (`-a`) 담당자 지정 지원.
     - `jira list --mine`: `--assignee me` 축약 플래그 지원.
  2. Jira Cloud REST API v3 엔드포인트 연동 (`pkg/client.go`):
     - `GET /rest/api/3/myself`: 현재 로그인 사용자 조회.
     - `GET /rest/api/3/users/search`: 전역 사용자 검색.
     - `GET /rest/api/3/user/assignable/search`: 프로젝트 할당 가능 사용자 검색.
     - `PUT /rest/api/3/issue/{key}/assignee`: 담당자 변경 및 해제 (`accountId: null`).
     - `ResolveUserAccountID`: `me`, `none`, 프로젝트 할당 가능 팀원 우선 조회 ➔ 전역 사용자 폴백 및 단일 일치자 자동 식별.
  3. 포맷터 및 다국어 지원:
     - `pkg/format.go`: `PrintUsersTable`, `PrintUsersMarkdown`, `PrintUsersJSON` 구현.
     - `internal/i18n/locales/{ko,en}.json`: `cmd.assign.*`, `cmd.users.*`, `cmd.list.flag_mine`, `fmt.users.*` 추가.
  4. 테스트 및 실전 검증:
     - `pkg/client_test.go`, `pkg/format_test.go`, `internal/app/commands_user_test.go` 단위 테스트 작성 (`go test -v ./...` 100% Pass).
     - 프로젝트 내 모든 열린 티켓(JC-16, JC-15, JC-14, JC-7, JC-6)을 `joinc.dream@gmail.com`에 일괄 할당 완료.
- **검증 소스**:
  - `pkg/client.go`, `internal/app/commands_user.go`, `internal/app/commands_issue.go`, `pkg/format.go`, `task/JC-16.md`

---

## 2026-10-16 (JC-13 Extension: Orthogonal --detail Flag & 1-Shot Agent Intelligence)
- **작업 티켓**: [JC-13](https://joincdream.atlassian.net/browse/JC-13)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 직교적 `--detail` (`-d`) 플래그 구현:
     - `pkg/client.go`: `ListIssuesWithOptions` 추가, `--detail` 활성화 시 Jira REST API fields에 `description,comment` 추가.
     - `internal/app/commands_issue.go`: `ListCommand`에 `--detail`, `-d` 플래그 바인딩.
     - `pkg/format.go`: `PrintIssuesMarkdownDetailed`, `PrintIssuesTableDetailed` 구현으로 단 1회 호출로 N개 티켓의 상세 설명 및 최근 코멘트 일괄 출력.
  2. 에이전트 인터랙션 피로 및 N+1 호출 문제 해결:
     - 상황별 커맨드(`today` 등) 대신 유닉스 철학("작은 시스템 큰 구현")에 입각한 직교적 플래그로 완결.
     - 에이전트가 사전 탐색(`which`) 및 개별 티켓 `get` 반복 루프 없이 `jira list --assignee me --status open --detail --md` 단 1회 호출로 내 모든 활성 작업 풀 맥락 획득.
  3. 실무자 멘탈 모델 기반 우선순위 추론 계약 (`SKILL.md` & 내장 템플릿):
     - 단순 `--due today` 기계적 조회가 아닌, [In Progress 우선 ➔ 마감일/중요도 가중치 ➔ 미착수 작업 추천] 의사결정 휴리스틱 정립.
     - `internal/templates/{skill.md,cursor.mdc,claude.md}` 및 `.agents/skills/jira/SKILL.md` 동기화.
  4. 테스트 검증:
     - `pkg/format_test.go` 및 `internal/app/commands_issue_test.go`에 `--detail` 플래그 및 상세 포맷팅 테스트 추가 (`go test -v ./...` 100% Pass).
- **검증 소스**:
  - `pkg/format_test.go`, `internal/app/commands_issue_test.go`, `task/JC-13.md`

---

## 2026-10-15 (JC-13: AI CLI Environment Integration & Agent Insight Block Support)
- **작업 티켓**: [JC-13](https://joincdream.atlassian.net/browse/JC-13)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. `jira init-agent` 서브커맨드 구현 (`internal/app/commands_agent.go`):
     - AI 코딩 에이전트(Cursor, Claude Code, Antigravity) 연동 룰 및 SKILL 파일 자동 생성기.
     - 대화형(Interactive) 체크박스/다중 번호 선택 프롬프트 및 비대화형 `--type <cursor|claude|antigravity|all>` 플래그 지원.
     - 기존 파일 보호(중복 스킵) 및 `--force` 덮어쓰기 지원.
     - 워크스페이스 스킬 경로 규격(`.agents/skills/jira/SKILL.md`) 준수.
  2. 에이전트 템플릿 임베딩 번들링 (`internal/templates/`):
     - Go 표준 `embed.FS`를 활용하여 Zero-Dependency로 템플릿 내장 (`cursor.mdc`, `claude.md`, `skill.md`).
  3. `Agent Insight Block` 출력 계약 규격화:
     - 에이전트가 Jira 티켓을 분석 및 브리핑할 때 [현황 ➔ AI 추론 & 맥락 ➔ 추천 액션] 3단 레이아웃 표준화.
     - Human Gatekeeper 가드레일 확립: 에이전트의 성급한 자율 마감(`move Done`) 금지 및 사용자 승인 필수화.
  4. 다국어 지원 (`internal/i18n/locales/{en,ko}.json`):
     - `cmd.init_agent.*` 메시지 카탈로그 등록.
  5. 단위 테스트 및 E2E 실전 협업 시나리오 검증:
     - `internal/app/commands_agent_test.go` 단위 테스트 작성 및 100% Pass.
     - Antigravity 에이전트와의 실전 E2E 협업(조회 ➔ 맥락 브리핑 ➔ 코멘트 등록 ➔ 웹 브라우저 토스 ➔ Human Gatekeeper 승인) 검증 완료.
- **검증 소스**:
  - `internal/app/commands_agent_test.go`, `go test -v ./...` (100% Pass), `task/JC-13.md`

---

## 2026-10-15 (JC-3: Web Browser Handoff Feature - jira open / browse)
- **작업 티켓**: [JC-3](https://joincdream.atlassian.net/browse/JC-3)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. `jira open [KEY]` 및 별칭 `jira browse` 서브커맨드 구현 (`internal/app/commands_open.go`).
  2. Zero-Dependency 크로스 플랫폼 브라우저 런처 구현 (Linux: `xdg-open`, macOS: `open`, Windows: `cmd /c start`).
  3. 커스텀 브라우저 실행 지원: `--browser <cmd>` 플래그 및 `JIRA_BROWSER` / `BROWSER` 환경변수 연동.
  4. `--url-only` 플래그 및 위치 인자 순서 독립성(`internal/app/flags.go`) 지원.
  5. URL 라우팅 최적화:
     - 이슈 키 전달 시: `https://<domain>/browse/<KEY>`
     - 프로젝트 모드(키 미전달 시): Jira 표준 단축 리다이렉트 URL인 `https://<domain>/browse/<ProjectKey>`로 연결하여 사전 API 호출 없이 대표 활성 보드로 자동 핸드오프.
  6. 비-GUI/헤드리스 환경 실행 실패 시 콘솔에 대상 URL을 안내하는 Graceful Fallback 처리.
  7. 다국어(`internal/i18n/locales/{en,ko}.json`) 및 단위 테스트(`internal/app/commands_open_test.go`) 100% Pass 검증.
- **검증 소스**:
  - `internal/app/commands_open_test.go`, `go test -v ./...` (테스트 100% Pass)

---

## 2026-09-20 (Initial OKF Knowledge Bundle & v1.5.0 Update)
- **작성자**: Antigravity AI Assistant & joincdream
- **변경 사항**:
  1. Google Open Knowledge Format(OKF) 표준 규격(`SPEC.md`)에 맞춘 지식 베이스 번들 초기화.
  2. `pkg/format.go` 내 Summary 50자 슬라이싱 제거 및 한글 UTF-8 깨짐 결함 해결 반영.
  3. `jira list` 및 `jira get` 명령어에 머신 리더블 출력 옵션(`--md`, `--json`, `-o`) 추가 및 맥락 통합 뷰 문서화.
  4. 프로젝트 로컬 프로필 자동 탐색(`.jira-profile`) 엔진 추가 및 4단계 프로필 결정 우선순위 명세화.
  5. `jira help` 출력 및 `~/.config/jira/SKILL.md` 문서 최신화.
- **검증 소스**:
  - `pkg/format_test.go`, `pkg/client_test.go`, `internal/app/app_test.go` (테스트 100% Pass)

---

## 2026-09-23 (JC-8: CLI Parsing Normalization & Guardrails)
- **작업 티켓**: [JC-8](https://joincdream.atlassian.net/browse/JC-8)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. `jira create --help` 실행 시 티켓이 생성되던 심각한 플래그 파싱 결함 수정 (모든 서브커맨드에 선제적 `isHelpRequested` 도입).
  2. CLI 옵션과 위치 인자의 순서 독립성(Order Independence) 및 Double Dash(`--`) 지원 유틸리티(`internal/app/flags.go`) 구현.
  3. 티켓 생성 사전 검증 가드레일 추가: 제목 공백 제외 최소 5자 이상(`utf8.RuneCountInString`), 상세 설명(`-d`) 기본 필수화(`--allow-empty-desc` 예외 지원), 마감일(`--due`) 로컬 포맷 검증.
  4. `EditCommand` 내 순서 독립적 파싱 및 마감일 로컬 유효성 검증 적용.
  5. 단위 테스트(`internal/app/commands_issue_test.go`) TC-01 ~ TC-15 구현 및 100% Pass 검증.
- **검증 소스**:
  - `internal/app/commands_issue_test.go`, `internal/app/app_test.go` (테스트 100% Pass)

---

## 2026-09-23 (JC-4: i18n Multilingual Architecture & Localization)
- **작업 티켓**: [JC-4](https://joincdream.atlassian.net/browse/JC-4)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 전체 코드베이스 전수 조사 및 127개 메시지 카탈로그 추출 (`task/messages.json`, `internal/i18n/locales/{en,ko}.json`).
  2. 외부 의존성 없는 Zero-Dependency 초경량 i18n 패키지(`internal/i18n`) 구축:
     - Go 표준 `embed.FS` 기반 단일 바이너리 내장.
     - 5단계 로케일 결정 우선순위 구현 (`--lang` > `JIRA_LANG` > 프로필 `language` > OS `$LANG` > `en`).
     - 단 5줄의 직관적이고 안전한 `Sprintf` / `T` 템플릿 포맷터 및 자가 치유(Graceful Key Fallback) 구현.
  3. CLI 전반의 다국어 마이그레이션:
     - `internal/app/app.go`: `--lang` 글로벌 플래그 추출, 바이링궐 Usage 배너.
     - `internal/app/commands_issue.go`: 모든 커맨드 에러, 도움말, 가드레일 문구 다국어화.
     - `internal/app/commands_config.go`: 대화형 마법사 프롬프트, 프로필 목록 다국어화.
     - `internal/app/commands_meta.go`, `pkg/labels.go`, `pkg/format.go`: 테이블/마크다운 헤더 다국어화.
     - `pkg/client.go`: INI `language` 필드 파싱 및 직렬화 지원.
  4. 테스트 스위트 확장:
     - `internal/i18n/i18n_test.go`: 우선순위 해석, 번역 정확성, 키 누락 안전성, en/ko 127개 키 전수 일치성 검증.
     - `internal/app/app_test.go`: `--lang en`, `--lang=ko` 언어 전환 및 에러 메시지 검증.
---

## 2026-09-28 (JC-9: AWS CLI-Style Profile Inheritance & Default Fallback)
- **작업 티켓**: [JC-9](https://joincdream.atlassian.net/browse/JC-9)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. AWS CLI 스타일 프로필 상속 엔진 구현 (`pkg/client.go`):
     - `Config` 구조체에 `SourceProfile` 필드 추가 및 `ReadProfileConfig` 파싱 지원.
     - 암시적 상속: 서브 프로필에 URL/Email/Token 누락 시 `[default]` 프로필 자동 Fallback.
     - 명시적 상속: `source_profile = <base>` 키를 통한 임의의 테넌트 프로필 상속 지원.
     - 다단계 상속 체인 전파 및 자식 프로필 필드별 부분 오버라이드 지원.
     - 순환 참조(`A -> B -> A`) 감지 및 최대 깊이(5단계) 초과 차단 가드레일 탑재.
  2. 대화형 설정 마법사 및 프로필 목록 확장 (`internal/app/commands_config.go`):
     - 기존 프로필 존재 시 신규 프로필 생성 프롬프트에서 인증 정보 상속 여부 질의 및 URL/토큰 입력 생략 지원.
     - `jira configure list` 출력 시 상속 관계(`(inherits: <profile>, project: <key>)`) 명시.
  3. i18n 메시지 카탈로그 등록 (`internal/i18n/locales/{en,ko}.json`):
     - `cmd.configure.prompt_inherit`, `prompt_source_profile`, `skip_credentials_notice` 한/영 등록.
---

## 2026-09-28 (JC-10: Generalized Query Filter Flags for jira list)
- **작업 티켓**: [JC-10](https://joincdream.atlassian.net/browse/JC-10)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. JQL 쿼리 빌더 유틸리티 구축 (`internal/app/query_builder.go`):
     - 특정 시점 하드코딩 안티패턴을 배제하고 직교적(Orthogonal) 범용 필터 플래그 합성 엔진 구현.
     - `--all`, `-A`: 활성 프로필의 단일 `project_key` 제한을 해제하고 전체 테넌트 스코프로 확장.
     - `--due <expr>`: 상대적/절대적 마감일 표현식(`today`, `<=today`, `tomorrow`, `<=tomorrow`, `week`, `YYYY-MM-DD`, `<=YYYY-MM-DD`) 파서 및 로컬 유효성 검증.
     - `--status <status>`: 상태 필터 매핑 (`open` [기본값: statusCategory != Done], `done`, `all`, 특정 상태명).
     - `--assignee <user>`, `-a`: 담당자 필터 매핑 (`me`, `none`, 사용자 이메일/ID).
  2. `ListCommand` 플래그 연동 (`internal/app/commands_issue.go`):
     - `parseFlagsAndPositional`을 적용하여 옵션과 위치 인자의 순서 독립성 지원.
     - 수동 JQL 직접 입력 시 플래그 빌더보다 우선 적용하여 100% 하위 호환성 유지.
  3. i18n 메시지 카탈로그 등록 (`internal/i18n/locales/{en,ko}.json`):
     - `cmd.list.usage`, `cmd.list.flag_all`, `cmd.list.flag_due`, `cmd.list.flag_status`, `cmd.list.flag_assignee` 한/영 등록.
  4. 테스트 스위트 확장:
     - `internal/app/query_builder_test.go`: `TestBuildFilterJQL` 10종 서브테스트 100% Pass.
     - `internal/app/commands_issue_test.go`: `TestListCommand_FilterFlags` Mock 서버 E2E 100% Pass.
- **검증 소스**:
  - `internal/app/query_builder_test.go`, `internal/app/commands_issue_test.go`, `internal/i18n/i18n_test.go` (전체 패키지 100% Pass)

---

## 2026-09-28 (JC-11: Weekly Progress & Briefing Report Command)
- **작업 티켓**: [JC-11](https://joincdream.atlassian.net/browse/JC-11)
- **작성자**: Antigravity AI Assistant & 윤상배
- **변경 사항**:
  1. 주간 업무 브리핑 및 프로젝트 진척 리포팅 엔진 구현 (`internal/app/commands_report.go`):
     - `ReportCommand`: `--all`/`-A`, `-p`/`--project`, `-a`/`--assignee`, `--period week|month|all`, `--md`, `--json` 지원.
     - `buildReportJQL`: 기간(주간 `startOfWeek()`, 월간 `startOfMonth()`) 및 완료/열린 이슈를 결합하는 JQL 자동 합성.
     - `aggregateReport`: 총 이슈, 완료율(%), 건강도(`NORMAL`/`WARNING`/`CRITICAL`), Overdue(지연 일수 산출), Due Soon(3일 내 마감), 담당자별 워크로드 및 프로젝트별 진행률 바 산출.
  2. 가독성 개선 및 전 섹션 Markdown Table 포맷터:
     - 시각적 노이즈를 유발하는 이모지를 전면 배제하고 단정하고 전문적인 텍스트 배지 적용.
     - 종합 요약, 위험/지연 작업, 완료 성과, 진행 중 작업, 담당자 워크로드, 프로젝트별 현황 전 섹션을 구조화된 Markdown Table로 출력.
  3. 라우터 및 다국어 지원:
     - `internal/app/app.go`: `report` 명령어 등록 및 사용법 가이드 갱신.
     - `internal/i18n/locales/{en,ko}.json`: `cmd.report.*` 번역 등록.
  4. 테스트 스위트 확장:
     - `internal/app/commands_report_test.go`: JQL 빌더, 집계 로직, 터미널/Markdown/JSON 출력 단위 및 Mock E2E 100% Pass.
- **검증 소스**:
  - `internal/app/commands_report_test.go`, `make test` (단위 테스트 100% Pass 및 코드 복잡도 `gocyclo <= 40`, `gocognit <= 70` 준수)


