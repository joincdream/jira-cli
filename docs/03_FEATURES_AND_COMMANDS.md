# 📖 03. 기능 명세 및 CLI 레퍼런스 (Features & CLI Reference)

이 문서는 `jira` CLI에서 제공하는 모든 명령어의 상세 명세, 인수(Arguments), 플래그(Flags), 내부 처리 흐름 및 출력 형식을 기술합니다.

---

## 📋 명령어 요약 매트릭스

| 명령어 | 별칭 (Aliases) | 목적 | 대상 Jira API v3 엔드포인트 |
| :--- | :--- | :--- | :--- |
| `list` | `ls` | 이슈 목록 조회 (JQL 필터링) | `GET /rest/api/3/search/jql` |
| `get` | `view`, `show` | 단일 이슈 상세 조회 (코멘트/하위작업 포함) | `GET /rest/api/3/issue/{key}` |
| `create` | `new`, `add` | 신규 이슈 생성 | `POST /rest/api/3/issue` |
| `edit` | `update`, `modify` | 기존 이슈 수정 | `PUT /rest/api/3/issue/{key}` |
| `move` | `transition`, `status` | 이슈 상태 워크플로우 전이 | `POST /rest/api/3/issue/{key}/transitions` |
| `comment` | - | 이슈에 코멘트 추가 | `POST /rest/api/3/issue/{key}/comment` |
| `transitions`| - | 변경 가능한 상태 목록 조회 | `GET /rest/api/3/issue/{key}/transitions` |
| `delete` | `rm` | 이슈 삭제 | `DELETE /rest/api/3/issue/{key}` |
| `labels` | `tags` | 표준 라벨 카탈로그 조회 | 로컬 `labels.json` 파싱 |
| `configure` | `configuration`, `config` | 대화형 설정 마법사 | 로컬/전역 파일 I/O |
| `version` | `-v`, `--version` | 버전 출력 | 내부 상수 (`DefaultVersion`) |
| `help` | `-h`, `--help` | 전체 도움말 출력 | - |

---

## 🔍 상세 명령어 명세

### 1. `list` (이슈 목록 조회)
- **문법**: `jira list [JQL_QUERY]`
- **인수**:
  - `JQL_QUERY` (선택): Jira Query Language 문자열. 생략 시 기본값은 `project = <PROJECT_KEY> ORDER BY created DESC`.
- **처리 흐름**:
  1. 클라이언트 초기화 및 JQL 구성.
  2. 요청 필드 제한: `summary`, `status`, `issuetype`, `priority`, `labels`, `assignee`, `duedate`, `created`, `updated`, `project`, `parent`, `subtasks`.
  3. `pkg.PrintIssuesTable()`을 통해 고정폭 터미널 테이블로 렌더링.
- **예시**:
  ```bash
  jira list
  jira list "labels = AI and status = '진행 중'"
  jira list "assignee is EMPTY order by priority desc"
  ```

---

### 2. `get` (이슈 상세 조회)
- **문법**: `jira get <KEY>`
- **인수**:
  - `<KEY>` (필수): Jira 이슈 키 (대소문자 무관, 자동으로 대문자 변환됨, 예: `kan-1` → `KAN-1`).
- **처리 흐름**:
  1. `GET /rest/api/3/issue/{key}` 호출.
  2. 메타데이터(상태, 담당자, 마감일 등) 출력.
  3. ADF 포맷의 Description 본문을 평문 텍스트로 변환하여 출력.
  4. 하위 작업(Subtasks) 및 코멘트(Comments) 내역 목록 출력.
- **예시**:
  ```bash
  jira get KAN-12
  ```

---

### 3. `create` (신규 이슈 생성)
- **문법**: `jira create <SUMMARY> -d <DESC> [flags]` (인자와 플래그의 순서 독립성 지원)
- **인수**:
  - `<SUMMARY>` (필수): 이슈 제목 (최소 5자 이상, 플래그 오인 방지 검증).
- **지원 플래그**:
  - `-d, --desc string`: 본문 상세 설명 (Markdown 형식 지원, 기본 필수).
  - `--allow-empty-desc`: 상세 설명 없이 간이 티켓 생성 허용.
  - `-t, --type string`: 이슈 유형 (기본값: `작업`, 선택: `스토리`, `에픽`, `Subtask`, `Bug` 등).
  - `-l, --labels string`: 쉼표로 구분된 라벨 목록 (예: `AI,Planning`).
  - `--due string`: 마감일 (`YYYY-MM-DD` 형식, 로컬 사전 검증).
  - `-p, --project string`: 대상 프로젝트 키 (생략 시 기본 프로젝트 사용).
  - `--parent string`: 상위 이슈 키 (`Subtask` 생성 시 필수).
  - `--force-labels`: 표준 카탈로그에 정의되지 않은 임의의 라벨 등록 허용.
- **가드레일 및 처리 흐름**:
  1. `--help` 또는 `-h` 선제 감지 시 티켓 생성 없이 사용법 출력 및 `Exit 0`.
  2. 위치 인자와 플래그 순서 무관 파싱 (예: `jira create -d "설명" "제목"` 지원).
  3. Double Dash(`--`) 지원으로 `-`로 시작하는 제목 보호 (`jira create -- "-d 제목" -d "설명"`).
  4. 제목 최소 5자 이상 검증 및 설명 필수 검증 (누락 시 친절한 안내와 함께 로컬 차단).
  5. 마감일 입력 시 `YYYY-MM-DD` 형식 사전 검증.
  6. `--force-labels`가 없는 경우, `pkg.LoadLabelCatalog()`를 로드하여 입력된 라벨이 표준 카탈로그에 속하는지 검증하고 표준 대소문자로 자동 정규화.
  7. 본문 텍스트(`-d`)를 `pkg.BuildADFDocument()`로 ADF JSON 구조로 변환하여 `POST /rest/api/3/issue` 호출.
- **예시**:
  ```bash
  # 표준 생성 (순서 자유)
  jira create "RAG 파이프라인 PoC 개발" -d "### 1. 목적\n- 벡터 검색 성능 평가" -l "AI,PoC" --due 2026-09-25
  jira create -d "상세 설명" "RAG 파이프라인 PoC 개발"

  # 본문 없이 간이 티켓 생성
  jira create --allow-empty-desc "간이 작업 티켓"
  ```

---

### 4. `edit` (이슈 수정)
- **문법**: `jira edit <KEY> [flags]`
- **인수**:
  - `<KEY>` (필수): 수정할 이슈 키.
- **지원 플래그**:
  - `-s, --summary string`: 변경할 제목.
  - `-d, --desc string`: 변경할 상세 설명 (Markdown).
  - `-l, --labels string`: 새로 교체할 라벨 목록 (쉼표 구분).
  - `--due string`: 마감일 (`YYYY-MM-DD`, 또는 마감일 제거 시 `none` / `null`).
  - `--parent string`: 변경할 상위 이슈 키 (제거 시 `none`).
  - `--force-labels`: 카탈로그 미등록 라벨 허용.
- **처리 흐름**:
  1. 수정 플래그 중 하나 이상 지정되었는지 확인.
  2. 라벨 지정 시 유효성 검사 및 정규화 수행.
  3. 마감일 또는 상위 이슈가 `none`인 경우 `null` 페이로드를 구성하여 Jira 필드 초기화.
  4. `PUT /rest/api/3/issue/{key}` 호출.
- **예시**:
  ```bash
  jira edit KAN-5 --due 2026-10-01
  jira edit KAN-5 --due none
  jira edit KAN-5 -l "Architecture,Cloud"
  ```

---

### 5. `move` (상태 워크플로우 전이)
- **문법**: `jira move <KEY> <STATUS>`
- **인수**:
  - `<KEY>` (필수): 이슈 키.
  - `<STATUS>` (필수): 변경하고자 하는 대상 상태 이름 또는 전이(Transition) ID.
    - 대소문자 무관 비교 (예: `진행 중`, `완료`, `해야 할 일`, `In Progress`, `Done`).
- **처리 흐름**:
  1. `GET /rest/api/3/issue/{key}/transitions`를 호출하여 현재 이슈에서 전이 가능한 상태 목록을 동적으로 조회.
  2. 대상 상태 이름 또는 ID와 일치하는 Transition ID 탐색.
  3. 일치하는 전이가 없으면 친절하게 전이 가능한 전체 후보 목록을 에러로 출력.
  4. `POST /rest/api/3/issue/{key}/transitions` 페이로드 전송.
- **예시**:
  ```bash
  jira move KAN-1 "진행 중"
  jira move KAN-1 "완료"
  ```

---

### 6. `comment` (코멘트 추가)
- **문법**: `jira comment <KEY> <MESSAGE>`
- **인수**:
  - `<KEY>` (필수): 대상 이슈 키.
  - `<MESSAGE>` (필수): 등록할 코멘트 내용 (Markdown 지원).
- **처리 흐름**:
  1. 메시지를 `BuildADFDocument()`를 통해 ADF 블록으로 변환.
  2. `POST /rest/api/3/issue/{key}/comment` 호출.
- **예시**:
  ```bash
  jira comment KAN-1 "1차 PR 머지 완료되었습니다."
  ```

---

### 7. `transitions` (전이 가능 상태 목록)
- **문법**: `jira transitions <KEY>`
- **출력 내용**: 해당 이슈가 현재 상태에서 이동할 수 있는 대상 상태 목록과 전이 ID 출력.

---

### 8. `delete` (이슈 삭제)
- **문법**: `jira delete <KEY>`
- **옵션**: 내부적으로 하위 작업이 있을 경우 함께 삭제(`deleteSubtasks=true`)하도록 API 호출.

---

### 9. `labels` (표준 라벨 카탈로그 조회)
- **문법**: `jira labels`
- **출력 내용**: `project`, `tech`, `activity` 카테고리별 표준 라벨명과 설명을 터미널에 구조화하여 출력.

---

### 10. `configure` (대화형 프로필 설정 및 조회)
- **문법**:
  - `jira configure [--profile <name>]` : 지정 프로필(기본: `default`) 대화형 설정
  - `jira configure list` : 등록된 프로필 목록 및 현재 활성 프로필 조회
- **파일 위치**: `~/.config/jira/config` (INI 포맷, 권한 `0600`)
- **전역 플래그**:
  - `--profile <name>` : 사용할 프로필 지정 (환경 변수: `JIRA_PROFILE`)
  - `--lang <en|ko>` : UI 표시 언어 설정 (기본: `en`, 환경 변수: `JIRA_LANG`)
- **예시**:
  ```bash
  jira configure
  jira configure --profile cloit
  jira configure list
  jira --profile cloit --lang ko list
  ```

