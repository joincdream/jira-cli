# 🚀 Jira CLI (`jira`)

> **초경량, 고성능, 무의존성(Zero-Dependency) Go 기반 Atlassian Jira Cloud CLI 도구**

Jira CLI는 개발자와 AI 에이전트(LLM)가 터미널 환경에서 Atlassian Jira 작업을 가장 빠르고 직관적으로 수행할 수 있도록 설계된 도구입니다. 외부 서드파티 라이브러리 없이 Go 표준 라이브러리만으로 구현되어 단일 바이너리로 즉시 실행 가능합니다.

---

## ✨ 주요 특징 (Key Features)

- **Zero Dependencies**: 외부 서드파티 모듈 없이 Go 표준 라이브러리(`net/http`, `encoding/json`, `flag` 등)만으로 구동되어 빌드가 매우 빠르고 공급망 보안에 안전합니다.
- **내장 ADF (Atlassian Document Format) 엔진**: 일반 Markdown 텍스트(제목, 목록, 코드블록, 인용문, 굵은 글씨 등)를 Jira Cloud v3의 풍부한 ADF 구조로 자동 변환하여 등록하며, ADF 응답을 터미널에 읽기 쉬운 텍스트로 복원합니다.
- **AWS CLI 스타일 멀티 프로필 지원 (INI Configuration)**:
  - `~/.config/jira/config` 단일 파일에서 `[default]`, `[work]`, `[personal]` 등 여러 계정을 관리하며, `--profile` 플래그 또는 `JIRA_PROFILE` 환경 변수로 즉시 전환합니다.
- **표준 라벨 카탈로그 검증**: 프로젝트 표준 라벨(`labels.json`)을 기반으로 오타 방지 및 대소문자 정규화를 수행하여 일관된 태그 품질을 유지합니다.
- **대화형 설정 마법사**: `jira configure [--profile <name>]` 명령어를 통해 손쉽게 프로필을 생성/수정하고 `jira configure list`로 조회합니다.
- **AI Agent 친화적 설계**: 표준 입출력(stdout/stderr)과 명확한 종료 코드(Exit Code)를 제공하여 Antigravity CLI 등 LLM 에이전트와의 협업에 최적화되어 있습니다.

---

## 📂 프로젝트 아키텍처 요약

```text
.
├── cmd/
│   └── main.go                 # 엔트리포인트 (OS 시그널 처리 및 App 실행)
├── internal/
│   └── app/
│       ├── app.go              # CLI 라우터, 전역 플래그(--profile), 의존성 주입
│       ├── command.go          # Command 인터페이스 정의
│       ├── commands_issue.go   # Jira 이슈 관련 명령어 (list, get, create, edit, move 등)
│       ├── commands_config.go  # 대화형 설정 및 프로필 마법사 (configure, configure list)
│       └── commands_meta.go    # 메타 명령어 (help, version, labels)
├── pkg/
│   ├── client.go               # Jira REST API v3 클라이언트 & INI 프로필 로더
│   ├── format.go               # 터미널 테이블 및 상세 출력 포맷터
│   ├── labels.go               # 라벨 카탈로그 관리자 및 유효성 검사기
│   └── types.go                # 데이터 구조체 및 Markdown <-> ADF 변환 엔진
├── docs/                       # 기획 / 설계 / 기능 / 확장 상세 문서 (LLM 및 유지보수용)
├── Makefile                    # 빌드, 설치, 테스트 자동화
└── go.mod
```

---

## 🛠️ 빌드 및 설치 (Build & Installation)

### 요구사항
- Go 1.22 이상

### 바이너리 빌드
```bash
make build
# 빌드 결과물: ../../bin/jira (또는 bin/jira)
```

### 전역 설치 (`~/.local/bin/jira`)
```bash
make install
```
> `~/.local/bin`이 `$PATH`에 등록되어 있는지 확인하세요.

---

## ⚙️ 설정 가이드 (Configuration)

### 1. 대화형 마법사로 설정하기 (권장)

```bash
# 기본(default) 프로필 대화형 설정
jira configure

# 특정 프로필 생성 또는 수정
jira configure --profile cloit

# 등록된 프로필 목록 확인
jira configure list
```

### 2. 수동 설정 파일 구성 (`~/.config/jira/config`)

```ini
# Jira CLI Configuration (권한: 0600)
[default]
instance_url = https://joincdream.atlassian.net
email = joinc.dream@gmail.com
api_token = your-atlassian-api-token
project_key = KAN

[work]
instance_url = https://work-domain.atlassian.net
email = user@work.com
api_token = work-api-token
project_key = WORK
```

*Jira API Token은 [Atlassian 계정 관리 콘솔](https://id.atlassian.com/manage-profile/security/api-tokens)에서 발급받을 수 있습니다.*

---

## 📖 핵심 명령어 사용법 (Usage)

### 1. 이슈 목록 조회 (`list`, 별칭: `ls`)
```bash
# 기본 활성 이슈 목록 조회 (터미널 테이블)
jira list

# 생성형 AI 및 LLM 프롬프트용 마크다운 테이블 출력
jira list --md
jira list -o md

# AI Agent 및 자동화 파이프라인 연동용 JSON 출력
jira list --json
jira list -o json

# JQL 조건 검색
jira list "labels = AI and status = '진행 중'" --md
jira list "duedate <= '2026-09-30'"
```

### 2. 이슈 상세 조회 (`get`, 별칭: `view`, `show`)
```bash
# 기본 터미널 상세 조회
jira get KAN-10

# 생성형 AI가 전체 맥락(설명, 하위작업, 코멘트)을 이해하기 최적화된 마크다운 문서 출력
jira get KAN-10 --md

# 원본 구조화된 JSON 데이터 출력
jira get KAN-10 --json
```

### 3. 이슈 생성 (`create`, 별칭: `new`, `add`)
```bash
# 기본 작업(Task) 생성
jira create "신규 기능 개발" -d "상세 요구사항 명세" -l "AI,Planning" --due 2026-09-30

# 스토리 또는 에픽 유형으로 생성
jira create "데이터 수집 파이프라인" -t 스토리 -l "Architecture"

# 하위 작업(Subtask) 생성
jira create "단위 테스트 작성" -t Subtask --parent KAN-10
```

### 4. 이슈 정보 수정 (`edit`, 별칭: `update`, `modify`)
```bash
# 마감일 변경
jira edit KAN-10 --due 2026-10-05

# 마감일 제거
jira edit KAN-10 --due none

# 라벨 교체
jira edit KAN-10 -l "AI,Development,QA"

# 제목 및 설명 수정
jira edit KAN-10 -s "변경된 제목" -d "업데이트된 본문 설명"
```

### 5. 상태 전이 (`move`, 별칭: `transition`, `status`)
```bash
# 상태 변경 (이름 또는 ID 지원)
jira move KAN-10 "진행 중"
jira move KAN-10 "완료"

# 전이 가능한 상태 확인
jira transitions KAN-10
```

### 6. 코멘트 추가 (`comment`)
```bash
jira comment KAN-10 "1차 검토 완료. PR 머지 대기 중입니다."
```

### 7. 이슈 삭제 (`delete`, 별칭: `rm`)
```bash
jira delete KAN-10
```

### 8. 표준 라벨 카탈로그 조회 (`labels`, 별칭: `tags`)
```bash
jira labels
```

---

## 🧪 테스트 실행

```bash
make test
# 또는
go test -v ./...
```

---

## 📚 지식 베이스 및 개발 문서 (Knowledge & Docs)

### 🧠 Google OKF (Open Knowledge Format) 지식 베이스 (`knowledge/`)
생성형 AI(LLM)와 개발자가 코드베이스를 원활히 유지보수하고 기능을 확장할 수 있도록 Google Open Knowledge Format 규격의 정형화된 지식 베이스를 제공합니다:
- [knowledge/index.md](knowledge/index.md) - 전체 지식 카탈로그 및 AI 라우팅 인덱스
- [knowledge/architecture.md](knowledge/architecture.md) - 3-Tier 시스템 아키텍처 및 의존성 주입 설계
- [knowledge/configuration.md](knowledge/configuration.md) - INI 멀티 프로필 및 `.jira-profile` 자동 탐색
- [knowledge/commands-and-routing.md](knowledge/commands-and-routing.md) - CLI 명령어 및 머신 리더블(`--md`, `--json`) 포맷 엔진
- [knowledge/adf-engine.md](knowledge/adf-engine.md) - Markdown ↔ ADF 변환 엔진
- [knowledge/data-models.md](knowledge/data-models.md) - Jira REST API v3 DTO 매핑
- [knowledge/labels-catalog.md](knowledge/labels-catalog.md) - 표준 라벨 검증 및 정규화 정책
- [knowledge/log.md](knowledge/log.md) - 지식 베이스 변경 및 검증 이력

### 📄 상세 설계 문서 (`docs/`)
- [01. 시스템 아키텍처 및 설계 원칙](docs/01_ARCHITECTURE.md)
- [02. 설정 및 인증 아키텍처](docs/02_CONFIGURATION.md)
- [03. 기능 명세 및 CLI 레퍼런스](docs/03_FEATURES_AND_COMMANDS.md)
- [04. ADF (Atlassian Document Format) 변환 엔진](docs/04_ADF_ENGINE.md)
- [05. 라벨 정책 및 카탈로그 시스템](docs/05_LABELS_MANAGEMENT.md)
- [06. 개발 및 확장 가이드 (LLM & Developer Guide)](docs/06_EXTENDING_GUIDE.md)

---

## 📄 라이선스 (License)

이 프로젝트는 MIT 라이선스에 따라 배포됩니다.
