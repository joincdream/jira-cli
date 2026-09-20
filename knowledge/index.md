---
title: "Jira CLI Knowledge Index"
type: index
description: "Jira CLI 코드베이스 유지보수, 기능 확장 및 AI 협업을 위한 Open Knowledge Format(OKF) 지식 카탈로그"
tags:
  - okf
  - index
  - catalog
  - jira-cli
status: stable
timestamp: 2026-09-20T18:00:00+09:00
sources:
  - README.md
  - cmd/main.go
  - internal/app/app.go
verified: true
---

# 🧠 Jira CLI Knowledge Index (OKF)

Jira CLI는 Atlassian Jira Cloud를 터미널과 AI Agent 환경에서 효율적으로 제어하기 위한 초경량, 무의존성(Zero-Dependency) Go 도구입니다. 본 지식 베이스(Knowledge Bundle)는 Google Open Knowledge Format(OKF) 규격을 준수하여 작성되었으며, 인간 개발자와 생성형 AI가 코드베이스를 안정적으로 유지보수하고 기능을 확장할 수 있도록 돕습니다.

---

## 📚 지식 카탈로그 (Knowledge Catalog)

| 문서명 | 유형 (`type`) | 연결 소스 (`sources`) | 설명 |
| :--- | :---: | :--- | :--- |
| [architecture.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/architecture.md) | `architecture` | `cmd/main.go`, `internal/app/app.go`, `internal/app/command.go` | 무의존성 원칙, 3-Tier 계층 구조 및 의존성 주입(DI) 설계 |
| [configuration.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/configuration.md) | `architecture` | `pkg/client.go`, `internal/app/commands_config.go` | INI 멀티 프로필, `.jira-profile` 로컬 프로젝트 자동 탐색 및 우선순위 규칙 |
| [commands-and-routing.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/commands-and-routing.md) | `api` | `internal/app/commands_issue.go`, `internal/app/commands_meta.go` | CLI 명령어 라우팅, 플래그 파싱 및 머신 리더블(`--md`, `--json`) 출력 엔진 |
| [adf-engine.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/adf-engine.md) | `concept` | `pkg/types.go` | Markdown ↔ Atlassian Document Format(ADF) 양방향 파싱 및 변환 알고리즘 |
| [data-models.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/data-models.md) | `data-model` | `pkg/types.go`, `pkg/format.go` | Jira REST API v3 DTO 매핑, Issue 구조체 및 포맷터 |
| [labels-catalog.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/labels-catalog.md) | `guide` | `pkg/labels.go`, `internal/app/commands_meta.go` | 표준 라벨 카탈로그 검증 규칙, 대소문자 정규화 및 탐색 경로 |
| [log.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/log.md) | `log` | - | 지식 베이스 변경 및 검증 이력 추적 (Changelog) |

---

## 🧭 AI 에이전트 작업 지침 (AI Agent Navigational Guide)

생성형 AI가 본 코드베이스를 수정하거나 새 기능을 개발할 때는 다음 순서로 지식을 참조합니다:

1. **설정 및 인증 변경**: [configuration.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/configuration.md) 확인 (`~/.config/jira/config`와 `.jira-profile` 연동 로직 준수)
2. **새로운 명령어 추가 / 플래그 변경**: [commands-and-routing.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/commands-and-routing.md) 및 [architecture.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/architecture.md) 확인 (Command 인터페이스 및 Factory 패턴 준수)
3. **이슈 설명 및 코멘트 서식 처리**: [adf-engine.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/adf-engine.md) 확인 (ADF 트리 노드 규격 준수)
4. **라벨 관련 작업**: [labels-catalog.md](file:///mnt/data/myjob/cloit/jira-cli/knowledge/labels-catalog.md) 확인 (오타 방지 검증 파이프라인 준수)
