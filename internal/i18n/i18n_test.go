package i18n

import (
	"os"
	"testing"
)

func TestResolutionPriority(t *testing.T) {
	// Clean environment
	os.Unsetenv("JIRA_LANG")
	os.Unsetenv("LC_ALL")
	os.Unsetenv("LANG")

	// Case 1: Default fallback -> en
	if got := ResolveLanguage("", ""); got != "en" {
		t.Errorf("expected 'en', got '%s'", got)
	}

	// Case 2: OS locale LANG=ko_KR.UTF-8 -> ko
	os.Setenv("LANG", "ko_KR.UTF-8")
	if got := ResolveLanguage("", ""); got != "ko" {
		t.Errorf("expected 'ko', got '%s'", got)
	}

	// Case 3: Profile setting overrides OS locale
	if got := ResolveLanguage("", "en"); got != "en" {
		t.Errorf("expected 'en', got '%s'", got)
	}

	// Case 4: JIRA_LANG overrides profile setting
	os.Setenv("JIRA_LANG", "ko")
	if got := ResolveLanguage("", "en"); got != "ko" {
		t.Errorf("expected 'ko', got '%s'", got)
	}

	// Case 5: CLI flag overrides JIRA_LANG
	if got := ResolveLanguage("en", "ko"); got != "en" {
		t.Errorf("expected 'en', got '%s'", got)
	}

	// Cleanup
	os.Unsetenv("JIRA_LANG")
	os.Unsetenv("LANG")
}

func TestTranslations(t *testing.T) {
	SetLanguage("en")
	if got := T("cmd.create.err_desc_required"); got == "" || got == "cmd.create.err_desc_required" {
		t.Errorf("unexpected translation for en: %s", got)
	}

	SetLanguage("ko")
	if got := T("cmd.create.err_desc_required"); got == "" || got == "cmd.create.err_desc_required" {
		t.Errorf("unexpected translation for ko: %s", got)
	}

	// Parameter formatting
	SetLanguage("en")
	msgEn := Sprintf("cmd.create.err_summary_min_length", "abc")
	if msgEn != "Issue summary must be at least 5 characters excluding whitespace. (Given: 'abc')" {
		t.Errorf("unexpected Sprintf en: %s", msgEn)
	}

	SetLanguage("ko")
	msgKo := Sprintf("cmd.create.err_summary_min_length", "abc")
	if msgKo != "이슈 제목은 공백 제외 최소 5자 이상이어야 합니다. (입력값: 'abc')" {
		t.Errorf("unexpected Sprintf ko: %s", msgKo)
	}
}

func TestMissingKeyGracefulFallback(t *testing.T) {
	SetLanguage("ko")
	missingKey := "non_existent_random_key_123"

	// No panic, returns key as template
	got := Sprintf(missingKey)
	if got != missingKey {
		t.Errorf("expected '%s', got '%s'", missingKey, got)
	}

	// With args
	gotWithArgs := Sprintf("missing.key.%s", "hello")
	if gotWithArgs != "missing.key.hello" {
		t.Errorf("expected 'missing.key.hello', got '%s'", gotWithArgs)
	}
}

func TestCatalogCompleteness(t *testing.T) {
	enKeys := messages["en"]
	koKeys := messages["ko"]

	if len(enKeys) == 0 || len(koKeys) == 0 {
		t.Fatalf("locales not loaded properly: en=%d, ko=%d", len(enKeys), len(koKeys))
	}

	for k := range enKeys {
		if _, exists := koKeys[k]; !exists {
			t.Errorf("key '%s' present in en but missing in ko", k)
		}
	}

	for k := range koKeys {
		if _, exists := enKeys[k]; !exists {
			t.Errorf("key '%s' present in ko but missing in en", k)
		}
	}
}
