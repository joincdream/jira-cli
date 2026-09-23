# 🌐 07. 오픈소스 배포 검토 및 전략 보고서 (Open Source Strategy & Product Review)

> **Jira CLI의 가치 제안(Value Proposition), 페르소나별 페인포인트 분석, 오픈소스 포지셔닝 및 개선 로드맵**  
> 본 문서는 `jira` CLI를 오픈소스 소프트웨어(OSS)로 공개 배포할 때 고려해야 할 핵심 가치, 경쟁 우위, 타깃 페르소나별 비즈니스 임팩트와 미해결 과제, 그리고 구체적인 제품 개선 로드맵을 종합적으로 정리한 전략 보고서입니다.

---

## 📌 1. 개요 및 검토 배경 (Executive Summary)

Atlassian Jira는 전 세계 수많은 기업의 표준 협업 및 프로젝트 관리 도구이지만, 웹 UI의 높은 복잡도와 느린 반응성(로딩 지연 3~5초)으로 인해 터미널 중심의 개발자와 자동화 파이프라인에서 지속적인 컨텍스트 스위칭(Context Switching) 피로를 유발합니다.

본 프로젝트(`jira`)는 **Go 표준 라이브러리 및 준표준 기반의 무의존성(Zero-Dependency)**, **내장 ADF 변환 엔진**, 그리고 **AI 에이전트(LLM) 친화적 I/O**를 핵심 무기로 삼아 개발되었습니다. 본 문서는 현재 기능과 계획된 개선 작업([JC-3]~[JC-9])이 이루어졌을 때 타깃 고객 페르소나에게 어떤 비즈니스 가치를 제공하고, 어떤 한계를 가지는지를 명확히 규명합니다.

---

## 💡 2. Jira CLI의 쓸모 및 핵심 가치 (Value Proposition)

### ① 개발자의 터미널/IDE 컨텍스트 스위칭 원천 차단
- 개발자는 코드를 작성하다가 무거운 웹 브라우저 탭을 열어 Jira에 접속하고, 로그인 세션을 갱신하며, 수많은 위젯이 렌더링되기를 기다릴 필요가 없습니다.
- 터미널이나 IDE 내장 터미널에서 즉시 이슈를 확인하고(`jira get`), 작업을 시작하며(`jira move "진행 중"`), 코멘트를 작성(`jira comment`)할 수 있습니다.

### ② AI Coding Agent (LLM)를 위한 네이티브 인터페이스 (**★ 최신 핵심 가치**)
- 2026년 기준 개발 환경은 Cursor, Claude Code, Antigravity, Copilot Workspace 등 CLI 환경에서 직접 쉘 명령을 실행하는 자율형 코딩 에이전트가 주도하고 있습니다.
- 이들 AI 에이전트에게 복잡한 브라우저 인터랙션 대신, 터미널 명령을 통해 마크다운 형식으로 티켓을 제공하고(`jira get KAN-123 --md`) 결과를 Jira에 자동 기록하는 **최적의 Agentic Tool** 역할을 수행합니다.

### ③ CI/CD 및 자동화 파이프라인의 초경량 연동
- GitHub Actions, GitLab CI 등 배포 파이프라인에서 릴리스 생성 시 배포된 Jira 티켓들을 일괄 완료 처리하거나 빌드 링크/커밋 로그를 자동으로 코멘트로 남기는 작업을 별도의 복잡한 플러그인 없이 단일 바이너리로 구현할 수 있습니다.

---

## 🏆 3. 현재 코드베이스의 강점 (Strengths & Differentiators)

기존의 유사 오픈소스 프로젝트(예: `ankitpokhrel/jira-cli`, `go-jira/jira`)와 대비되는 독보적인 강점은 다음과 같습니다:

| 강점 영역 | 상세 내용 | 비교 우위 |
| :--- | :--- | :--- |
| **Zero Third-Party Dependencies** | 외부 서드파티 라이브러리 전무. Go 표준 및 준표준(`golang.org/x/text`)만으로 구현 | 공급망 보안(CVE) 원천 차단, 초고속 컴파일, 가벼운 단일 바이너리 배포 |
| **자체 내장 ADF 변환 엔진** | Jira Cloud REST API v3의 난제인 복잡한 JSON 트리(ADF)를 Markdown과 양방향 자동 변환 | 사용자와 AI는 단순 마크다운만 작성하면 되며, 풍부한 서식(제목, 볼드, 코드블록 등) 완벽 지원 |
| **AI 친화적 출력 표준** | 테이블(`table`), 마크다운(`--md`), 구조화된 JSON(`--json`) 기본 지원 | LLM 프롬프트 주입 및 `jq` 파이프라인 연계에 극도로 유리 |
| **AWS CLI 스타일 멀티 프로필** | `~/.config/jira/config` 단일 INI 파일에서 여러 계정/도메인 격리 관리 및 디렉토리별 `.jira-profile` 자동 전환 지원 | 개인/회사/고객사 등 다중 Jira 환경을 다루는 프리랜서·SI·엔터프라이즈 개발자 필수 기능 |
| **표준 라벨 카탈로그 검증** | 프로젝트별 `labels.json`을 탐색하여 대소문자 정규화 및 오타 방지 | 팀 단위 라벨 오염(예: `frontend`, `Front-End`, `front_end`) 방지 |

---

## 🔍 4. 시각화 방식 딥다이브: TUI vs Local Serve vs Hand-off

CLI 도구에서 "시각적 결과 확인(Visualization)"을 어떻게 다룰 것인가에 대한 심층 검토 결과입니다.

### 4.1 터미널 TUI 칸반보드의 본질적 한계 (Why NOT TUI?)
일부 CLI 툴(`ankitpokhrel/jira-cli` 등)은 터미널 안에서 칸반보드를 직접 구현하지만, 실무 환경에서는 심각한 한계에 부딪힙니다:
1. **화면 해상도 및 정보 밀도의 한계**:
   - 칸반보드는 4~6개의 열(Column)을 한눈에 조망(Overview)하는 것이 목적인데, 좁은 터미널 폭(80~120 컬럼)에서는 한 열당 15~20자밖에 표시되지 않아 제목이 잘리고 스크롤 지옥에 빠집니다.
2. **조작성(UX) 역주행**:
   - 마우스 드래그 앤 드롭 대신 복잡한 방향키/단축키를 더듬거리며 옮기는 것은 `jira move KAN-123 Done` 한 줄 타이핑보다 훨씬 느립니다.
3. **유닉스 철학 위배 및 AI 에이전트 차단**:
   - TUI의 가상 화면 버퍼는 파이프라인(`|`), 리다이렉션(`>`), 복사-붙여넣기를 방해하며, AI 에이전트(LLM)에게는 해석 불가능한 ANSI 노이즈(블랙박스)가 됩니다.
4. **유지보수 비용 폭증**:
   - 터미널 리사이징, CJK 2바이트 문자 깨짐 등 비즈니스 로직 외적인 UI 버그 대응에 개발 리소스가 낭비되며 무의존성 철학이 파괴됩니다.

### 4.2 로컬 웹 서버(`serve`) 방식의 모순 (Why NOT `jira serve`?)
`pprof`, `allure`처럼 `jira serve` 명령으로 로컬 웹 서버를 띄워 칸반을 보여주는 아이디어는 직관적으로 매력적이나, Jira 생태계에서는 치명적인 모순이 있습니다:
- **`pprof`/`allure`와의 결정적 차이**: 이 도구들은 **원래 웹 UI가 아예 없는 도구**이므로 로컬 웹 서버가 유일한 시각화 수단입니다.
- **Jira의 현실**: Jira는 이미 수천억 원이 투자된 **세계 최고 수준의 공식 웹 애플리케이션(`*.atlassian.net`)이 이미 존재**합니다.
- **결과적 결함**: 로컬 웹을 만들어도 공식 웹의 필터, 첨부파일, 멘션, 서드파티 플러그인 기능을 따라갈 수 없으므로 사용자에게는 *"기능이 반쪽짜리인 어설픈 복제본(Inferior Clone)"*이자 불필요한 기능 중복(Over-engineering)으로 전락합니다.

### 4.3 최선의 아키텍처: "Headless CLI + Hand-off (브라우저 토스)"
가장 현명한 해법은 GitHub CLI(`gh browse`)의 성공 방식을 따르는 것입니다:
- **CLI의 본질**: 터미널 안에서 0.1초 만에 빠른 처리, 스크립트 자동화, AI 에이전트 협업 전담.
- **시각적 조망 필요 시**: 단 20줄의 OS 브라우저 런처(`jira open`, [JC-3])를 통해 이미 완벽한 **공식 Jira 웹으로 즉시 토스(Hand-off)**.

---

## 👥 5. 타깃 페르소나 심층 분석 및 페인포인트 매트릭스 (Persona & Value Matrix)

현재 기능과 계획된 개선 과제([JC-3]~[JC-9])를 바탕으로 주요 타깃 페르소나의 페인포인트와 해결책, 그리고 본 도구가 해결하지 못하는 명확한 한계를 분석합니다.

### 5.1 페르소나 A: AI-First 개발자 (Agentic Developer)
> *"코딩은 Cursor와 Claude Code가 하고, 나는 검토만 한다. Jira 티켓도 에이전트가 알아서 읽고 처리했으면 좋겠다."*

* **배경**: IDE 및 CLI 터미널 내에서 자율형 AI 에이전트를 페어 프로그래머로 상시 활용하는 최신 소프트웨어 엔지니어.
* **주요 페인포인트 (Pain Points)**:
  1. **웹 UI의 비접근성**: 브라우저 기반 Jira는 자바스크립트 렌더링과 인증 세션 때문에 AI 에이전트가 브라우저를 통해 직접 읽고 조작하기 극도로 불안정함.
  2. **ADF JSON의 토큰 낭비**: Jira API 원본 응답(ADF 구조)은 JSON 계층이 지나치게 깊어, LLM 컨텍스트에 주입 시 수천 개의 토큰 낭비와 환각(Hallucination)을 유발함.
  3. **수동 컨텍스트 전달 피로**: 웹에서 티켓 요구사항을 복사 ➔ IDE에 붙여넣기 ➔ 구현 ➔ 다시 브라우저를 열어 "완료"로 이동하고 커밋 링크를 남기는 번거로운 수작업.
* **Jira CLI의 해결 방식 (How It Solves)**:
  - **`jira get <KEY> --md`**: Jira Cloud v3의 복잡한 ADF 구조를 깨끗하고 완벽한 마크다운으로 자동 변환하여 프롬프트에 직결 (토큰 비용 70~80% 절감).
  - **에이전트 자율 피드백 루프**: 에이전트가 터미널 툴로서 `jira comment <KEY> "구현 요약"` 및 `jira move <KEY> "완료"`를 실행하여 작업 완결.
  - **영문 표준화 및 i18n ([JC-4])**: 에이전트가 결정론적으로 파싱 가능한 일관된 표준 출력 및 에러 코드 제공.
* **해결하지 못하는 페인포인트 (Unsolved)**:
  - 기획서 내 대용량 이미지, Figma 링크, PDF 첨부파일의 자체 OCR/멀티모달 직접 분석 불가 (티켓 내 URL 텍스트만 전달).
  - Jira의 다단계 승인(Approvals) 및 전자 서명 결재 워크플로우의 자율 처리 불가.

---

### 5.2 페르소나 B: 터미널 중심의 실무 엔지니어 (Terminal-centric Engineer)
> *"마우스 만지는 시간도 아깝다. Vim/Tmux 터미널에서 작업 흐름(Flow)을 깨지 않고 티켓을 빠르게 처리하고 싶다."*

* **배경**: 백엔드, 시스템, 인프라 개발자로 하루 종일 Neovim, Tmux, Git CLI 환경에서 생활하는 키보드 중심(Keyboard-driven) 개발자.
* **주요 페인포인트 (Pain Points)**:
  1. **컨텍스트 스위칭 피로**: 단순 상태 변경("In Progress", "Done")이나 코멘트 한 줄 작성을 위해 3~5초씩 걸리는 무거운 Jira 웹을 띄워야 함.
  2. **팀 단위 라벨 오염**: 개발자마다 대소문자나 오타(예: `frontend`, `Front-End`, `front_end`)를 입력하여 필터 검색 누락 및 데이터 오염 발생.
  3. **브랜치-티켓 키 불일치**: `feature/JC-12-auth` 브랜치에서 작업하다가도 티켓 키 번호를 잊어 매번 확인해야 함.
* **Jira CLI의 해결 방식 (How It Solves)**:
  - **0.1초 원라이너**: 터미널을 벗어나지 않고 즉시 상태 전이(`jira move`) 및 조회(`jira list`).
  - **Git 브랜치 컨텍스트 자동 인식 ([JC-5])**: 현재 브랜치명에서 티켓 키를 자동 파싱하여 `jira move "In Progress"`만 입력해도 현재 티켓 자동 타깃팅.
  - **표준 라벨 카탈로그 검증**: `labels.json` 기반의 대소문자 자동 정규화 및 오타 방지로 데이터 일관성 유지.
  - **스마트 웹 핸드오프 ([JC-3])**: 전체 칸반 조망이 필요할 때만 `jira open`으로 공식 웹에 0.1초 만에 스마트 토스.
* **해결하지 못하는 페인포인트 (Unsolved)**:
  - 마우스 드래그 앤 드롭을 통한 백로그 순위(Rank) 미세 재배치 (공식 웹 사용 필수).
  - WYSIWYG 리치 텍스트 테이블/색상 편집 (터미널에서는 마크다운 표 수준으로만 표현 가능).

---

### 5.3 페르소나 C: 데브옵스 / SRE / 플랫폼 엔지니어 (DevOps & Platform Engineer)
> *"배포 파이프라인(CI/CD) 안에서 가볍고 안전하게 Jira 이슈를 자동 업데이트하고 싶다. 의존성 지옥은 질색이다."*

* **배경**: GitHub Actions, GitLab CI 등 CI/CD 파이프라인 구축 및 릴리스 자동화를 전담하는 플랫폼/인프라 엔지니어.
* **주요 페인포인트 (Pain Points)**:
  1. **파이프라인 종속성 및 보안 부담**: CI 러너 환경에서 Jira 연동을 위해 무거운 서드파티 도구, Node.js/Python 런타임을 설치해야 하므로 빌드 시간이 늘어나고 공급망 보안(CVE) 취약점 위험 발생.
  2. **복잡한 REST API 연동**: Atlassian Cloud REST API v3의 ADF 생성 및 인증 처리가 까다로워 `curl` 쉘 스크립트 작성 및 유지보수에 과도한 리소스 소모.
* **Jira CLI의 해결 방식 (How It Solves)**:
  - **Zero Third-Party Dependencies**: 단일 정적 바이너리 구조로 CI 환경에서 `curl` 다운로드 후 0.1초 만에 실행 가능하며 공급망 공격 위험 0%.
  - **기계 판독형 I/O**: `--json` 출력과 엄격한 Exit Code(0: 성공, 1: 실패)를 제공하여 `jq` 및 쉘 스크립트 파이프라인과 완벽 연동.
  - **릴리스 자동화**: 배포 완료 시 릴리스 노트 코멘트 등록 및 연관 티켓 일괄 상태 전이 자동화.
* **해결하지 못하는 페인포인트 (Unsolved)**:
  - 수백~수천 개 티켓의 대규모 비동기 배치(Bulk Operation) 처리 큐 (스크립트 루프 처리 필요, Atlassian API Rate Limit 유의).
  - 웹훅(Webhook) 수신을 통한 이벤트 기반 양방향 상시 트리거 데몬 기능 부재 (순수 CLI 클라이언트임).

---

### 5.4 페르소나 D: 다중 프로젝트/고객사를 관리하는 테크리드 & 컨설턴트 (Multi-Tenant Tech Lead)
> *"고객사 A, 고객사 B, 사내 프로젝트를 동시에 넘나든다. 테넌트와 프로젝트가 바뀔 때마다 겪는 설정 지옥을 끝내고 싶다."*

* **배경**: SI 구축, 클라우드 MSP, 외주 컨설팅을 수행하며 복수의 Atlassian 계정 및 복수의 Jira 스페이스를 동시에 다루는 리드 엔지니어.
* **주요 페인포인트 (Pain Points)**:
  1. **계정 전환 피로**: 테넌트가 다른 고객사 사이트를 오갈 때마다 브라우저 프로필을 전환하거나 시크릿 창을 새로 열어야 함.
  2. **설정 및 토큰 복제 지옥**: 동일한 테넌트 내에서 프로젝트만 다를 때도 URL, 이메일, API 토큰을 모든 프로필마다 중복 복사해야 하고, 토큰 만료 시 모든 프로필을 일일이 수정해야 함.
* **Jira CLI의 해결 방식 (How It Solves)**:
  - **AWS CLI 스타일 프로필 상속 ([JC-9])**: 베이스 테넌트 프로필에서 토큰을 1곳만 관리하고, 서브 프로젝트는 `project_key = JC` 단 2줄로 설정 완료 (토큰 갱신 1회로 동기화).
  - **디렉토리 기반 자동 프로필 전환**: 프로젝트 저장소 루트의 `.jira-profile`을 감지하여 해당 디렉토리로 이동(`cd`)하면 계정 컨텍스트가 자동으로 전환.
* **해결하지 못하는 페인포인트 (Unsolved)**:
  - 관리자/PM 레벨의 대형 기능: 로드맵/간트 차트, 포트폴리오 타임라인, 스프린트 생성 및 마감 회고, 권한 스키마 설정 등 (공식 웹 필수).

---

### 5.5 페르소나별 가치 & 한계 종합 요약 매트릭스 (Summary Matrix)

| 구분 | 페르소나 A (AI-First 개발자) | 페르소나 B (터미널 개발자) | 페르소나 C (데브옵스/SRE) | 페르소나 D (다중 테넌트 리드) |
| :--- | :--- | :--- | :--- | :--- |
| **핵심 가치** | LLM 프롬프트 직결 및 자율 액션 루프 | 마우스 프리, 0.1초 원라이너 워크플로우 | 무의존성 경량 바이너리, 안전한 CI 자동화 | 토큰 단일 지점 관리 및 자동 컨텍스트 전환 |
| **핵심 기능** | `--md` 포맷터, `comment`, `move` | `jira open`([JC-3]), Git 브랜치 감지([JC-5]) | Zero-Dep, `--json`, Exit Code 규격 | 프로필 상속([JC-9]), `.jira-profile` |
| **해결된 고통** | ADF 토큰 낭비, 비정형 HTML 스크래핑 | 웹 로딩 지연(3~5초), 라벨 오타 오염 | 무거운 의존성 설치, 복잡한 curl 스크립트 | 토큰 복붙 지옥, 다중 계정 전환 피로 |
| **해결 불가 영역** | 멀티모달(이미지/PDF) OCR, 승인 워크플로우 | 백로그 랭크 드래그앤드롭, WYSIWYG 편집 | 대량 비동기 벌크 큐, 상시 웹훅 리스너 | 스프린트/권한 관리, 로드맵/간트차트 |

---

## ⚠️ 6. 현실적 제약 및 제품 경계 (Boundaries & Remaining Limitations)

1. **Jira 복잡한 워크플로우(Transition Screens) 대응**:
   - 상태 전이 시 "해결책(Resolution)", "수정 버전(Fix Version)" 등 필수 화면 필드를 요구하는 엔터프라이즈 프로젝트에서 `400 Bad Request` 에러가 발생할 수 있으며, 친절한 에러 파싱 및 웹 완결 가이드([JC-6])를 제공해야 합니다.
2. **커스텀 필드(Custom Fields) 지원 제약**:
   - 기업별 커스텀 필드(Story Points, Sprint 등)에 대한 매핑 유연성이 아직 부족하여 점진적 설정 매핑 확장이 필요합니다.

---

## 🎯 7. 오픈소스 포지셔닝 및 틈새 시장 전략 (Market Positioning)

```text
❌ 피해야 할 포지셔닝: "웹 브라우저를 터미널 안에서 흉내 내는 화려한 TUI 대시보드"
⭕ 지향해야 할 포지셔닝: "The Zero-Dependency, AI-Native & Headless Jira CLI with Seamless Web Hand-off"
```

### 핵심 슬로건
> **"터미널에서는 0.1초 만에 가볍게, 복잡한 시각화는 공식 웹으로 스마트하게 토스한다."**

---

## 🛠️ 8. 기능 개선 및 제품화 로드맵 (Roadmap)

오픈소스 릴리스 전후로 적용할 구체적인 개선 로드맵입니다:

### Phase 1. 오픈소스 공개 필수 및 UX 극대화 (Sprint 1)
- [x] **[JC-8] CLI 인자/플래그 파싱 표준화 및 티켓 생성 유효성 검증 가드레일 강화 (P0 - 완료)**:
  - `--help` 및 `-h` 플래그 입력 시 API 호출 방지 및 즉시 사용법(Usage) 출력 후 종료(Exit 0).
  - 옵션과 위치 인자의 순서 독립성(Order Independence) 보장 및 Double Dash(`--`) 지원.
  - 티켓 생성 사전 검증(Client-side Guardrail): 제목 5자 이상/플래그형 문자열 차단, 설명(`-d`) 기본 필수화(`--allow-empty-desc` 예외 지원), 마감일 날짜 포맷 로컬 검증.
- [ ] **[JC-3] 웹 핸드오프 커맨드 추가 (`jira open` / `jira browse`)**:
  - `jira open <KEY>`: 기본 웹 브라우저에서 해당 티켓 URL 즉시 열기 (`https://<domain>/browse/<KEY>`).
  - `jira open`: 현재 프로젝트의 활성 칸반/스크럼 보드 즉시 열기.
  - OS별 표준 명령(`xdg-open` on Linux, `open` on macOS, `rundll32` on Windows)을 활용한 Zero-Dependency 구현.
- [ ] **[JC-4] Go 공식 준표준 패키지(golang.org/x/text) 기반 i18n 다국어 시스템 구축**:
  - Go 코어 팀의 공식 준표준 라이브러리(`golang.org/x/text/message`, `golang.org/x/text/language`)를 채택하여 표준적이고 안정적인 다국어 환경 구축.
  - 글로벌 오픈소스 표준 언어인 영어(`en`)를 기본값으로 설정하고 한국어(`ko`) 완벽 지원.
  - 로케일 매칭: CLI 플래그(`--lang`) > 환경변수(`JIRA_LANG`) > 프로필 설정(`language`) > OS 로케일(`LANG`) > 기본값(`en`).
- [ ] **[JC-5] Git 브랜치 컨텍스트 기반 이슈 키 자동 감지**:
  - 현재 git 브랜치(`feature/JC-12-auth`)에서 티켓 키(`JC-12`)를 정규식으로 파싱하여, 키 인수를 생략해도 현재 티켓을 자동 조작 (`jira get`, `jira move "In Progress"`, `jira open`).
- [ ] **[JC-6] Jira 상태 전이(Transition) 화면 에러 가이드 강화**:
  - 상태 전이 실패 시 반환되는 Jira API 에러를 파싱하여 "이 상태 변경에는 [해결책(Resolution)] 필드가 필수입니다"와 같이 명확한 실패 사유 출력 및 웹 완결 가이드(`jira open`) 제공.
- [ ] **[JC-9] AWS CLI 스타일 프로필 상속(source_profile 및 Default Fallback) 지원**:
  - `[default]` 프로필 자동 상속: `instance_url`, `email`, `api_token` 누락 시 기본 프로필에서 자동 주입.
  - 명시적 상속: `source_profile = <name>` 지원으로 제2의 테넌트(고객사/외주 계정)의 멀티 프로젝트 공유 지원.
  - API 토큰 단일 지점 관리 및 `jira configure` 상속 설정 연동.
- [ ] **[JC-7] 오픈소스 배포 자동화 파이프라인 및 문서화 구축**:
  - GoReleaser + GitHub Actions 멀티 OS 바이너리 릴리스, Homebrew Tap, `vhs` 데모 GIF, AI Agent(`SKILL.md`) 연동 가이드.

### Phase 2. 엔터프라이즈 확장성 확보
- [ ] **설정 기반 커스텀 필드 매핑**:
  - `~/.config/jira/config`에 `epic_field = customfield_10014`, `story_points = customfield_10020` 형태로 선언하여 CLI 옵션으로 지정 가능하도록 지원.
- [ ] **대화형 전이 모드 (선택적 프롬프트)**:
  - 전이 시 필수 필드가 누락되었을 때 터미널에서 간단한 값 입력을 요청하는 표준 라이브러리 기반 미니멀 프롬프트.

---

## 📦 9. 배포 및 오픈소스 생태계 확산 전략 (Go-to-Market)

1. **설치 편의성 극대화 (Zero-Friction Install)**:
   - **GoReleaser + GitHub Actions**: 태그 푸시 시 Linux(amd64/arm64), macOS(Intel/Apple Silicon), Windows 정적 바이너리 자동 릴리스.
   - **Homebrew Tap**: `brew install <user>/tap/jira` 지원.
   - **One-line Shell Installer**: `curl -sSfL .../install.sh | sh` 제공.
2. **시각적 데모 (Terminal GIF)**:
   - `vhs` 도구를 활용하여 터미널에서 `jira list --md`, `jira create`, `jira move`, `jira open`이 부드럽게 이어지는 10~15초 분량의 데모 애니메이션을 README 최상단에 배치.
3. **AI Coding Agent 연동 가이드 및 Skill 템플릿 제공**:
   - Claude Code, Antigravity, Cursor에서 본 CLI를 서브프로세스 툴로 등록하는 설정 파일 예시(`SKILL.md` 또는 Agent instructions)를 제공하여 바이럴 유도.
4. **오픈소스 거버넌스 구비**:
   - `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, 이슈 템플릿 구비.
