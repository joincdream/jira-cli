# 🛠️ 06. 개발 및 확장 가이드 (Developer & LLM Extension Guide)

> **AI 에이전트(LLM)와 개발자를 위한 유지보수 및 기능 확장 핸드북**  
> 새로운 명령어를 추가하거나 기존 기능을 수정할 때 반드시 본 문서의 설계 규약과 체크리스트를 준수해야 합니다.

---

## 🧭 1. 절대 준수 원칙 (Non-Negotiable Rules)

1. **외부 의존성 추가 금지 (Zero External Dependencies)**:
   - Go 표준 라이브러리(`net/http`, `flag`, `encoding/json`, `context`, `text/tabwriter` 등)만 사용합니다.
   - 서드파티 CLI/HTTP 라이브러리를 임의로 `go get`하지 않습니다.
2. **Context 전파 필수**:
   - 모든 커맨드와 HTTP 요청은 `ctx context.Context`를 첫 번째 인수로 전달받고, 취소 시그널(Ctrl+C 등) 발생 시 즉시 중단되어야 합니다.
3. **일관된 에러 및 출력 규격**:
   - 실패 시 반드시 표준 에러(`stderr`)로 메시지를 출력하고, `Exit Code 1`을 반환합니다.
   - 표준 출력(`stdout`)은 터미널 사용자 및 LLM 파서가 읽기 쉬운 정돈된 테이블 또는 키-값 형식이어야 합니다.
4. **테스트 동반 필수**:
   - 비즈니스 로직 추가/수정 시 반드시 `*_test.go` 단위 테스트를 작성하여 검증합니다.

---

## 🚀 2. 신규 명령어(Command) 추가 절차

예시: 이슈의 담당자를 변경하는 `assign` 커맨드를 추가하는 워크플로우

### Step 1. `pkg/client.go`에 API 메서드 추가 (필요 시)
`pkg/types.go`에 필요한 DTO를 정의하고, `pkg/client.go`에 HTTP 요청 로직을 추가합니다:
```go
// pkg/client.go
func (c *Client) AssignIssue(ctx context.Context, key, accountID string) error {
    payload := map[string]string{
        "accountId": accountID,
    }
    endpoint := fmt.Sprintf("/rest/api/3/issue/%s/assignee", url.PathEscape(key))
    _, _, err := c.doRequest(ctx, "PUT", endpoint, payload)
    return err
}
```

### Step 2. `internal/app/`에 커맨드 구조체 구현
`Command` 인터페이스(`Name`, `Aliases`, `Description`, `Execute`)를 구현합니다:
```go
// internal/app/commands_issue.go
type AssignCommand struct {
    clientProvider ClientProvider
}

func NewAssignCommand(cp ClientProvider) *AssignCommand {
    return &AssignCommand{clientProvider: cp}
}

func (c *AssignCommand) Name() string        { return "assign" }
func (c *AssignCommand) Aliases() []string  { return []string{"assignee"} }
func (c *AssignCommand) Description() string { return "이슈의 담당자를 변경합니다." }

func (c *AssignCommand) Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
    if len(args) < 2 {
        return fmt.Errorf("사용법: jira assign <KEY> <ACCOUNT_ID>")
    }
    key := strings.ToUpper(args[0])
    accountID := args[1]

    client, err := c.clientProvider()
    if err != nil {
        return fmt.Errorf("클라이언트 초기화 오류: %w", err)
    }

    if err := client.AssignIssue(ctx, key, accountID); err != nil {
        return fmt.Errorf("담당자 변경 실패: %w", err)
    }

    fmt.Fprintf(stdout, "✅ [%s] 담당자가 성공적으로 변경되었습니다.\n", key)
    return nil
}
```

### Step 3. `internal/app/app.go`에 등록
`registerBuiltinCommands()` 함수에 신규 커맨드를 등록합니다:
```go
// internal/app/app.go
func (a *App) registerBuiltinCommands() {
    ...
    a.Register(NewAssignCommand(a.clientProvider)) // 등록 추가
}
```

### Step 4. `PrintUsage()` 도움말 갱신
`internal/app/app.go`의 `PrintUsage()` 함수 텍스트에 새 명령어와 옵션을 추가합니다.

---

## 🧪 3. 단위 테스트 작성 가이드

이 프로젝트는 의존성 주입(`ClientProvider`, `CatalogLoader`)을 채택하고 있으므로 실제 Jira 네트워크 연결 없이 완벽한 테스트가 가능합니다.

### Mock Server를 활용한 Client 테스트 (`pkg/client_test.go`)
```go
func TestClient_AssignIssue(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/rest/api/3/issue/KAN-1/assignee" && r.Method == "PUT" {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    client := NewClientWithConfig(Config{
        InstanceURL: server.URL,
        Email:       "test@example.com",
        APIToken:    "token",
        ProjectKey:  "KAN",
    })

    err := client.AssignIssue(context.Background(), "KAN-1", "user-123")
    if err != nil {
        t.Fatalf("기대하지 않은 에러: %v", err)
    }
}
```

### App 레벨 테스트 (`internal/app/app_test.go`)
`app.NewApp()`에 Mock Provider 함수를 전달하여 커맨드 라우터 및 플래그 파싱을 검증합니다:
```go
func TestApp_AssignCommand_Mock(t *testing.T) {
    mockClient := ... // Mock 설정
    app := NewApp("test", func() (*pkg.Client, error) {
        return mockClient, nil
    }, nil)

    var stdout, stderr bytes.Buffer
    code := app.Run(context.Background(), []string{"assign", "KAN-1", "user-123"}, &stdout, &stderr)
    if code != 0 {
        t.Fatalf("기대 exit code 0, 실제: %d, stderr: %s", code, stderr.String())
    }
}
```

---

## 📋 4. 개발 체크리스트 (Checklist for LLM & Developers)

새로운 기능을 완성한 후 다음 체크리스트를 반드시 확인하십시오:

- [ ] Go 표준 라이브러리 외 외부 모듈이 `go.mod`에 추가되지 않았는가?
- [ ] 신규 커맨드가 `Command` 인터페이스를 충족하고 `app.registerBuiltinCommands()`에 등록되었는가?
- [ ] `App.PrintUsage()`에 새로운 명령어가 기재되었는가?
- [ ] `description`이나 `comment` 본문을 다룰 경우 `BuildADFDocument()` / `ExtractTextFromADF()`를 올바르게 적용하였는가?
- [ ] 라벨을 조작할 경우 `ValidateAndNormalizeLabels()`를 통해 검증하고 `--force-labels` 옵션을 지원하는가?
- [ ] `go test -v ./...` 명령이 모든 테스트를 통과하는가?
- [ ] `make build`를 통해 빌드가 정상 완료되는가?
