package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"tools/jira/pkg"
)

func setupUserMockApp(t *testing.T) (*App, *int32, *int32) {
	var assignCalled int32
	var usersCalled int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/api/3/myself":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"accountId": "acc-myself",
				"displayName": "My Self",
				"emailAddress": "myself@example.com",
				"active": true
			}`))

		case r.Method == "GET" && r.URL.Path == "/rest/api/3/user/assignable/search":
			atomic.AddInt32(&usersCalled, 1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"accountId": "acc-assignable-1",
					"displayName": "Assignable User",
					"emailAddress": "assignable@example.com",
					"active": true
				}
			]`))

		case r.Method == "GET" && r.URL.Path == "/rest/api/3/users/search":
			atomic.AddInt32(&usersCalled, 1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[
				{
					"accountId": "acc-global-1",
					"displayName": "Global User",
					"emailAddress": "global@example.com",
					"active": true
				}
			]`))

		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/") && strings.HasSuffix(r.URL.Path, "/assignee"):
			atomic.AddInt32(&assignCalled, 1)
			w.WriteHeader(http.StatusNoContent)

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"key": "JC-10",
				"fields": {
					"summary": "Sample Issue",
					"status": {"name": "To Do"},
					"project": {"key": "JC", "name": "Jira CLI"}
				}
			}`))

		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(server.Close)

	clientProvider := func() (*pkg.Client, error) {
		cfg := pkg.Config{
			InstanceURL: server.URL,
			Email:       "test@example.com",
			APIToken:    "dummy_token",
			ProjectKey:  "JC",
		}
		return pkg.NewClientWithConfig(cfg), nil
	}

	mockCatalogLoader := func() (*pkg.LabelCatalog, error) {
		return &pkg.LabelCatalog{
			Version:    "1.0.0",
			Categories: map[string]pkg.LabelCategory{},
		}, nil
	}

	app := NewApp("1.5.0", clientProvider, mockCatalogLoader)
	return app, &assignCalled, &usersCalled
}

func TestAssignCommand(t *testing.T) {
	t.Run("assign to me", func(t *testing.T) {
		app, assignCalled, _ := setupUserMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"assign", "JC-10", "me"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(assignCalled) != 1 {
			t.Errorf("expected 1 assign call, got %d", atomic.LoadInt32(assignCalled))
		}
		if !strings.Contains(stdout.String(), "JC-10") {
			t.Errorf("expected JC-10 in output, got %s", stdout.String())
		}
	})

	t.Run("assign to none", func(t *testing.T) {
		app, assignCalled, _ := setupUserMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"assign", "JC-10", "none"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(assignCalled) != 1 {
			t.Errorf("expected 1 assign call, got %d", atomic.LoadInt32(assignCalled))
		}
	})

	t.Run("missing arguments", func(t *testing.T) {
		app, _, _ := setupUserMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"assign", "JC-10"}, &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected code 1 for missing user, got %d", code)
		}
	})
}

func TestUsersCommand(t *testing.T) {
	t.Run("global search table", func(t *testing.T) {
		app, _, usersCalled := setupUserMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"users", "global"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(usersCalled) != 1 {
			t.Errorf("expected 1 users call, got %d", atomic.LoadInt32(usersCalled))
		}
		if !strings.Contains(stdout.String(), "Global User") {
			t.Errorf("expected Global User in output, got: %s", stdout.String())
		}
	})

	t.Run("assignable search markdown", func(t *testing.T) {
		app, _, usersCalled := setupUserMockApp(t)
		var stdout, stderr bytes.Buffer

		code := app.Run(context.Background(), []string{"users", "--assignable", "-p", "JC", "--md"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected code 0, got %d, stderr: %s", code, stderr.String())
		}
		if atomic.LoadInt32(usersCalled) != 1 {
			t.Errorf("expected 1 users call, got %d", atomic.LoadInt32(usersCalled))
		}
		if !strings.Contains(stdout.String(), "Assignable User") {
			t.Errorf("expected Assignable User in output, got: %s", stdout.String())
		}
	})
}
