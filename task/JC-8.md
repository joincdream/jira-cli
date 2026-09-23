# 📋 [JC-8] CLI 인자/플래그 파싱 표준화 및 티켓 생성 가드레일 구현 계획

- **티켓 키**: [JC-8](https://joincdream.atlassian.net/browse/JC-8)
- **티켓 요약**: `[Bug & Improvement] CLI 인자/플래그 파싱 표준화 및 티켓 생성 유효성 검증 가드레일 강화`
- **담당자**: AI 에이전트 & 윤상배
- **상태**: 완료 (Done)
- **라벨**: `Development`, `QA`

---

## 💡 사용자 관점의 핵심 개선 효과 (What Changes for Users?)

기술적인 세부 사항을 몰라도, 본 작업이 완료되면 **사용자가 체감하는 변화 4가지**는 다음과 같습니다:

| 구분 | ❌ 현재의 불편함 (Before) | ✨ 개선 후 사용자 경험 (After) |
| :--- | :--- | :--- |
| **도움말 확인** | `jira create --help`를 입력하면 도움말은 안 나오고, **Jira에 제목이 `"--help"`인 쓰레기 티켓이 몰래 생성**되어 수동으로 지워야 함 | `jira create --help` 입력 시 티켓이 절대 생성되지 않고, **깔끔한 사용법과 옵션 안내문만 화면에 표시**됨 |
| **옵션 입력 순서** | 제목을 반드시 맨 앞에 적어야 해서, 옵션을 먼저 쓰면 에러가 나거나 제목으로 오인됨<br>*(예: `jira create -d "내용" "제목"` 실패)* | **옵션을 앞/뒤 어디에 쓰든 자연스럽게 인식**함<br>*(예: `jira create -d "내용" "제목"`과 `jira create "제목" -d "내용"` 모두 완벽 작동)* |
| **빈 티켓 방지** | 설명(`-d`)을 깜빡하고 제목만 쳐도 **본문이 텅 빈 무성의한 티켓이 덜컥 생성**되어 팀원들이 되물어봄 | 상세 설명이 누락되면 **생성을 즉시 멈추고 설명을 입력하도록 안내**함<br>*(간이 티켓이 필요할 때만 `--allow-empty-desc`로 생성)* |
| **날짜 오타 처리** | 마감일 형식을 틀리게 적으면(`2026/09/30`) 무거운 Jira 서버까지 갔다가 **알 수 없는 외계어 서버 에러(`400 Bad Request`)**를 마주침 | 터미널에서 0.1초 만에 **"마감일은 YYYY-MM-DD 형식으로 입력하세요"라고 친절하게 한글로 바로 알려줌** |

---

## 1. 개요 및 배경 (Background & Objectives)

### 1.1 현재 문제점
1. **`--help` 플래그 오작동 버그 (Critical)**:
   - `jira create --help` 실행 시, 첫 번째 인수인 `"--help"`를 티켓 요약(Summary) 문자열로 간주하고 실제 Jira Cloud에 제목이 `"--help"`인 티켓을 생성해버림.
   - `jira edit --help`, `jira get --help` 등 다른 서브커맨드에서도 동일하게 `--help`를 이슈 키로 오인함.
2. **순서 종속성 (Order Dependency) 문제**:
   - `args[0]`을 무조건 위치 인자(제목/키)로 처리하므로 `jira create -d "설명" "제목"`과 같이 플래그가 위치 인자보다 앞에 오면 파싱이 실패하거나 오작동함.
3. **데이터 무결성 가드레일 부재**:
   - 내용이 텅 빈 티켓(설명 누락)이 아무런 검증 없이 생성됨.
   - 제목이 한두 글자이거나 의미 없는 문자열도 그대로 통과됨.
   - 잘못된 마감일 형식(`--due 2026/09/30`)을 로컬에서 사전 검증하지 않고 API로 전송하여 `400 Bad Request` 유발.

### 1.2 목표
- **POSIX/GNU CLI 모범 사례 준수**:
  - `-h`, `--help`는 어떤 위치에 있든 최우선으로 도움말(Usage)을 출력하고 정상 종료(Exit Code 0).
  - 옵션과 위치 인자의 순서 독립성(Order Independence) 보장.
  - Double Dash(`--`) 지원으로 하이픈으로 시작하는 데이터 보호.
- **Fail-Fast 사전 검증(Client-side Guardrail) 구축**:
  - 제목 최소 5자 이상 및 플래그형 문자열(`-` 접두사) 차단.
  - 설명(`-d`) 기본 필수화 (간이 생성이 필요한 경우 `--allow-empty-desc` 플래그로 명시적 허용).
  - 마감일(`--due`) 로컬 포맷 검증.

---

## 2. 영향 범위 분석 (Impact Analysis)

| 파일 | 변경 대상 및 내용 |
| :--- | :--- |
| `internal/app/commands_issue.go` | • `CreateCommand`: 위치 인자/플래그 파싱 리팩토링, 유효성 검증 가드레일 추가<br>• `EditCommand`: `--help` 선제 처리 및 순서 독립적 파싱 적용<br>• `GetCommand`, `DeleteCommand`, `MoveCommand`, `CommentCommand`: `--help` 처리 및 단일 인자 검증 로직 개선 |
| `internal/app/app.go` / `command.go` | 필요 시 공통 인자 정규화(Flag/Positional 분리) 헬퍼 함수 배치 |
| `internal/app/commands_issue_test.go` | 신규 단위 테스트 추가 (다양한 인자 순서, `--help` 정상 종료, 가드레일 실패 검증) |
| `docs/03_FEATURES_AND_COMMANDS.md` | `create` 커맨드 옵션(`--allow-empty-desc`) 및 변경된 사용법 반영 |
| `knowledge/commands-and-routing.md` | 플래그 파싱 규칙 및 가드레일 정책 동기화 |

---

## 3. 상세 설계 (Detailed Design)

### 3.1 Go 표준 `flag`의 한계 극복을 위한 인자 정규화 (Argument Normalization)
Go 표준 라이브러리 `flag.FlagSet`은 첫 번째 비플래그(non-flag) 인자를 만나는 즉시 파싱을 중단하는 제약이 있습니다. 
외부 라이브러리 없이 Go 표준 라이브러리만으로 이를 해결하기 위해, `args`를 사전에 분리하여 재정렬하는 헬퍼 함수를 설계합니다:

```go
// parseArgsAndFlags separates flags and positional arguments, respecting '--'.
// It checks for -h / --help early and returns appropriate flags.
```

### 3.2 `CreateCommand` 사전 검증 가드레일 (Validation Rules)

1. **도움말 검사**:
   - `args` 내에 `-h` 또는 `--help`가 포함된 경우 `fs.Usage()`를 `stdout`으로 출력하고 `nil` 반환 (Exit 0).
2. **제목(Summary) 검증**:
   - 위치 인자가 1개 미만인 경우 에러 (`"이슈 제목(Summary)을 입력해야 합니다"`).
   - `len(strings.TrimSpace(summary)) < 5` 인 경우 에러 (`"이슈 제목은 최소 5자 이상이어야 합니다"`).
   - `strings.HasPrefix(summary, "-")` 인 경우 에러 (`"이슈 제목이 플래그 형식('-')으로 시작할 수 없습니다. 문자 그대로 입력하려면 '--' 뒤에 지정하세요"`).
3. **상세 설명(Description) 검증**:
   - `strings.TrimSpace(desc) == ""`이고 `!allowEmptyDesc`인 경우 에러:
     ```text
     ❌ 티켓 생성 실패: 상세 설명(-d, --desc)이 입력되지 않았습니다.
     💡 팀 협업 및 명확한 작업 추적을 위해 본문 설명은 필수입니다.
        (설명 없이 간이 티켓을 생성하려면 '--allow-empty-desc' 플래그를 사용하세요)
     ```
4. **마감일(Due Date) 검증**:
   - `due != ""`인 경우 `time.Parse("2006-01-02", due)` 검증. 유효하지 않으면 즉시 에러 반환.

---

## 4. 단계별 구현 절차 (Implementation Steps)

### Phase 1. 공통 파서 및 헬퍼 구현
- [x] `internal/app` 내에 인자 재정렬(Flag Normalizer) 및 선제적 `--help` 감지 헬퍼 함수 작성 (`internal/app/flags.go`).
- [x] Double Dash(`--`) 이후 인자를 순수 위치 인자로 보호하는 로직 구현.

### Phase 2. `CreateCommand` 리팩토링 및 가드레일 적용
- [x] `CreateCommand.Execute`를 신규 파서 구조로 전면 교체.
- [x] `--allow-empty-desc` 플래그 추가.
- [x] 제목 최소 5자(`utf8.RuneCountInString`), 플래그 오인 방지, 마감일 날짜 검증, 본문 필수 검증 가드레일 통합.

### Phase 3. 기타 이슈 커맨드(`edit`, `get`, `delete`, `move` 등) 수평 전개
- [x] `EditCommand`: `--help` 안전 처리, 순서 독립적 파싱 및 마감일 로컬 유효성 검증 적용.
- [x] `GetCommand`, `DeleteCommand`, `MoveCommand`, `CommentCommand`, `TransitionsCommand`, `ListCommand`: `--help` 시 사용법 정상 출력 (Exit 0) 보장.

### Phase 4. 단위 테스트 작성 및 검증
- [x] 아래 '5. 상세 테스트 계획'의 테스트 매트릭스(TC-01 ~ TC-15) 구현 (`internal/app/commands_issue_test.go`).
- [x] 전체 테스트 스위트 100% 통과 확인 (`go test ./...` PASS).

### Phase 5. 문서화 및 티켓 상태 완료 전환
- [x] `docs/03_FEATURES_AND_COMMANDS.md` 및 `knowledge/commands-and-routing.md` 업데이트.
- [x] `knowledge/log.md` 작업 로그 기록.
- [x] `jira comment JC-8 "구현 및 검증 완료"` 및 `jira move JC-8 "완료"`.

---

## 5. 상세 테스트 계획 (Detailed Test Plan)

### 5.1 테스트 전략 및 환경 (Strategy & Mocking)
- **격리된 단위 테스트 (Zero Network Call)**:
  - 실제 Jira Cloud API를 호출하지 않고, 기존 `internal/app/app_test.go`의 Mock 패턴(`MockClientProvider`, `MockCatalogLoader`)을 활용하여 100% 로컬 메모리 환경에서 테스트를 수행합니다.
- **I/O 및 에러 검증**:
  - `bytes.Buffer`로 `stdout` 및 `stderr`를 캡처하여 도움말 내용, 에러 메시지 형식, 그리고 반환된 `error` 객체의 유효성을 정밀 검증합니다.

### 5.2 테스트 케이스 매트릭스 (Test Cases Matrix)

| ID | 카테고리 | 입력 명령 (CLI Arguments) | 기대 결과 (Expected Result) | 검증 포인트 |
| :---: | :--- | :--- | :--- | :--- |
| **TC-01** | 도움말 파싱 | `jira create --help` | • Exit Code 0 (nil 반환)<br>• stdout에 사용법(Usage) 출력<br>• Mock API 호출 0회 | `--help` 시 티켓 생성 방지 |
| **TC-02** | 도움말 파싱 | `jira create -h` | • Exit Code 0 (nil 반환)<br>• stdout에 사용법 출력<br>• Mock API 호출 0회 | 단축 플래그 `-h` 지원 |
| **TC-03** | 순서 독립성 | `jira create -d "본문" "유효한 제목입니다"` | • 성공 (Exit 0)<br>• Summary="유효한 제목입니다", Desc="본문" | 플래그가 위치 인자보다 앞설 때 |
| **TC-04** | 순서 독립성 | `jira create "유효한 제목입니다" -d "본문"` | • 성공 (Exit 0)<br>• Summary="유효한 제목입니다", Desc="본문" | 위치 인자가 플래그보다 앞설 때 (기존 호환) |
| **TC-05** | Double Dash | `jira create -- "-d로 시작하는 제목" -d "설명"` | • 성공 (Exit 0)<br>• Summary="-d로 시작하는 제목" | `--` 이후 위치 인자 플래그 오인 방지 |
| **TC-06** | 가드레일 (인자 누락) | `jira create` | • 에러 반환 (Exit 1)<br>• stderr에 "이슈 제목(Summary)을 입력해야 합니다" | 필수 위치 인자 누락 차단 |
| **TC-07** | 가드레일 (제목 길이) | `jira create "짧음"` | • 에러 반환 (Exit 1)<br>• stderr에 "최소 5자 이상" 안내 | 5자 미만 제목 차단 |
| **TC-08** | 가드레일 (공백 제목) | `jira create "     "` | • 에러 반환 (Exit 1)<br>• stderr에 "공백 제외 최소 5자 이상" 안내 | 공백 문자열 차단 |
| **TC-09** | 가드레일 (플래그형 제목) | `jira create "-invalid" -d "설명"` | • 에러 반환 (Exit 1)<br>• stderr에 "플래그 형식('-') 시작 불가" 안내 | `-` 접두사 오인 차단 |
| **TC-10** | 가드레일 (설명 누락) | `jira create "유효한 제목입니다"` | • 에러 반환 (Exit 1)<br>• stderr에 "상세 설명(-d, --desc) 필수" 안내<br>• `--allow-empty-desc` 옵션 가이드 출력 | 빈 본문 티켓 생성 차단 |
| **TC-11** | 가드레일 (설명 예외) | `jira create --allow-empty-desc "유효한 제목입니다"` | • 성공 (Exit 0)<br>• Summary="유효한 제목입니다", Desc="" | `--allow-empty-desc` 정상 작동 |
| **TC-12** | 가드레일 (날짜 포맷) | `jira create "유효한 제목" -d "설명" --due 2026/12/31` | • 에러 반환 (Exit 1)<br>• stderr에 "YYYY-MM-DD 형식이어야 합니다" | 로컬 날짜 정규식/파싱 검증 |
| **TC-13** | 가드레일 (정상 날짜) | `jira create "유효한 제목" -d "설명" --due 2026-12-31` | • 성공 (Exit 0)<br>• Due="2026-12-31" | 유효한 날짜 정상 전달 |
| **TC-14** | 회귀 (edit 도움말) | `jira edit --help` | • Exit Code 0<br>• stdout에 사용법 출력, API 호출 0회 | edit 커맨드 `--help` 안전성 |
| **TC-15** | 회귀 (get 도움말) | `jira get --help` | • Exit Code 0<br>• stdout에 사용법 출력, API 호출 0회 | get 커맨드 `--help` 안전성 |

### 5.3 합격 기준 (Pass/Fail Criteria)
1. **단위 테스트 통과율 100%**: `go test -v ./internal/app/...` 실행 시 TC-01부터 TC-15까지 모든 케이스 통과.
2. **네트워크 격리 확인**: 테스트 실행 중 실제 Atlassian API 엔드포인트로의 네트워크 트래픽 0건.
3. **사용자 친화적 에러 메시지**: 가드레일 위반 시 단순 실패가 아니라, 부족한 항목과 해결 방법(예: `--allow-empty-desc` 가이드)이 `stderr`에 명확히 출력되는지 문자열 검증.

---

## 6. 승인 및 피드백 체크리스트
- [x] 티켓 상태 '진행 중' 변경 완료
- [x] 상세 테스트 계획(Test Plan) 및 매트릭스 수립 완료
- [ ] 구현 계획 승인 후 Phase 1 구현 착수
