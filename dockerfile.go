package parser

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Dockerfile разбирает Dockerfile / Containerfile / *.Dockerfile / Dockerfile.*.
// Распознаются формы:
//
//	FROM image[:tag][@sha256:...]
//	FROM --platform=... image[:tag] [AS name]
//
// Сознательно пропускаются:
//   - magic-base "scratch";
//   - аргумент-шаблонные основы вида "FROM ${BASE}";
//   - FROM, ссылающиеся на ранее объявленную стадию по алиасу.
type Dockerfile struct{}

// Name возвращает имя стратегии.
func (strategy *Dockerfile) Name() string { return "dockerfile" }

// Match — Dockerfile-подобные имена.
func (strategy *Dockerfile) Match(filePath string) bool {
	base := strings.ToLower(filepath.Base(filePath))
	return base == "dockerfile" || base == "containerfile" ||
		strings.HasSuffix(base, ".dockerfile") ||
		strings.HasPrefix(base, "dockerfile.") ||
		strings.HasPrefix(base, "containerfile.")
}

// Группы регекса:
//
//	1. префикс с отступом и "FROM "
//	2. опциональные флаги "--platform=..."
//	3. сама ссылка на образ
//	4. остаток строки (например " AS builder")
var dockerfileFromRegexp = regexp.MustCompile(`(?im)^(\s*FROM\s+)((?:--platform=\S+\s+)*)(\S+)(.*)$`)
var dockerfileAsRegexp = regexp.MustCompile(`(?i)\s+AS\s+(\S+)`)

// Parse находит все FROM-строки.
//
// Перед регекс-сканом мы склеиваем line-continuations: последовательность
// "\\\n" в Dockerfile означает "продолжение строки на следующей". Чтобы
// "FROM \\\n  nginx:1.21" распознавалось как одна логическая инструкция,
// мы заменяем "\\\n[whitespace]*" на одиночный пробел — но НЕ в самом
// content, а только для скана. Найденные позиции потом мапятся обратно
// в оригинальные байты через карту смещений.
func (strategy *Dockerfile) Parse(filePath string, content []byte) ([]*Occurrence, error) {
	flat, mapping := joinLineContinuations(content)

	stageAliases := map[string]bool{}
	matches := dockerfileFromRegexp.FindAllSubmatchIndex(flat, -1)

	var occurrences []*Occurrence
	for _, match := range matches {
		flatImageStart, flatImageEnd := match[6], match[7]
		restString := string(flat[match[8]:match[9]])
		imageString := string(flat[flatImageStart:flatImageEnd])

		// Запоминаем алиас стадии, чтобы последующий FROM <alias> не
		// принимался за внешний образ.
		if asMatch := dockerfileAsRegexp.FindStringSubmatch(restString); asMatch != nil {
			stageAliases[asMatch[1]] = true
		}
		if stageAliases[imageString] || strings.EqualFold(imageString, "scratch") {
			continue
		}
		parsedRef, ok := tryParseImageRef(imageString)
		if !ok {
			continue
		}
		// Преобразуем оффсеты flat -> original.
		imageStart := mapping[flatImageStart]
		imageEnd := mapping[flatImageEnd]
		occurrences = append(occurrences, &Occurrence{
			Image:     parsedRef,
			File:      filePath,
			Line:      lineNumberAt(content, imageStart),
			Kind:      FullReference,
			StartByte: imageStart,
			EndByte:   imageEnd,
		})
	}

	sort.Slice(occurrences, func(left, right int) bool {
		return occurrences[left].StartByte < occurrences[right].StartByte
	})
	return occurrences, nil
}

// joinLineContinuations возвращает копию content без line-continuations
// ("\\\n[ \t]*" → один пробел) и карту mapping, где mapping[i] — это
// байтовое смещение в оригинальном content, соответствующее i-му байту
// в flat.
//
// Длина mapping = len(flat) + 1, последний элемент равен len(content).
// Это позволяет безопасно мапить flat-EndByte (включая позицию-за-концом).
func joinLineContinuations(content []byte) (flat []byte, mapping []int) {
	flat = make([]byte, 0, len(content))
	mapping = make([]int, 0, len(content)+1)
	for index := 0; index < len(content); index++ {
		// Если видим "\\\n" — пропускаем и обратный слэш, и newline,
		// заменяя их одним пробелом, плюс глотаем последующий whitespace
		// в начале следующей строки.
		if content[index] == '\\' && index+1 < len(content) && content[index+1] == '\n' {
			flat = append(flat, ' ')
			mapping = append(mapping, index)
			index += 1 // пропустить \n тоже
			// проглотить ведущие пробелы/табы следующей строки
			for index+1 < len(content) && (content[index+1] == ' ' || content[index+1] == '\t') {
				index++
			}
			continue
		}
		flat = append(flat, content[index])
		mapping = append(mapping, index)
	}
	mapping = append(mapping, len(content))
	return flat, mapping
}
