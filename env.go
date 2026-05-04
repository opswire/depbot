package parser

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
)

// EnvFile разбирает .env / .env.* / *.env. Файл — набор пар KEY=VALUE,
// по одной на строку, с опциональным префиксом "export" и кавычками.
//
// Эвристика, чтобы не путать порт или таймаут с образом: либо имя ключа
// содержит IMAGE/DOCKER/CONTAINER, либо значение содержит ":" или "/".
type EnvFile struct{}

// Name возвращает имя стратегии.
func (strategy *EnvFile) Name() string { return "env" }

// Match — .env, .env.<что-то>, *.env.
func (strategy *EnvFile) Match(filePath string) bool {
	base := filepath.Base(filePath)
	return base == ".env" ||
		strings.HasPrefix(base, ".env.") ||
		strings.HasSuffix(strings.ToLower(base), ".env")
}

// envAssignmentRegexp ищет KEY=value (опционально с export-префиксом и
// кавычками). RE2 не поддерживает backreference, поэтому варианты с
// двойными кавычками, одинарными и без кавычек разнесены в три альтернативы.
//
// Группы: 1 — ключ, 2/3/4 — значение в "..."/'...'/без кавычек.
var envAssignmentRegexp = regexp.MustCompile(
	`(?m)^[ \t]*(?:export[ \t]+)?([A-Za-z_][A-Za-z0-9_]*)[ \t]*=[ \t]*(?:"([^"\n]*)"|'([^'\n]*)'|([^\s#]+))[ \t]*(?:#.*)?$`)

// Parse находит все KEY=VALUE присваивания, удовлетворяющие эвристике.
func (strategy *EnvFile) Parse(filePath string, content []byte) ([]*Occurrence, error) {
	var occurrences []*Occurrence
	for _, match := range envAssignmentRegexp.FindAllSubmatchIndex(content, -1) {
		keyName := string(content[match[2]:match[3]])

		// Выбираем непустую группу значения.
		var valueStart, valueEnd int
		for groupIndex := 2; groupIndex <= 4; groupIndex++ {
			if match[2*groupIndex] >= 0 {
				valueStart, valueEnd = match[2*groupIndex], match[2*groupIndex+1]
				break
			}
		}
		if valueStart == 0 && valueEnd == 0 {
			continue
		}

		// Защита от случайно подошедших закомментированных строк.
		lineStart := bytes.LastIndexByte(content[:match[2]], '\n') + 1
		if lineStart < len(content) && content[lineStart] == '#' {
			continue
		}

		rawValue := string(content[valueStart:valueEnd])
		if !looksImagedAssignment(keyName, rawValue) {
			continue
		}
		parsedRef, ok := tryParseImageRef(rawValue)
		if !ok {
			continue
		}
		occurrences = append(occurrences, &Occurrence{
			Image:     parsedRef,
			File:      filePath,
			Line:      lineNumberAt(content, valueStart),
			Kind:      FullReference,
			StartByte: valueStart,
			EndByte:   valueEnd,
		})
	}
	return occurrences, nil
}
