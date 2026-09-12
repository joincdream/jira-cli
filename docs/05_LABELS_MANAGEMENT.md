# 🏷️ 05. 라벨 정책 및 카탈로그 시스템 (Labels Management)

이 문서는 Jira 이슈의 일관된 분류와 검색 품질을 보장하기 위한 **표준 라벨 카탈로그 시스템**과 검증 메커니즘을 설명합니다.

---

## 🎯 1. 도입 배경 및 목적

여러 팀원이나 AI 에이전트가 Jira 티켓을 생성/수정할 때 발생할 수 있는 주요 문제:
- **오타 및 표기 불일치**: `AgentGo`, `agentgo`, `Agent_Go` 등 동일 의미의 라벨이 파편화되어 JQL 필터 검색이 실패함.
- **불명확한 태깅**: 프로젝트와 무관한 임의의 태그가 난립하여 보드 관리가 어려워짐.

Jira CLI는 **사전 정의된 표준 카탈로그 기반의 검증 및 대소문자 정규화**를 통해 이 문제를 해결합니다.

---

## 📂 2. 카탈로그 파일 위치 및 탐색 순서

구현 위치: [`pkg/labels.go`](file:///home/yundream/myjob/cloit/todo/tools/jira/pkg/labels.go)

CLI 실행 시 현재 작업 디렉토리(`cwd`)에서부터 상위 디렉토리로 올라가며 다음 파일을 탐색합니다:
1. `.agents/labels.json`
2. `labels.json`

만약 상위 루트까지 탐색해도 파일이 존재하지 않는 경우, [`getDefaultCatalog()`](file:///home/yundream/myjob/cloit/todo/tools/jira/pkg/labels.go#L153)에 정의된 내장 기본 카탈로그가 자동으로 활성화됩니다.

---

## 📄 3. 카탈로그 JSON 스키마

```json
{
  "version": "1.0.0",
  "categories": {
    "project": {
      "name": "프로젝트 및 고객사",
      "description": "연관된 프로젝트 또는 고객사 구분",
      "labels": [
        { "name": "DIVE", "description": "DIVE MCP AI Agent 플랫폼 구축 사업" },
        { "name": "AgentGo_Studio", "description": "AgentGo Studio 서비스 개발" },
        { "name": "Fursys", "description": "퍼시스 관련 프로젝트 및 과제" }
      ]
    },
    "tech": {
      "name": "기술 및 도메인",
      "description": "관련 기술 스택 및 도메인",
      "labels": [
        { "name": "AI", "description": "인공지능, VLM, LLM, Agent 기술" },
        { "name": "MCP", "description": "Model Context Protocol 서버 및 연동" },
        { "name": "Cloud", "description": "AWS, GCP, 클라우드 인프라" }
      ]
    },
    "activity": {
      "name": "업무 성격 및 산출물",
      "description": "작업의 활동 성격 및 결과물 형태",
      "labels": [
        { "name": "Planning", "description": "기획, 일정 수립, MM 산정, R&R 정의" },
        { "name": "Development", "description": "기능 구현 및 코딩" },
        { "name": "QA", "description": "테스트 및 품질 검증" }
      ]
    }
  }
}
```

---

## ⚙️ 4. 라벨 검증 및 자동 정규화 알고리즘

구현 함수: `ValidateAndNormalizeLabels(inputLabels []string) ([]string, []string, error)`

1. **대소문자 무관 매핑 테이블 구축**:
   - 카탈로그의 모든 라벨을 `정규이름 -> 정규이름`, `소문자이름 -> 정규이름` 맵으로 인덱싱합니다.
2. **입력 라벨 정규화**:
   - 사용자가 `jira create "..." -l "dive,ai,planning"`처럼 소문자로 입력해도, 카탈로그 매칭을 통해 `["DIVE", "AI", "Planning"]`의 표준 대소문자로 자동 보정합니다.
3. **미승인 라벨 차단**:
   - 카탈로그에 없는 라벨(예: `RandomTag`)이 포함되어 있을 경우 에러를 반환하고 티켓 생성을 방지합니다:
     ```text
     ❌ 유효하지 않은 라벨: RandomTag
     💡 'jira labels' 명령어로 등록 가능한 표준 라벨 카탈로그를 확인하세요.
     ```

---

## 🔓 5. 예외 우회: `--force-labels` 플래그

긴급 상황이거나 카탈로그에 아직 반영되지 않은 신규 라벨을 즉시 사용해야 하는 경우:
```bash
jira create "긴급 핫픽스" -l "Hotfix,UnregisteredTag" --force-labels
jira edit KAN-10 -l "NewFeature" --force-labels
```
`--force-labels` 옵션을 지정하면 카탈로그 유효성 검사를 건너뛰고 입력된 라벨 그대로 Jira 서버에 전달합니다.
