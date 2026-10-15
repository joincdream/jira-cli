# Claude Code Project Guidelines - Jira Integration

## Jira Tooling Commands
- 맥락 파악: `jira get <KEY> --md` (토큰 절약 마크다운)
- 내 미완료 작업: `jira list --assignee me --status open --md`
- 웹 브라우저 열기: `jira open <KEY>` (또는 `jira open` 프로젝트 보드)
- 작업 피드백 등록: `jira comment <KEY> "<MESSAGE>"`
- 상태 전이 (사용자 명시 지시 시에만): `jira move <KEY> "<STATUS>"`

## Output Standard
- Jira 티켓에 대해 사용자에게 답변할 때는 불필요한 전체 덤프를 피하고, **Agent Insight Block** 규격(현황, AI 추론, 추천 액션)으로 일관되게 브리핑할 것.

## Human-in-the-Loop Rules
- 단위 테스트를 통과했다고 해서 자의적으로 티켓을 완료(`jira move "완료"`)하지 말 것.
- 사용자가 "완료 처리해 줘"라고 명시적으로 지시할 때만 상태를 변경할 것.
