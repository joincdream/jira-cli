# Claude Code Project Guidelines - Jira Integration

## Jira Tooling Commands
- 원샷 업무 파악: `jira list --assignee me --status open --detail --md` (진행 중 작업 우선 & 설명/코멘트 1회 획득)
- 특정 티켓 맥락: `jira get <KEY> --md` (토큰 절약 마크다운)
- 웹 브라우저 열기: `jira open <KEY>` (또는 `jira open` 프로젝트 보드)
- 작업 피드백 등록: `jira comment <KEY> "<MESSAGE>"`
- 상태 전이: `jira move <KEY> "<STATUS>"` (작업 착수 시 '진행 중' 자동 전이, 완료 시 사용자 지시 대기)

## Output Standard
- Jira 티켓에 대해 사용자에게 답변할 때는 불필요한 전체 덤프를 피하고, **Agent Insight Block** 규격(현황, AI 추론, 추천 액션)으로 일관되게 브리핑할 것.

## Task Lifecycle & Human-in-the-Loop Rules
- **작업 착수 시**: 사용자가 티켓 작업이나 계획 수립을 지시하면 즉시 `jira move <KEY> "진행 중"`으로 전이할 것.
- **작업 마감 시**: 단위 테스트를 통과했다고 해서 자의적으로 티켓을 완료(`jira move "완료"`)하지 말 것. 사용자가 명시적으로 지시할 때만 완료로 변경할 것.

