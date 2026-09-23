# 📋 [JC-4] Go 준표준 패키지(golang.org/x/text) 기반 i18n 다국어 시스템 구축 계획

- **티켓 키**: [JC-4](https://joincdream.atlassian.net/browse/JC-4)
- **티켓 요약**: `[Feature] Go 준표준 패키지(golang.org/x/text) 기반 i18n 다국어 시스템 구축`
- **담당자**: AI 에이전트 & 윤상배
- **상태**: 진행 중 (In Progress)
- **라벨**: `Architecture`, `Development`

---

## 💡 사용자 관점의 핵심 개선 효과 (What Changes for Users?)

기술적인 세부 사항을 몰라도, 본 작업이 완료되면 **사용자가 체감하는 변화 4가지**는 다음과 같습니다:

| 구분 | ❌ 현재의 불편함 (Before) | ✨ 개선 후 사용자 경험 (After) |
| :--- | :--- | :--- |
| **글로벌 개발자 지원** | 모든 메시지가 한국어로 하드코딩되어 있어, 해외 오픈소스 개발자나 외국인 동료는 에러 메시지와 도움말을 전혀 이해할 수 없음 | **기본 언어가 영문 표준(`en`)으로 제공**되어 전 세계 개발자가 즉시 사용 가능하며 글로벌 오픈소스 표준을 준수함 |
| **한국어 자동 유지** | 기본이 영어로 바뀌더라도 국내 사용자가 매번 영어로 봐야 한다면 불편함 | **내 OS 언어가 한국어(`LANG=ko_KR...`)이면 별도 설정 없이도 자동으로 친숙한 한국어**로 출력됨 |
| **언어 전환의 자유** | 터미널 언어를 바꿀 방법이 아예 없었음 | **CLI 플래그(`--lang ko`/`--lang en`)**, **환경 변수(`JIRA_LANG`)**, **프로필 설정(`language = ko`)**으로 언제든 자유롭게 전환 가능 |
| **향후 신규 기능 안정성** | 새 기능이 추가될 때마다 언어가 뒤죽박죽 섞이거나 나중에 다시 번역해야 함 | 중앙 다국어 사전(`internal/i18n`)에서 모든 문구를 통합 관리하여 **일관된 톤앤매너** 유지 |

---

## 1. 개요 및 배경 (Background & Objectives)

### 1.1 현재 문제점
1. **메시지 하드코딩**: 커맨드 설명, 도움말, 에러 메시지, 터미널 테이블 헤더(`Key`, `상태`, `요약` 등)가 한국어로 하드코딩되어 있어 글로벌 오픈소스 배포 불가.
2. **다국어 아키텍처 부재**: 이후 구현될 기능([JC-9] 프로필 상속, [JC-3] jira open, [JC-5] Git 연동 등)마다 한국어 하드코딩이 반복되어 향후 대규모 재작업(Double Work) 발생 위험.

### 1.2 목표
- Go 코어 팀 공식 준표준 라이브러리인 **`golang.org/x/text/language` 및 `golang.org/x/text/message`**를 도입하여 표준적이고 안정적인 i18n 인프라 구축.
- **5단계 로케일 결정 우선순위** 구현 (CLI 플래그 > 환경변수 > 프로필 설정 > OS 환경변수 > 기본값 `en`).
- 기본 언어는 글로벌 표준인 **영어(`en`)**, 완벽한 **한국어(`ko`)** 번역 카탈로그 제공.

---

## 2. 영향 범위 분석 (Impact Analysis)

| 파일/패키지 | 변경 내용 및 역할 |
| :--- | :--- |
| `go.mod` | Go 공식 준표준 패키지 `golang.org/x/text` 추가 |
| `internal/i18n/` (신규) | • `i18n.go`: 로케일 해석(`ResolveLanguage`), `message.Printer` 래핑, `T()`, `Sprintf()` 제공<br>• `catalog.go`: 영어(en) 및 한국어(ko) 메시지 번역 사전 정의 |
| `internal/app/app.go` | 글로벌 `--lang` 플래그 추출 및 앱 시작 시 `i18n.Init` 초기화 |
| `internal/app/commands_*.go` | 하드코딩된 문자열(도움말, 에러, 안내 메시지)을 `i18n.Sprintf(...)`로 전환 |
| `pkg/format.go` | 터미널 테이블 헤더 및 상세 보기 라벨의 다국어화 |
| `internal/i18n/i18n_test.go` (신규) | 로케일 우선순위 결정 및 언어별 번역 출력 단위 테스트 |

---

## 3. 상세 설계 (Detailed Design)

### 3.1 로케일 결정 우선순위 (5-Tier Resolution Order)

```text
1. CLI 플래그:      --lang ko  (또는 --lang=en)
       ↓ (미지정 시)
2. 환경 변수:        JIRA_LANG=ko
       ↓ (미지정 시)
3. 프로필 설정:      ~/.config/jira/config 내 [profile] 섹션의 language = ko
       ↓ (미지정 시)
4. 시스템 OS 로케일:  $LC_ALL 또는 $LANG (예: ko_KR.UTF-8 ➔ ko)
       ↓ (미지정 시)
5. 기본값 (Fallback): language.English ("en")
```

### 3.2 패키지 인터페이스 (`internal/i18n`)

```go
package i18n

import "golang.org/x/text/language"

// Initialize i18n with active language tag.
func Init(tag language.Tag)

// Sprintf formats a message according to the active language.
func Sprintf(key string, args ...interface{}) string

// T returns a simple translated string.
func T(key string) string

// CurrentLanguage returns the active language tag.
func CurrentLanguage() language.Tag
```

### 3.3 메시지 키 네이밍 컨벤션
- `cmd.<command>.desc`: 커맨드 한 줄 설명 (예: `cmd.create.desc`)
- `cmd.<command>.usage`: 커맨드 사용법 헤더
- `err.<category>.<detail>`: 에러 메시지 (예: `err.create.summary_required`, `err.create.desc_required`)
- `table.header.<field>`: 테이블 헤더 (예: `table.header.key`, `table.header.status`)
- `msg.success.<action>`: 성공 메시지 (예: `msg.success.created`, `msg.success.moved`)

---

## 4. 단계별 구현 절차 (Implementation Steps)

### Phase 1. 의존성 추가 및 i18n 코어 패키지 구현
- [x] 머신 리더블 전체 메시지 인벤토리 추출 (`task/messages.json`, 127건).
- [x] Go 표준 `embed.FS` 및 `encoding/json` 기반 Flat Map 카탈로그 (`internal/i18n/locales/*.json`).
- [x] `internal/i18n/i18n.go`: 5단계 로케일 우선순위 해석기 및 단 5줄의 초경량 `Sprintf`/`T` 구현.

### Phase 2. CLI 라우터 및 글로벌 플래그 연동
- [x] `internal/app/app.go`에서 `--lang <tag>` / `--lang=<tag>` 글로벌 플래그 파싱 구현.
- [x] `~/.config/jira/config`의 `language` 필드 지원 추가.
- [x] `App.Run` 진입 시점에 활성 로케일 초기화 (`i18n.Init`).

### Phase 3. 기존 커맨드 및 포맷터 마이그레이션
- [x] `internal/app/commands_issue.go`: 에러 메시지, 도움말, 가드레일 안내문 다국어화.
- [x] `pkg/format.go`: 테이블 헤더, 메타데이터 라벨 다국어화.
- [x] `internal/app/commands_config.go`: 설정 마법사 안내문 다국어화.
- [x] `pkg/labels.go`: 표준 라벨 뷰 및 검증 에러 다국어화.

### Phase 4. 단위 테스트 작성 및 검증
- [x] `internal/i18n/i18n_test.go` 작성 (우선순위 해석, 영어/한국어 번역 일치성 검증).
- [x] `internal/app/app_test.go` 다국어 전환(`--lang en`, `--lang=ko`) 통합 테스트 작성.
- [x] 전체 테스트 스위트 100% Pass 확인 (`go test -v ./...`).

### Phase 5. 지식 베이스 동기화 및 완료 전환
- [x] `knowledge/commands-and-routing.md`에 i18n 아키텍처 및 우선순위 규칙 문서화.
- [x] `knowledge/configuration.md` 및 `docs/02_CONFIGURATION.md`에 `language` 설정 옵션 문서화.
- [x] `knowledge/log.md` 작업 기록 작성.

---

## 5. 상세 테스트 계획 (Detailed Test Plan)

### 5.1 테스트 전략 및 환경
- `internal/i18n` 단위 테스트: 실제 OS 환경변수 및 가상 설정을 격리 주입하여 5단계 로케일 우선순위가 정확히 동작하는지 검증.
- CLI 통합 테스트: `--lang en` 및 `--lang ko` 옵션을 주고 `jira create --help` 또는 에러 유발 시 해당 언어로 정확히 출력되는지 `stdout`/`stderr` 캡처 검증.

### 5.2 테스트 케이스 매트릭스 (Test Cases Matrix)

| ID | 카테고리 | 조건 및 입력 | 기대 결과 (Expected Result) | 검증 결과 |
| :---: | :--- | :--- | :--- | :---: |
| **TC-01** | 우선순위 1 | CLI 플래그 `--lang ko` (OS는 en) | 한국어(`ko`)로 메시지 출력 | **Pass** |
| **TC-02** | 우선순위 1 | CLI 플래그 `--lang en` (OS는 ko) | 영어(`en`)로 메시지 출력 | **Pass** |
| **TC-03** | 우선순위 2 | 환경변수 `JIRA_LANG=ko` | 한국어(`ko`)로 메시지 출력 | **Pass** |
| **TC-04** | 우선순위 3 | 프로필 설정 `language = ko` | 한국어(`ko`)로 메시지 출력 | **Pass** |
| **TC-05** | 우선순위 4 | OS 환경변수 `LANG=ko_KR.UTF-8` | 한국어(`ko`)로 자동 감지 | **Pass** |
| **TC-06** | 우선순위 5 | 모든 설정 없음 (기본값) | 영어(`en`)로 기본 출력 | **Pass** |
| **TC-07** | 번역 검증 | `i18n.Sprintf("cmd.create.usage")` (en) | "Usage: jira create <SUMMARY>..." 영문 출력 | **Pass** |
| **TC-08** | 번역 검증 | `i18n.Sprintf("cmd.create.usage")` (ko) | "사용법: jira create <SUMMARY>..." 한글 출력 | **Pass** |
| **TC-09** | 가드레일 에러 | `--lang en` 상태에서 설명 누락 | "Description (-d, --desc) is required." 영문 에러 출력 | **Pass** |
| **TC-10** | 가드레일 에러 | `--lang ko` 상태에서 설명 누락 | "상세 설명(-d, --desc)이 입력되지 않았습니다." 한글 에러 출력 | **Pass** |
| **TC-11** | 테이블 포맷 | `--lang en` 상태에서 `jira list` | 테이블 헤더: `KEY TYPE STATUS LABELS ASSIGNEE...` | **Pass** |
| **TC-12** | 테이블 포맷 | `--lang ko` 상태에서 `jira list` | 테이블 헤더: `KEY TYPE STATUS LABELS ASSIGNEE...` | **Pass** |

### 5.3 합격 기준 (Pass/Fail Criteria)
1. `go test ./...` 100% Pass.
2. 미등록 키 호출 시 패닉(Panic) 없이 키 문자열 그대로 반환(Graceful Fallback).
3. 영문/한글 메시지 간 인수(`%s`, `%d` 등) 포맷 불일치 0건.

---

## 6. 승인 및 피드백 체크리스트
- [x] 티켓 상태 '진행 중' 변경 완료
- [x] 상세 구현 및 테스트 계획 수립 완료
- [x] 사용자 승인 및 구현 검증 완료 (100% Pass)

