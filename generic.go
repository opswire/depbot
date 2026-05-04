package parser

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Generic — fallback-стратегия для текстовых файлов из whitelist'а
// (README, manuals, конфиги). Регексом ловит подстроки, похожие на
// Docker-ссылки, и проверяет каждую через image.Parse. Строгий парсер
// сам отсекает short-form, host:port, чистые версии и тому подобное:
// принимаются только полные ссылки domain.tld/name:semver[@sha256:...].
type Generic struct{}

// Name возвращает имя стратегии.
func (strategy *Generic) Name() string { return "generic" }

// genericAllowedExtensions — расширения текстовых файлов, в которых имеет
// смысл искать ссылки на образы fallback-стратегией.
var genericAllowedExtensions = map[string]bool{
	".md":         true,
	".txt":        true,
	".rst":        true,
	".adoc":       true,
	".log":        true,
	".conf":       true,
	".cfg":        true,
	".ini":        true,
	".properties": true,
}

// Match — Generic запускается только на текстовых форматах из
// genericAllowedExtensions.
func (strategy *Generic) Match(filePath string) bool {
	extension := strings.ToLower(filepath.Ext(filePath))
	return genericAllowedExtensions[extension]
}

// genericTokenRegexp матчит подряд идущие "Docker-валидные" символы,
// внутри которых есть хотя бы один "/" (для domain/name) и ":" или "@"
// (для тега или дайджеста). Группа 1 — сам токен.
var genericTokenRegexp = regexp.MustCompile(
	`([A-Za-z0-9][A-Za-z0-9.\-]*[/][A-Za-z0-9._\-/]*[:@][A-Za-z0-9._\-/:@]+)`)

// Parse находит все токены, которые tryParseImageRef признаёт ссылками.
func (strategy *Generic) Parse(filePath string, content []byte) ([]*Occurrence, error) {
	return scanByRegex(filePath, content, content, genericTokenRegexp, 1, commentLineSkip), nil
}
