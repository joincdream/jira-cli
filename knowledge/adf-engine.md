---
title: "ADF (Atlassian Document Format) Conversion Engine"
type: concept
description: "일반 Markdown 텍스트와 Jira Cloud v3 ADF(Atlassian Document Format) 간 양방향 변환 알고리즘"
tags:
  - adf
  - markdown
  - parser
  - rich-text
status: stable
timestamp: 2026-09-20T18:00:00+09:00
sources:
  - pkg/types.go
  - pkg/types_test.go
verified: true
---

# 📝 ADF (Atlassian Document Format) 변환 엔진

Jira Cloud REST API v3는 이슈의 설명(`description`)과 코멘트(`comment`) 필드에 평문 대신 **ADF (Atlassian Document Format)**라 불리는 복잡한 JSON 트리 구조를 요구합니다. Jira CLI는 사용자와 AI 에이전트가 친숙한 Markdown 텍스트로 작업할 수 있도록 내장 ADF 변환 엔진을 제공합니다.

---

## 🌲 1. ADF 트리 구조 개요

ADF는 문서(`doc`) 루트 아래에 블록 노드(`paragraph`, `heading`, `bulletList`, `codeBlock` 등)와 인라인 노드(`text`, `marks`)로 구성된 AST(Abstract Syntax Tree)입니다.

```json
{
  "type": "doc",
  "version": 1,
  "content": [
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "일반 텍스트 " },
        { "type": "text", "text": "굵은 글씨", "marks": [{ "type": "strong" }] }
      ]
    }
  ]
}
```

---

## 🔄 2. Markdown → ADF 변환 (`BuildADFDocument`)

[pkg/types.go](file:///mnt/data/myjob/cloit/jira-cli/pkg/types.go)의 `BuildADFDocument(text string)`는 줄 단위 순차 파싱 기법을 사용하여 마크다운을 올바른 ADF 구조로 직렬화합니다.

### 지원 문법 및 노드 매핑
1. **코드 블록 (Fenced Code Blocks)**:
   - ` ```go ... ``` ` 형태를 감지하여 `codeBlock` 노드로 변환하며 언어 속성(`attrs.language`)을 보존합니다.
2. **제목 (Headings)**:
   - `#` 개수(1~6개)를 계산하여 `attrs.level`을 가진 `heading` 노드로 생성합니다.
3. **인용문 (Blockquotes)**:
   - `>`로 시작하는 라인을 `blockquote` 노드로 감쌉니다.
4. **리스트 (Bullet & Ordered Lists)**:
   - `- `, `* ` (Bullet) 및 `1. ` (Ordered) 형태의 연속된 항목들을 모아 `flushLists()`를 통해 부모 `bulletList` 또는 `orderedList` 컨테이너 노드로 묶어줍니다.
5. **수평선 (Horizontal Rules)**:
   - `---`, `***`, `___`를 `rule` 노드로 변환합니다.
6. **인라인 서식 (`parseInlineMarks`)**:
   - `**bold**` → `marks: [{"type": "strong"}]`
   - `` `inline_code` `` → `marks: [{"type": "code"}]`

---

## 📤 3. ADF → 텍스트 역변환 (`ExtractTextFromADF`)

Jira API로부터 수신된 중첩 JSON 트리에서 터미널 출력 및 마크다운 문서 뷰에 적합한 텍스트를 복원합니다.

- **알고리즘**:
  - `extractTextRecursive(node, sb, depth)`를 통해 노드 타입을 검사하며 재귀 탐색합니다.
  - `listItem`은 `• ` 불릿 기호로 들여쓰기 처리됩니다.
  - `rule`은 수평선 구분자로 복원됩니다.
  - `paragraph`, `heading`, `codeBlock` 노드는 자연스러운 개행(`\n`)을 보장합니다.
- **예외 복원력**:
  - Jira Server 또는 구버전 API에서 ADF 대신 일반 `string`이 전달되는 경우에도 타입 단언(`body.(string)`)을 통해 오류 없이 안전하게 평문으로 처리합니다.
