# 📝 04. ADF (Atlassian Document Format) 변환 엔진

이 문서는 Jira Cloud REST API v3에서 본문 필드(`description`, `comment`) 작성 및 표시에 필수적인 **ADF(Atlassian Document Format) 변환 엔진**의 내부 구조와 파싱 알고리즘을 설명합니다.

---

## ❓ 왜 ADF 변환 엔진이 필요한가?

Atlassian Jira Cloud REST API v3는 이슈의 설명(`description`)과 댓글(`comment`) 필드에 일반 문자열이나 HTML을 허용하지 않으며, **ADF(Atlassian Document Format)**라는 엄격한 JSON 기반 AST(Abstract Syntax Tree) 구조만을 허용합니다.

만약 일반 텍스트나 잘못된 JSON을 전송하면 Jira API는 `400 Bad Request` 에러를 반환합니다.

이 프로젝트는 별도의 외부 Markdown 파서나 무거운 종속성 없이, **Go 표준 라이브러리만으로 동작하는 경량 양방향 ADF 엔진**을 구현하여 다음을 가능하게 합니다:
1. **Markdown → ADF (생성/수정/댓글 시)**: 사용자가 터미널이나 에이전트에서 작성한 일반 마크다운 문자열을 완전한 Jira ADF JSON 트리로 변환.
2. **ADF → Plain Text (조회 시)**: Jira 서버에서 반환한 중첩된 ADF JSON 트리를 터미널 화면에 보기 좋은 텍스트(글머리 기호, 구분선 등)로 복원.

---

## 🏗️ 1. ADF 트리 구조 개요

ADF 문서는 루트 노드(`type: "doc"`)와 자식 노드(`content`)의 트리 구조로 구성됩니다.

```json
{
  "type": "doc",
  "version": 1,
  "content": [
    {
      "type": "heading",
      "attrs": { "level": 2 },
      "content": [{ "type": "text", "text": "배경 및 목적" }]
    },
    {
      "type": "bulletList",
      "content": [
        {
          "type": "listItem",
          "content": [
            {
              "type": "paragraph",
              "content": [
                { "type": "text", "text": "중요한 항목: " },
                { "type": "text", "text": "필수 작업", "marks": [{ "type": "strong" }] }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

---

## 🔨 2. Markdown → ADF 빌더 (`BuildADFDocument`)

구현 위치: [`pkg/types.go`](file:///home/yundream/myjob/cloit/todo/tools/jira/pkg/types.go)

### 지원 마크다운 문법
- **Code Block**: ` ```go ... ``` ` (언어 속성 `language` 추출 지원)
- **Headings**: `#` ~ `######` (Heading 1 ~ 6 레벨 매핑)
- **Bullet List**: `- item` 또는 `* item`
- **Ordered List**: `1. item`, `2. item` 등 숫자 목록
- **Blockquote**: `> quote text`
- **Horizontal Rule**: `---`, `***`, `___`
- **Inline Marks**:
  - 굵은 글씨: `**굵은 텍스트**` → `marks: [{"type": "strong"}]`
  - 인라인 코드: `` `코드` `` → `marks: [{"type": "code"}]`

### 파서 상태 머신 (State Machine) 알고리즘

입력 텍스트를 줄(`\n`) 단위로 순회하면서 라인의 접두사와 상태 플래그를 기반으로 AST 노드를 빌드합니다:

```mermaid
flowchart TD
    ReadLine[한 줄 읽기] --> CodeBlockCheck{inCodeBlock 상태인가?}
    
    CodeBlockCheck -->|Yes| CheckEndCode{``` 종료 태그인가?}
    CheckEndCode -->|Yes| FinishCode[codeBlock 노드 완성 및 content 추가]
    CheckEndCode -->|No| AppendCode[codeLines에 라인 추가]
    
    CodeBlockCheck -->|No| CheckStartCode{``` 시작 태그인가?}
    CheckStartCode -->|Yes| StartCode[flushLists() 후 inCodeBlock = true]
    
    CheckStartCode -->|No| CheckHR{--- / *** 구분선인가?}
    CheckHR -->|Yes| AddHR[flushLists() 후 rule 노드 추가]
    
    CheckHR -->|No| CheckHeading{# 으로 시작하는 제목인가?}
    CheckHeading -->|Yes| AddHeading[flushLists() 후 heading 노드 추가]
    
    CheckHeading -->|No| CheckBullet{- 또는 * 글머리인가?}
    CheckBullet -->|Yes| AddBullet[inBulletList 상태 유지 및 아이템 누적]
    
    CheckBullet -->|No| CheckOrdered{숫자. 순서 목록인가?}
    CheckOrdered -->|Yes| AddOrdered[inOrderedList 상태 유지 및 아이템 누적]
    
    CheckOrdered -->|No| CheckQuote{> 인용문인가?}
    CheckQuote -->|Yes| AddQuote[flushLists() 후 blockquote 노드 추가]
    
    CheckQuote -->|No| AddPara[flushLists() 후 일반 paragraph 노드 추가]
```

#### `flushLists()`의 역할
- 불릿 목록(`bulletList`)이나 순서 목록(`orderedList`)은 연속된 여러 라인을 그룹핑해야 합니다.
- 새로운 단락이나 제목, 코드 블록을 만나면 지금까지 누적된 목록 아이템들을 상위 리스트 노드로 감싸서 최종 트리에 커밋합니다.

#### 인라인 토크나이저 (`parseInlineText`)
- 텍스트 내부의 `**` (Bold)와 `` ` `` (Code)를 룬(rune) 단위로 스캔하여 텍스트 노드와 `marks` 배열로 분할합니다.

---

## 👓 3. ADF → Plain Text 렌더러 (`ExtractTextFromADF`)

구현 위치: [`pkg/types.go`](file:///home/yundream/myjob/cloit/todo/tools/jira/pkg/types.go)

### 처리 방식
Jira API 응답의 `description` 또는 `comment.body`를 재귀적으로 순회합니다:
- **`text` 노드**: 텍스트 값을 `strings.Builder`에 기록.
- **`rule` 노드**: `---...---` 형태의 80칸 터미널 구분선 출력.
- **`listItem` 노드**: 들여쓰기와 함께 `• ` 불릿 기호 추가.
- **`paragraph`, `heading`, `codeBlock`, `blockquote` 노드**: 블록 끝에 개행(`\n`) 추가.

---

## 🧪 4. 검증 및 테스트

[`pkg/types_test.go`](file:///home/yundream/myjob/cloit/todo/tools/jira/pkg/types_test.go)에 다음 항목에 대한 단위 테스트가 작성되어 있습니다:
- 빈 문자열 처리
- 복합 마크다운(제목 + 코드블록 + 목록 + 굵은 글씨) 변환 일치 여부
- 양방향 라운드트립(Round-trip) 안정성 테스트
