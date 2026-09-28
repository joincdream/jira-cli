---
name: jira-cli
description: "Zero-dependency lightweight Jira Cloud CLI for managing Jira issues, multi-profile tenant configurations, and workflow automations."
---

# 🛠️ Jira CLI Agent Skill (`jira`)

Jira Cloud REST API v3 기반의 초경량 Zero-Dependency CLI 도구인 `jira`의 사용 지침 및 명령어 규격입니다.
AI Agent는 본 가이드를 준수하여 이슈 조회, 생성, 수정, 워크플로 전이 및 멀티 프로젝트/테넌트 업무를 수행합니다.

---

## 📌 1. 핵심 원칙 (Core Protocols for AI Agents)

1. **머신 리더블 포맷 우선 (`--md` / `--json`)**:
   - 데이터 분석, 요약 및 LLM 프롬프트 주입 시에는 테이블 뷰 대신 `--md` 또는 `--json` 플래그를 사용하십시오.
   - 예: `jira get JC-10 --md`, `jira list --all --due '<=today' --json`
2. **이슈 요약문(Summary) 자르기 금지**:
   - `jira list` 및 `jira get`은 50자 강제 슬라이싱 없이 전체 텍스트를 출력하므로 UTF-8 한글 깨짐 없이 문맥을 그대로 유지합니다.
3. **Fail-Fast 가드레일 준수**:
   - `jira create`: 제목 공백 제외 최소 5자 이상 필수, 본문 설명(`-d`) 기본 필수 (설명 없이 생성 시 `--allow-empty-desc`).
   - 마감일(`--due`): `YYYY-MM-DD` 또는 상대 표현(`today`, `<=today`, `tomorrow`, `week`) 사전 검증.
4. **결과 코드 규격**:
   - 성공 시 `stdout` (Exit code 0), 실패 시 `stderr` (Exit code 1).

---

## 🧭 2. 글로벌 플래그 (Global Flags)

모든 서브커맨드 앞이나 뒤에 지정할 수 있습니다:

| 플래그 | 설명 | 예시 |
| :--- | :--- | :--- |
| `--profile <name>` | 특정 Jira 프로필(테넌트/스페이스) 선택 | `jira --profile cloit list` |
| `--lang <ko\|en>` | CLI 출력 언어 강제 지정 | `jira --lang en list` |

---

## 📋 3. 주요 서브커맨드 레퍼런스

### 1) 이슈 목록 조회 (`jira list`)
```bash
# 기본: 현재 프로젝트의 열린(미완료) 이슈 목록 조회
jira list

# 관리자 뷰: 전체 프로젝트 통틀어 오늘까지 해야 할 일 (연체 포함)
jira list --all --due '<=today'

# 전체 프로젝트에서 내게 할당된 열린 이슈
jira list --all --assignee me

# 완료된 이슈까지 전체 조회
jira list --status all

# 마크다운 포맷으로 출력 (AI 맥락 주입용)
jira list --all --due '<=today' --md

# 직접 JQL 쿼리 실행
jira list "labels = AI and status = '진행 중'" --json
```

**지원 필터 플래그**:
* `--all`, `-A`: 활성 프로필의 `project_key` 제한을 해제하고 전체 테넌트 스코프로 확장.
* `--due <expr>`: `today`, `<=today`, `tomorrow`, `<=tomorrow`, `week`, `YYYY-MM-DD`, `<=YYYY-MM-DD`.
* `--status <status>`: `open` (기본값: `statusCategory != Done`), `done`, `all`, 또는 특정 상태명.
* `--assignee <user>`, `-a`: `me` (`currentUser()`), `none` (`is EMPTY`), 특정 사용자 이메일/ID.

---

### 2) 이슈 상세 조회 (`jira get`)
```bash
# 단일 이슈의 상세 메타데이터, 설명, 하위작업, 코멘트 전체 맥락을 마크다운으로 조회
jira get JC-10 --md

# 구조화된 JSON 데이터로 조회
jira get JC-10 --json
```

---

### 3) 이슈 생성 (`jira create`)
```bash
# 표준 이슈 생성 (제목 최소 5자, 설명 필수)
jira create "CLI 인자 파싱 표준화" -d "상세 설명 본문 내용" -l "Architecture,Development"

# 특정 프로젝트 및 마감일 지정
jira create "신규 기능 기획" -d "기획서 초안 작성" -p BE --due 2026-10-31

# 설명 없이 간이 티켓 생성
jira create "간단한 버그 수정 건" --allow-empty-desc

# 하위 작업(Subtask) 생성
jira create "DB 마이그레이션 스크립트 작성" -d "상세 내용" -t "Subtask" --parent JC-10
```

---

### 4) 이슈 수정 (`jira edit`)
```bash
# 마감일 및 라벨 수정
jira edit JC-10 --due 2026-10-15 -l "Development,QA"

# 마감일 제거 (none)
jira edit JC-10 --due none

# 제목 및 본문 수정
jira edit JC-10 -s "수정된 요약 제목" -d "업데이트된 상세 설명"
```

---

### 5) 워크플로 상태 전이 (`jira move` / `jira transitions`)
```bash
# 변경 가능한 상태 목록 확인
jira transitions JC-10

# 상태 전이 실행 (이름 기준)
jira move JC-10 "진행 중"
jira move JC-10 "완료"
```

---

### 6) 코멘트 추가 (`jira comment`)
```bash
# 진행 상황 코멘트 기록
jira comment JC-10 "단위 테스트 100% 통과 및 바이너리 배포 완료하였습니다."
```

---

### 7) 프로필 및 설정 관리 (`jira configure`)
```bash
# 프로필 목록 및 상속 관계 조회
jira configure list

# 대화형 프로필 설정 (기존 프로필 상속 지원)
jira configure --profile sub-project
```

---

## ⚙️ 4. 멀티 프로필 및 상속 아키텍처 (`~/.config/jira/config`)

동일 테넌트 내 멀티 프로젝트 운영 시 API 토큰 중복 없이 `[default]`로부터 자동 상속받습니다:

```ini
# 공통 테넌트 인증 키 1회 기재
[default]
instance_url = https://joincdream.atlassian.net
email = joinc.dream@gmail.com
api_token = ATATT3xFfGF0...
project_key = KAN

# 서브 프로젝트: [default] 자동 상속 (project_key만 정의)
[jira-cli]
project_key = JC

# 외부 테넌트 기반 명시적 상속 (source_profile)
[client-mobile]
source_profile = client-corp
project_key = MOB
```

* **로컬 디렉토리 바인딩 (`.jira-profile`)**:
  저장소 루트에 `.jira-profile` 파일을 두고 `jira-cli`라고 적어두면, 해당 폴더에서 실행되는 모든 명령어에 해당 프로필이 자동 적용됩니다.
