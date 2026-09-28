---
title: "Jira CLI Knowledge Log"
type: log
description: "Jira CLI Open Knowledge Format(OKF) 지식 베이스 생성 및 갱신 이력"
tags:
  - okf
  - log
  - changelog
status: stable
timestamp: 2026-09-28T10:05:00+09:00
verified: true
---

# 📜 Jira CLI Knowledge Log

본 문서는 `knowledge/` 번들 내 지식 자산의 생성, 코드 변경에 따른 동기화 및 검증 이력을 기록합니다.

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


