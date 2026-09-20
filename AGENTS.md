# 🤖 AGENTS.md - Jira CLI AI Agent Guidelines

본 문서는 Antigravity CLI, Cursor, Claude Code 등 생성형 AI 에이전트가 Jira CLI(`jira`) 코드베이스를 탐색, 유지보수, 리팩토링 및 신규 개발할 때 **반드시 준수해야 하는 최우선 행동 지침**입니다.

---

## 🧠 1. OKF 지식 베이스 선행 참조 원칙 (Knowledge-First Protocol)

코드를 임의로 추측하여 수정하지 마십시오. 작업 착수 전 반드시 [knowledge/index.md](knowledge/index.md)를 통해 프로젝트 지식을 먼저 획득해야 합니다.

### 작업 유형별 필수 참조 OKF 문서

| 작업 유형 | 필수 참조 문서 (`knowledge/*.md`) | 핵심 확인 내용 |
| :--- | :--- | :--- |
| **전체 구조 및 신규 기능 설계** | [knowledge/architecture.md](knowledge/architecture.md) | 3-Tier 구조, Zero-Dependency 원칙, Command 인터페이스 및 DI 패턴 |
| **설정, 인증, 프로필 관련 작업** | [knowledge/configuration.md](knowledge/configuration.md) | INI 멀티 프로필, 4단계 프로필 결정 우선순위, `.jira-profile` 자동 탐색 |
| **CLI 명령어 추가 및 플래그 수정** | [knowledge/commands-and-routing.md](knowledge/commands-and-routing.md) | `parseOutputFormat`, `--md`/`--json` 포맷터, 종료 코드 규격 |
| **이슈 설명, 코멘트 서식 처리** | [knowledge/adf-engine.md](knowledge/adf-engine.md) | Markdown ↔ ADF 양방향 파서, 블록/인라인 노드 매핑 규격 |
| **Jira API 요청/응답 처리** | [knowledge/data-models.md](knowledge/data-models.md) | Go 구조체와 Jira Cloud REST API v3 DTO 매핑, Null 처리 규칙 |
| **라벨(Tags) 검증 및 추가** | [knowledge/labels-catalog.md](knowledge/labels-catalog.md) | 표준 라벨 카탈로그 탐색 경로, 대소문자 정규화, `--force-labels` |

---

## 🚫 2. 엄격한 엔지니어링 제약 조건 (Strict Constraints)

1. **Zero Third-Party Dependencies (외부 의존성 절대 금지)**:
   - `go.mod`에 외부 서드파티 라이브러리를 추가하지 마십시오.
   - 모든 기능은 Go 표준 라이브러리(`net/http`, `encoding/json`, `flag`, `text/tabwriter` 등)만으로 구현해야 합니다.
2. **AI Agent 친화적 출력 표준 준수**:
   - 요약문(Summary)이나 긴 문자열을 임의로 바이트 슬라이싱하지 마십시오 (UTF-8 한글 깨짐 방지).
   - 모든 데이터 조회성 명령어는 인간용 터미널 뷰 외에 `--md` (마크다운) 및 `--json` (구조화된 JSON) 출력을 반드시 지원해야 합니다.
   - 성공 시 `stdout`(Exit code 0), 실패 시 `stderr`(Exit code 1) 규칙을 엄수하십시오.
3. **보안 및 자격 증명 보호**:
   - 소스 코드나 git 저장소에 API Token, 비밀번호를 하드코딩하거나 커밋하지 마십시오.
   - 모든 민감 정보는 `~/.config/jira/config` (권한 `0600`)에서만 읽어옵니다.

---

## 🔄 3. 작업 완료 후 지식 베이스 동기화 의무 (Sync Protocol)

코드 변경이나 기능 확장이 발생한 경우, 지식의 파편화를 막기 위해 반드시 다음 동기화 절차를 완료해야 합니다:

1. **대응되는 OKF 문서 갱신**:
   - 변경된 로직이나 플래그, API DTO를 `knowledge/*.md` 문서에 즉시 반영하고 상단 YAML Frontmatter의 `timestamp`를 갱신합니다.
2. **변경 로그 기록 ([knowledge/log.md](knowledge/log.md))**:
   - 변경 일시, 작업 내용 및 검증 테스트 결과를 `knowledge/log.md`에 추가합니다.
3. **SKILL 문서 동기화 ([~/.config/jira/SKILL.md](file:///home/yundream/.config/jira/SKILL.md))**:
   - CLI 인터페이스 변경 시 에이전트 스킬 문서도 함께 최신 상태로 유지합니다.

---

## 🧪 4. 검증 및 테스트 원칙 (Verification)

코드를 수정한 후에는 반드시 전체 테스트 스위트를 실행하여 회귀 버그가 없음을 증명해야 합니다:

```bash
# 단위 및 E2E Mock 테스트 실행 (100% Pass 필수)
go test -v ./...

# 바이너리 빌드 및 설치 검증
make install
```
