---
title: "Jira CLI Knowledge Log"
type: log
description: "Jira CLI Open Knowledge Format(OKF) 지식 베이스 생성 및 갱신 이력"
tags:
  - okf
  - log
  - changelog
status: stable
timestamp: 2026-09-20T18:00:00+09:00
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
- **검증 소스**:
  - `internal/i18n/i18n_test.go`, `internal/app/app_test.go`, `pkg/format_test.go` (전체 패키지 100% Pass)

