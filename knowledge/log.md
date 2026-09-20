---
title: "Jira CLI Knowledge Log"
type: log
description: "Jira CLI Open Knowledge Format(OKF) 지식 베이스 생성 및 갱신 이력"
tags:
  - okf
  - log
  - changelog
status: stable
timestamp: 2026-09-20T18:00:00+09:00
verified: true
---

# 📜 Jira CLI Knowledge Log

본 문서는 `knowledge/` 번들 내 지식 자산의 생성, 코드 변경에 따른 동기화 및 검증 이력을 기록합니다.

---

## 2026-09-20 (Initial OKF Knowledge Bundle & v1.5.0 Update)
- **작성자**: Antigravity AI Assistant & joincdream
- **변경 사항**:
  1. Google Open Knowledge Format(OKF) 표준 규격(`SPEC.md`)에 맞춘 지식 베이스 번들 초기화.
  2. `pkg/format.go` 내 Summary 50자 슬라이싱 제거 및 한글 UTF-8 깨짐 결함 해결 반영.
  3. `jira list` 및 `jira get` 명령어에 머신 리더블 출력 옵션(`--md`, `--json`, `-o`) 추가 및 맥락 통합 뷰 문서화.
  4. 프로젝트 로컬 프로필 자동 탐색(`.jira-profile`) 엔진 추가 및 4단계 프로필 결정 우선순위 명세화.
  5. `jira help` 출력 및 `~/.config/jira/SKILL.md` 문서 최신화.
- **검증 소스**:
  - `pkg/format_test.go`, `pkg/client_test.go`, `internal/app/app_test.go` (테스트 100% Pass)
