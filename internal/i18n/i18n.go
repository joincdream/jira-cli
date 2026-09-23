package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	mu          sync.RWMutex
	messages    = make(map[string]map[string]string)
	currentLang = "en"
)

func init() {
	loadLocale("en")
	loadLocale("ko")
	currentLang = ResolveLanguage("", "")
}

func loadLocale(lang string) {
	data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.json", lang))
	if err != nil {
		return
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err == nil {
		messages[lang] = m
	}
}

// NormalizeLang converts raw language inputs (e.g., "ko_KR.UTF-8", "KO", "en-US") to supported code ("ko", "en").
func NormalizeLang(l string) string {
	l = strings.ToLower(strings.TrimSpace(l))
	if strings.HasPrefix(l, "ko") {
		return "ko"
	}
	if strings.HasPrefix(l, "en") {
		return "en"
	}
	return "en"
}

// ResolveLanguage determines the active language code based on 5-tier priority:
// 1. CLI flag (--lang)
// 2. Environment variable (JIRA_LANG)
// 3. Profile setting in config (language = ...)
// 4. OS environment variable (LC_ALL, LANG)
// 5. Fallback default ("en")
func ResolveLanguage(cliLang, profileLang string) string {
	if cliLang != "" {
		return NormalizeLang(cliLang)
	}
	if env := os.Getenv("JIRA_LANG"); env != "" {
		return NormalizeLang(env)
	}
	if profileLang != "" {
		return NormalizeLang(profileLang)
	}
	if lc := os.Getenv("LC_ALL"); lc != "" {
		return NormalizeLang(lc)
	}
	if lang := os.Getenv("LANG"); lang != "" {
		return NormalizeLang(lang)
	}
	return "en"
}

// Init sets the active language based on CLI and profile parameters.
func Init(cliLang, profileLang string) {
	SetLanguage(ResolveLanguage(cliLang, profileLang))
}

// SetLanguage sets the active language.
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	currentLang = NormalizeLang(lang)
}

// CurrentLanguage returns the active language code.
func CurrentLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// Sprintf returns a formatted string localized to the active language.
// If the key is missing in active language, it falls back to "en".
// If still missing, key itself is used as the template.
func Sprintf(key string, args ...interface{}) string {
	mu.RLock()
	lang := currentLang
	mu.RUnlock()

	template, ok := messages[lang][key]
	if !ok {
		template, ok = messages["en"][key]
		if !ok {
			template = key
		}
	}

	return fmt.Sprintf(template, args...)
}

// T returns the translated string for a key with no formatting arguments.
func T(key string) string {
	return Sprintf(key)
}
