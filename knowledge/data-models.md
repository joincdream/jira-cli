---
title: "Data Models & Jira API DTO Mapping"
type: data-model
description: "Jira Cloud REST API v3 데이터 구조체 매핑, DTO 및 상태 모델 정의"
tags:
  - data-model
  - dto
  - struct
  - types
status: stable
timestamp: 2026-09-20T18:00:00+09:00
sources:
  - pkg/types.go
  - pkg/client.go
verified: true
---

# 📊 데이터 모델 및 Jira API DTO 매핑

본 문서는 [pkg/types.go](file:///mnt/data/myjob/cloit/jira-cli/pkg/types.go)에 정의된 Jira 도메인 엔티티 구조체와 Atlassian Cloud REST API v3 페이로드 간의 직렬화/역직렬화 규격을 정의합니다.

---

## 🏛️ 1. 핵심 엔티티 모델 (Core Entity Models)

### `Issue` & `IssueFields`
Jira 티켓의 전체 메타데이터 및 본문을 보관하는 루트 엔티티입니다.

```go
type Issue struct {
    ID     string      `json:"id"`
    Key    string      `json:"key"`     // 예: "KAN-48", "JC-1"
    Self   string      `json:"self"`    // API URL
    Fields IssueFields `json:"fields"`
}

type IssueFields struct {
    Summary     string        `json:"summary"`
    Description interface{}   `json:"description,omitempty"` // ADF 구조체 또는 문자열
    Status      Status        `json:"status"`
    IssueType   IssueType     `json:"issuetype"`
    Priority    *Priority     `json:"priority,omitempty"`
    Labels      []string      `json:"labels,omitempty"`
    Assignee    *User         `json:"assignee,omitempty"`
    Reporter    *User         `json:"reporter,omitempty"`
    DueDate     string        `json:"duedate,omitempty"`    // "YYYY-MM-DD"
    Created     string        `json:"created,omitempty"`
    Updated     string        `json:"updated,omitempty"`
    Project     Project       `json:"project"`
    Subtasks    []Issue       `json:"subtasks,omitempty"`   // 하위 작업 목록
    Comment     *CommentBlock `json:"comment,omitempty"`    // 코멘트 목록
}
```

---

## 🌐 2. Jira REST API v3 요청/응답 페이로드 (DTOs)

### 1) 이슈 목록 검색 (`SearchResponse`)
- **엔드포인트**: `GET /rest/api/3/search/jql?jql=...&fields=...`
- **구조체**:
  ```go
  type SearchResponse struct {
      Issues        []Issue `json:"issues"`
      IsLast        bool    `json:"isLast"`
      NextPageToken string  `json:"nextPageToken,omitempty"`
  }
  ```

### 2) 이슈 생성 (`CreateIssueRequest`)
- **엔드포인트**: `POST /rest/api/3/issue`
- **필드 구성**:
  ```go
  type CreateIssueFields struct {
      Project     ProjectRef  `json:"project"`             // key: "KAN"
      Summary     string      `json:"summary"`
      Description interface{} `json:"description,omitempty"`// BuildADFDocument 결과
      IssueType   TypeRef     `json:"issuetype"`           // name: "작업", "스토리" 등
      Labels      []string    `json:"labels,omitempty"`
      DueDate     string      `json:"duedate,omitempty"`
      Parent      *ParentRef  `json:"parent,omitempty"`    // Subtask인 경우 상위키
  }
  ```

### 3) 부분 필드 업데이트 (`UpdateIssueOptions`)
- **엔드포인트**: `PUT /rest/api/3/issue/{key}`
- **Null 및 None 처리 규칙**:
  - `DueDate`: `"none"`, `""`, `"null"` 지정 시 JSON `null`로 전송하여 마감일 삭제.
  - `ParentKey`: `"none"`, `""` 지정 시 상위 연결 해제.
  - `Labels`: `UpdateLabels: true`인 경우에만 기존 라벨을 덮어씀.
