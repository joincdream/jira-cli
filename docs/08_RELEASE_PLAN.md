# 🚀 08. Jira CLI 정식 릴리즈 계획서 (Release Plan)

> **문서 상태**: Approved for Execution  
> **최종 배포 버전**: v1.0.0 (Official Open Source Public Release)  
> **최종 수정일**: 2026-10-19  
> **책임자**: Antigravity AI Assistant & 윤상배  

---

## 📌 1. 개요 및 릴리즈 목표 (Executive Summary)

### 1.1 배경 및 목적
Jira CLI(`jira`)는 **Zero Third-Party Dependencies(순수 Go 기반)**, **자체 ADF ↔ Markdown 양방향 변환 엔진**, **AI 에이전트(LLM) 친화적 I/O**, 그리고 **오프라인 로컬 캐시와 실시간 Watch 엔진/TUI 대시보드**를 핵심 경쟁력으로 완성되었습니다.

본 계획서는 `jira` CLI를 글로벌 오픈소스 생태계와 일반 개발자, 그리고 Cursor, Claude Code, Antigravity 등을 사용하는 AI 엔지니어들에게 정식 버전인 **v1.0.0**으로 공개하기 위해 필요한 **사용자 온보딩 경험 최적화(Onboarding DX)**, **빌드/배포 자동화**, **다국어 문서화(영어 우선/한국어 지원)**, **오픈소스 거버넌스**, **품질 보증(QA)**, **GTM(Go-to-Market)** 실행 단계를 구체화합니다.

### 1.2 릴리즈 핵심 원칙 (Core Principles)
1. **버전 체계 확립 (v1.0.0)**: 기능의 완성도와 안정성을 갖춘 정식 첫 메이저 릴리즈(`v1.0.0`)로 출시.
2. **첫 3분 온보딩 경험 최적화 (Frictionless Onboarding)**:
   - AWS CLI 스타일의 친절하고 직관적인 인터랙티브 글로벌 설정(`jira configure`).
   - 로컬 작업 디렉토리(Git 저장소)와 Jira 프로젝트를 매끄럽게 연결하는 대화형 초기화(`jira init`).
   - 설정 입력 직후 Atlassian API 연결 상태(Connection Ping) 즉시 검증 및 피드백.
3. **글로벌 퍼스트 문서화 (Global-First Documentation)**:
   - 루트 `README.md`는 글로벌 오픈소스 개발자 및 AI 에이전트를 위해 **영어(English)**로 우선 작성 및 노출.
   - 한국어 사용자를 위해 동일한 품질의 **`README.ko.md`**를 병행 제공하며, 문서 최상단에 언어 전환 네비게이션 배치 (`🌐 English | [한국어](README.ko.md)`).
4. **단일 정적 바이너리 배포**: CGO 의존성이 없는 순수 Go 바이너리로 Linux, macOS(Intel/Apple Silicon), Windows를 완벽 지원.
5. **원클릭 설치 환경**: Homebrew Tap 및 원라인 curl 셸 스크립트를 통한 5초 이내 설치 환경 구축.
6. **Agent-Native 포지셔닝**: AI 에이전트가 즉시 활용할 수 있는 `SKILL.md` 및 에이전트 도구 등록 템플릿 제공.
7. **결점 없는 품질과 보안**: 전체 테스트 스위트 100% Pass, 민감 정보 유출 제로, 설정 파일 권한(`0600`) 강제.

---

## 🏗️ 2. 릴리즈 작업 분류 체계 (WBS: Work Breakdown Structure)

```
Jira CLI v1.0.0 Release
├── Track 1: 사용자 온보딩 경험 최적화 (Onboarding DX & Configuration)
├── Track 2: 빌드 & 배포 자동화 파이프라인 (Build & Automation)
├── Track 3: 문서화 및 오픈소스 거버넌스 (Documentation & Governance)
├── Track 4: 품질 보증 및 사전 검증 (QA & Security Verification)
└── Track 5: 릴리즈 발행 및 커뮤니티 확산 (Release Execution & GTM)
```

---

### Track 1: 사용자 온보딩 경험 최적화 (Onboarding DX & Configuration)

신규 사용자가 바이너리를 다운로드하고 첫 명령어를 실행하기까지의 이탈률을 최소화하기 위한 온보딩 핵심 트랙입니다.

| 작업 ID | 태스크 명 | 상세 내용 및 산출물 | 우선순위 |
| :--- | :--- | :--- | :--- |
| **REL-O1** | **글로벌 설정 마법사 고도화 (`jira configure`)** | • **도메인 입력 정규화**: `company` 또는 `company.atlassian.net` 입력 시 `https://company.atlassian.net` 자동 완성 (하드코딩된 개인 도메인 기본값 제거 및 예시 플레이스홀더 제공)<br>• **토큰 발급 가이드 연동**: API Token 발급 공식 링크 안내 (`https://id.atlassian.com/manage-profile/security/api-tokens`) 및 터미널 마스킹 입력<br>• **연결 상태 즉시 검증 (Connection Ping)**: 입력 완료 직후 Jira API(`/rest/api/3/myself`)를 호출하여 인증 성공 여부 및 사용자 계정명 즉각 피드백 (`✅ 연결 성공: 윤상배 님 계정이 확인되었습니다.`). 실패 시 재입력 기회 제공 | P0 |
| **REL-O2** | **로컬 프로젝트 대화형 초기화 (`jira init`) 신설** | • **디렉토리 컨텍스트 바인딩**: 현재 작업 디렉토리(Git 저장소 등)에서 `jira init` 실행 시 상호작용형 위저드 제공<br>• **프로필 선택/매핑**: 기존 등록된 글로벌 프로필 목록을 보여주고 선택하거나 신규 프로필 생성 유도<br>• **로컬 설정 파일 생성**: 현재 디렉토리에 `.jira-profile`을 자동 생성하고 기본 프로젝트 키 바인딩<br>• **Git 연동 안내**: Git 저장소인 경우 `.gitignore` 등록 안내 및 현재 브랜치 감지 연계 결과 피드백 | P0 |
| **REL-O3** | **첫 실행 가드레일 (Zero-Config First-Run UX)** | • 설정 파일(`~/.config/jira/config`)이 없는 상태에서 사용자가 `jira list`, `jira tui` 등 데이터 조회 명령을 처음 실행했을 때, 단순 에러 종료 대신 대화형 프롬프트 제공:<br>  *"Jira CLI 설정이 없습니다. 지금 대화형 설정을 시작하시겠습니까? (Y/n)"* ➔ 수락 시 즉시 `jira configure` 실행<br>• 비대화형(CI/파이프라인) 환경에서는 즉시 Exit 1과 함께 가이드 출력 | P1 |
| **REL-O4** | **설정 상태 점검 (`jira config show` / `jira config test`)** | • 현재 활성화된 프로필, 설정 파일 위치(`~/.config/jira/config`), 로컬 바인딩(`.jira-profile`), 마스킹된 토큰 정보 확인<br>• Jira 서버 통신 상태 및 지연시간(Latency) 진단 기능 제공 | P1 |

---

### Track 2: 빌드 & 배포 자동화 파이프라인 (Build & Automation)

| 작업 ID | 태스크 명 | 상세 내용 및 산출물 | 우선순위 |
| :--- | :--- | :--- | :--- |
| **REL-B1** | **GoReleaser 설정** | • `.goreleaser.yaml` 작성<br>• Linux (amd64, arm64), macOS (darwin amd64, arm64), Windows (amd64) 타깃 정의<br>• `CGO_ENABLED=0` 순수 정적 빌드 보장<br>• `tar.gz`, `zip` 패키징 및 SHA-256 체크섬 파일 자동 생성 | P0 |
| **REL-B2** | **빌드 타임 메타데이터 주입** | • `main.go` / `commands_meta.go`에 `-ldflags` 지원 체계 구축<br>• Version(`v1.0.0`), Commit Hash, Build Date 주입<br>• `jira version` 실행 시 빌드 메타데이터 상세 출력 | P0 |
| **REL-B3** | **GitHub Actions CI/CD 구축** | • `.github/workflows/ci.yml`: Linux, macOS, Windows 3대 OS VM Matrix에서 `go test`, 복잡도 검사 자동화<br>• `.github/workflows/release.yml`: `v*` Git 태그 푸시 시 GoReleaser 실행 및 GitHub Release 자동 발행 | P0 |
| **REL-B4** | **원라인 설치 스크립트 작성** | • `install.sh`: OS/아키텍처 자동 감지 후 최신 GitHub Release 바이너리를 다운로드하여 `~/.local/bin`에 설치하는 원라인 스크립트 작성 (`curl -sSfL https://.../install.sh \| sh`) | P1 |
| **REL-B5** | **Homebrew Tap 레시피 준비** | • `homebrew-tap` 저장소 구축 및 GoReleaser의 `brews` 블록 연동 (`brew install joincdream/tap/jira`) | P1 |

---

### Track 3: 문서화 및 오픈소스 거버넌스 (Documentation & Governance)

| 작업 ID | 태스크 명 | 상세 내용 및 산출물 | 우선순위 |
| :--- | :--- | :--- | :--- |
| **REL-D1** | **다국어 README 구축 (영어 우선)** | • **루트 `README.md` (영어 우선)**: 글로벌 표준 영문 문서로 전면 개정 (포지셔닝, 3분 온보딩 `jira configure` & `jira init`, TUI/CLI 사용법, AI 에이전트 연동, 아키텍처 다이어그램 등)<br>• **`README.ko.md` (한국어 버전)**: 한국어 사용자를 위한 동일 구성의 공식 한글 설명서<br>• 최상단 언어 전환 뱃지/링크 구축 (`🌐 English | [한국어](README.ko.md)`) | P0 |
| **REL-D2** | **CHANGELOG.md 작성** | • [Keep a Changelog](https://keepachangelog.com/) 형식 준수<br>• v1.0.0 정식 첫 공개 기능 상세 분류 (Added, Changed, Fixed) | P0 |
| **REL-D3** | **오픈소스 거버넌스 구비** | • `LICENSE` (MIT 라이선스 재확인)<br>• `CONTRIBUTING.md` (기여 가이드, 로컬 개발 환경, 테스트 실행법)<br>• `CODE_OF_CONDUCT.md` (커뮤니티 행동 강령)<br>• `.github/ISSUE_TEMPLATE/` (버그 리포트, 기능 제안) 및 `PULL_REQUEST_TEMPLATE.md` | P1 |
| **REL-D4** | **터미널 애니메이션 데모 (VHS)** | • Charm `vhs`를 이용한 15초 분량 데모 GIF 생성<br>• `jira init` ➔ `jira list --md` ➔ `jira get` ➔ `jira move` ➔ `jira open` ➔ `jira tui` 시연 녹화 및 README 최상단 임베딩 | P1 |
| **REL-D5** | **AI Agent 연동 가이드 공식화** | • `docs/SKILL.md` 최신화<br>• Claude Code (`CLAUDE.md`), Cursor (`.cursorrules`), Antigravity (`AGENTS.md`) 연동 예시 가이드 제공 | P0 |

---

### Track 4: 품질 보증 및 사전 검증 (QA & Security Verification)

| 작업 ID | 태스크 명 | 상세 내용 및 산출물 | 우선순위 |
| :--- | :--- | :--- | :--- |
| **REL-Q1** | **전체 테스트 스위트 100% 통과** | • `go test -v ./...` 단위/통합/Mock 테스트 100% Pass 검증<br>• 레이스 컨디션 검사: `go test -race ./...`<br>• 코드 복잡도 한계 준수: `make complexity` 통과 | P0 |
| **REL-Q2** | **크로스 컴파일 및 OS별 호환성 검증** | • GitHub Actions Matrix(Linux, macOS, Windows)에서 런타임 호환성 검증<br>• `CGO_ENABLED=0` 무의존성 정적 빌드 검증 | P0 |
| **REL-Q3** | **보안 및 크레덴셜 누출 검사** | • 소스 코드 및 Git 히스토리 내 API Token, 개인 이메일, 실제 도메인 하드코딩 여부 전수 검사 (`git secrets` 또는 `gitleaks`)<br>• 설정 파일(`~/.config/jira/config`) 권한 `0600` 강제 로직 재확인 | P0 |
| **REL-Q4** | **클린 인스톨 온보딩 E2E 시나리오 검증** | • 설정이 전무한 클린 환경에서 온보딩 전 과정 리허설:<br>  1. `jira configure`로 초기 프로필 생성 및 Ping 검증<br>  2. 임의 프로젝트 디렉토리에서 `jira init`으로 로컬 바인딩<br>  3. 브랜치 컨텍스트 바인딩 및 TUI 대시보드 구동 확인 | P0 |

---

### Track 5: 릴리즈 발행 및 커뮤니티 확산 (Release Execution & GTM)

| 작업 ID | 태스크 명 | 상세 내용 및 산출물 | 우선순위 |
| :--- | :--- | :--- | :--- |
| **REL-G1** | **Git Tag 생성 및 Release 발행** | • `v1.0.0` 태그 생성 및 GitHub 원격 푸시<br>• GitHub Actions 워크플로우를 통한 GitHub Release 자동 생성<br>• 영문 릴리즈 노트(Release Notes) 게시 | P0 |
| **REL-G2** | **커뮤니티 공개 및 홍보** | • GeekNews(한국 개발자 커뮤니티) 쇼케이스 등록<br>• Reddit (`r/golang`, `r/programming`, `r/commandline`) 영문 포스팅 공유<br>• X(구 Twitter) 및 개발 블로그 기술 소개 글 게시 | P2 |

---

## 📅 3. 릴리즈 추진 타임라인 (Release Timeline)

```
[D-4] 온보딩 최적화 ───> [D-3] 빌드/CI 자동화 ───> [D-2] 문서화 & QA ───> [D-1] RC 검증 ───> [D-Day] v1.0.0 배포
- jira init 신규 구현    - GoReleaser 설정       - README (EN/KO)        - v1.0.0-rc1 태그  - v1.0.0 태그 푸시
- jira configure 개선   - GitHub Actions Matrix - CHANGELOG / VHS       - 클린 환경 E2E    - GitHub Release
- Connection Ping 검증  - ldflags 메타 주입     - 거버넌스 / 보안점검    - 최종 승인        - 커뮤니티 공지
```

### 단계별 상세 일정

#### Day 1 (D-4): 사용자 온보딩 경험 최적화 (Track 1)
- [ ] `jira configure` 입력 정규화 및 Atlassian API 연결 검증(Ping) 로직 구현
- [ ] `jira init` 명령어 구현 (대화형 로컬 디렉토리 바인딩 및 `.jira-profile` 생성)
- [ ] 설정 누락 시 대화형 자동 시작 가드레일 구현
- [ ] 온보딩 단위 및 Mock 테스트 작성

#### Day 2 (D-3): 빌드 & 배포 자동화 파이프라인 (Track 2)
- [ ] `.goreleaser.yaml` 작성 및 로컬 드라이런 검증
- [ ] `.github/workflows/ci.yml` (Linux, macOS, Windows Matrix) 및 `release.yml` 작성
- [ ] `-ldflags` 빌드 타임 메타데이터 주입 체계 연결
- [ ] 원라인 `install.sh` 스크립트 작성

#### Day 3 (D-2): 다국어 문서화 완성 및 품질/보안 검증 (Track 3, 4)
- [ ] `README.md` 영문 전면 개정 (온보딩 `jira configure` & `jira init` 강조)
- [ ] `README.ko.md` 한글 버전 동기화
- [ ] `vhs`를 통한 CLI/TUI 터미널 데모 GIF 녹화 및 추가
- [ ] `CHANGELOG.md` (v1.0.0) 작성
- [ ] 전체 단위 테스트, 린트, 코드 복잡도 통과 검증
- [ ] 시크릿/토큰 누출 검사 및 설정 권한 검증

#### Day 4 (D-1): Release Candidate(RC) 검증 및 리허설
- [ ] `v1.0.0-rc1` 태그 생성 및 내부 릴리즈 파이프라인 드라이런
- [ ] Linux / macOS / Windows 환경에서 아티팩트 다운로드 및 설치 동작 확인
- [ ] 릴리즈 체크리스트(Go/No-Go) 심의

#### Day 5 (D-Day): 정식 릴리즈 및 런치 (Track 5)
- [ ] `v1.0.0` 공식 Git Tag 푸시
- [ ] GitHub Actions 릴리즈 파이프라인 성공 확인
- [ ] GitHub Releases 릴리즈 노트 게시
- [ ] Homebrew Tap 갱신 확인
- [ ] GeekNews, Reddit, SNS 릴리즈 소식 배포

---

## 🚦 4. 릴리즈 판정 기준 (Go / No-Go Gate Criteria)

정식 릴리즈 태그(`v1.0.0`)를 발행하기 전, 다음 7대 조건을 모두 만족해야 합니다:

1. **온보딩 완결성**: `jira configure`와 `jira init`이 매끄럽게 연결되며, 설정 즉시 API 연결 검증이 성공함.
2. **테스트 100% 통과**: `go test ./...` 및 `make complexity` 실패 건수 0건.
3. **CGO-Free 정적 빌드**: 모든 배포 바이너리가 외장 C 라이브러리 링크 없이 동작함을 확인.
4. **보안 무결성**: 코드베이스 전체에 실제 Jira API 토큰, 비밀번호, 민감 URL이 존재하지 않음.
5. **다국어 문서 동기화**: `README.md`(EN)와 `README.ko.md`(KO)가 동일한 수준으로 최신 온보딩 및 CLI 기능을 충실히 설명함.
6. **크로스 플랫폼 빌드 성공**: Linux(amd64/arm64), macOS(darwin amd64/arm64), Windows(amd64) 컴파일 및 CI Matrix 통과.
7. **클린 설치 성공**: `install.sh` 및 다운로드 바이너리가 초기 환경에서 의존성 없이 즉시 구동됨.

---

## 📝 5. 변경 이력 (Document Revision History)

| 일자 | 버전 | 변경 내용 | 작성자 |
| :--- | :--- | :--- | :--- |
| 2026-10-19 | v1.0 | 릴리즈 계획서 초안 수립 | Antigravity AI Assistant & 윤상배 |
| 2026-10-19 | v1.1 | 최종 배포 버전 `v1.0.0` 확정 및 다국어 README 전략(영문 우선 `README.md`, 한글 `README.ko.md`) 반영 | Antigravity AI Assistant & 윤상배 |
| 2026-10-19 | v1.2 | **온보딩 경험 최적화(Track 1: Onboarding DX)** 신설 (`jira init` 대화형 로컬 바인딩, `jira configure` 도메인 자동완성 및 Connection Ping 검증, 첫 실행 가드레일) | Antigravity AI Assistant & 윤상배 |
