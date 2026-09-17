package i18n

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"sync"
	"text/template"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	mu           sync.RWMutex
	currentLang  = "en"
	templates    = make(map[string]*template.Template)
	fallbackTmpl = make(map[string]*template.Template)
)

func init() {
	loadLanguage("en", fallbackTmpl)
	Init("", "")
}

// NormalizeLanguage normalizes language tags like "ja-JP", "ja_JP", "JA" to "ja".
func NormalizeLanguage(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return ""
	}
	if idx := strings.IndexAny(lang, "-_."); idx != -1 {
		return lang[:idx]
	}
	return lang
}

// ResolveLanguage determines the language code to use based on priority:
// 1. flagLang (CLI argument)
// 2. prefLang (preferences.json)
// 3. DetectOSLanguage()
// 4. Default: "en"
func ResolveLanguage(flagLang, prefLang string) string {
	if norm := NormalizeLanguage(flagLang); norm != "" {
		return norm
	}
	if norm := NormalizeLanguage(prefLang); norm != "" {
		return norm
	}
	if osLang := NormalizeLanguage(DetectOSLanguage()); osLang != "" {
		return osLang
	}
	return "en"
}

// Init initializes the i18n package with language preference resolution.
func Init(flagLang, prefLang string) {
	lang := ResolveLanguage(flagLang, prefLang)
	SetLanguage(lang)
}

// SetLanguage sets the active language and loads its translations.
func SetLanguage(lang string) {
	norm := NormalizeLanguage(lang)
	if norm == "" {
		norm = "en"
	}

	newTmpl := make(map[string]*template.Template)
	loadLanguage(norm, newTmpl)

	mu.Lock()
	defer mu.Unlock()
	currentLang = norm
	templates = newTmpl
}

// CurrentLanguage returns the active language code.
func CurrentLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

func loadLanguage(lang string, dest map[string]*template.Template) {
	data, err := localesFS.ReadFile("locales/" + lang + ".json")
	if err != nil {
		return
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	for k, v := range raw {
		tmpl, err := template.New(k).Parse(v)
		if err == nil {
			dest[k] = tmpl
		}
	}
}

// T returns the translated string for key, formatting it with data if provided.
func T(key string, data ...map[string]any) string {
	mu.RLock()
	defer mu.RUnlock()

	tmpl, ok := templates[key]
	if !ok {
		tmpl = fallbackTmpl[key]
	}
	if tmpl == nil {
		return key
	}

	var param any
	if len(data) > 0 && data[0] != nil {
		param = data[0]
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, param); err != nil {
		return key
	}
	return buf.String()
}
