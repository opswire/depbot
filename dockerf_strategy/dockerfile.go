package parser

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// Dockerfile разбирает Dockerfile / Containerfile / *.Dockerfile / Dockerfile.*
// с помощью официального парсера moby/buildkit.
//
// Распознаются формы:
//
//	FROM image[:tag][@sha256:...]
//	FROM --platform=... image[:tag] [AS name]
//
// Сознательно пропускаются:
//   - "scratch" (magic-base);
//   - шаблонные основы вида "FROM ${BASE}";
//   - FROM, ссылающиеся на ранее объявленную стадию (по алиасу).
type Dockerfile struct{}

// Name возвращает имя стратегии.
func (strategy *Dockerfile) Name() string { return "dockerfile" }

// Match — Dockerfile-подобные имена файлов.
func (strategy *Dockerfile) Match(filePath string) bool {
	base := strings.ToLower(filepath.Base(filePath))
	return base == "dockerfile" ||
		base == "containerfile" ||
		strings.HasSuffix(base, ".dockerfile") ||
		strings.HasPrefix(base, "dockerfile.") ||
		strings.HasPrefix(base, "containerfile.")
}

// Parse находит все FROM-инструкции, возвращая Occurrence для каждого
// валидного внешнего образа.
//
// Парсинг делегируется github.com/moby/buildkit/frontend/dockerfile/parser,
// который корректно обрабатывает:
//   - line-continuations (\\\n);
//   - escape-директиву (# escape=`);
//   - комментарии;
//   - BuildKit-синтаксис (#syntax=…).
//
// После получения AST-узлов мы:
//  1. Обходим только инструкции FROM (node.Value == "from").
//  2. Пропускаем флаги (--platform=…) — они идут первыми дочерними
//     узлами с флагом Next/Flags.
//  3. Берём первый «настоящий» аргумент — имя образа.
//  4. Пропускаем scratch, шаблоны и алиасы стадий.
//  5. Запоминаем алиасы (AS name) для правила 4.
//  6. Для точного StartByte/EndByte ищем вхождение строки-образа
//     внутри оригинального content через findValueBytes.
func (strategy *Dockerfile) Parse(filePath string, content []byte) ([]*Occurrence, error) {
	result, err := parser.Parse(bytes.NewReader(content))
	if err != nil {
		// buildkit бросает "file with no instructions" на пустых файлах — это не ошибка для нас.
		if isEmptyFileError(err) {
			return nil, nil
		}
		return nil, err
	}

	stageAliases := map[string]bool{}
	var occurrences []*Occurrence

	for _, node := range result.AST.Children {
		if !strings.EqualFold(node.Value, "from") {
			continue
		}

		imageNode, alias := extractFromArgs(node)
		if imageNode == nil {
			continue
		}

		imageStr := imageNode.Value

		// Запомним алиас до проверок — даже если сам образ пропускаем,
		// алиас влияет на следующие стадии.
		if alias != "" {
			stageAliases[strings.ToLower(alias)] = true
		}

		if stageAliases[strings.ToLower(imageStr)] {
			continue
		}
		if strings.EqualFold(imageStr, "scratch") {
			continue
		}

		parsedRef, ok := tryParseImageRef(imageStr)
		if !ok {
			continue
		}

		// Buildkit дает 1-based StartLine для самого FROM-узла, но у его
		// дочерних аргументных узлов StartLine=0. Используем node.StartLine
		// (строку FROM-инструкции) для поиска байтового положения образа.
		fromLine := node.StartLine
		start, end, found := findValueBytes(content, fromLine, 1, imageStr)
		if !found {
			// Фолбэк: берём начало строки FROM.
			start = lineColumnToByteOffset(content, fromLine, 1)
			end = start + len(imageStr)
		}

		occurrences = append(occurrences, &Occurrence{
			Image:     parsedRef,
			File:      filePath,
			Line:      fromLine,
			Kind:      FullReference,
			StartByte: start,
			EndByte:   end,
		})
	}

	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].StartByte < occurrences[j].StartByte
	})
	return occurrences, nil
}

// extractFromArgs возвращает AST-узел с именем образа и алиас стадии (или "").
//
// Грамматика FROM-инструкции в AST buildkit:
//
//	node.Next  →  [--flag …]* image_name [AS alias]
//
// Флаги имеют вид "--platform=linux/amd64" и начинаются с "--".
// После флагов идёт имя образа, затем опционально токены "AS" и алиас.
func extractFromArgs(node *parser.Node) (imageNode *parser.Node, alias string) {
	cur := node.Next
	// Пропустить флаги
	for cur != nil && strings.HasPrefix(cur.Value, "--") {
		cur = cur.Next
	}
	if cur == nil {
		return nil, ""
	}
	imageNode = cur
	// Проверить наличие AS <alias>
	asNode := cur.Next
	if asNode != nil && strings.EqualFold(asNode.Value, "as") {
		if asNode.Next != nil {
			alias = asNode.Next.Value
		}
	}
	return imageNode, alias
}

// isEmptyFileError проверяет, что ошибка от buildkit-парсера означает
// «файл без инструкций» (пустой файл или только комментарии) —
// это штатная ситуация, не ошибка для нашего парсера.
func isEmptyFileError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "file with no instructions")
}
