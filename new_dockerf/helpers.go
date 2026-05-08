package parser

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/example/docker-image-tool/image"
)

func tryParseImageRef(rawValue string) (image.Ref, bool) {
	if rawValue == "" || looksTemplated(rawValue) {
		return image.Ref{}, false
	}
	parsed, err := image.Parse(rawValue)
	if err != nil {
		return image.Ref{}, false
	}
	return parsed, true
}

func looksTemplated(text string) bool {
	return strings.Contains(text, "{{") ||
		strings.Contains(text, "${") ||
		strings.Contains(text, "$(")
}

func lineNumberAt(content []byte, byteOffset int) int {
	if byteOffset > len(content) {
		byteOffset = len(content)
	}
	return 1 + bytes.Count(content[:byteOffset], []byte{'\n'})
}

func lineColumnToByteOffset(content []byte, line, column int) int {
	if line < 1 {
		return 0
	}
	currentLine, currentColumn := 1, 1
	for index := 0; index < len(content); index++ {
		if currentLine == line && currentColumn == column {
			return index
		}
		if content[index] == '\n' {
			currentLine++
			currentColumn = 1
		} else {
			currentColumn++
		}
	}
	return len(content)
}

func findValueBytes(content []byte, line, column int, value string) (start, end int, found bool) {
	if value == "" {
		return 0, 0, false
	}
	base := lineColumnToByteOffset(content, line, column)
	if base >= len(content) {
		return 0, 0, false
	}
	lineEnd := base + bytes.IndexByte(content[base:], '\n')
	if lineEnd < base {
		lineEnd = len(content)
	}
	relative := bytes.Index(content[base:lineEnd], []byte(value))
	if relative < 0 {
		return 0, 0, false
	}
	return base + relative, base + relative + len(value), true
}

// findValueBytesInRange ищет value в диапазоне строк [startLine, endLine].
// Используется для FROM с line-continuation, где образ может быть на строке
// отличной от строки самой инструкции FROM.
func findValueBytesInRange(content []byte, startLine, endLine int, value string) (start, end int, found bool) {
	if value == "" {
		return 0, 0, false
	}
	base := lineColumnToByteOffset(content, startLine, 1)
	// Найти конец последней строки диапазона.
	rangeEnd := lineColumnToByteOffset(content, endLine, 1)
	if nlIdx := bytes.IndexByte(content[rangeEnd:], '\n'); nlIdx >= 0 {
		rangeEnd += nlIdx
	} else {
		rangeEnd = len(content)
	}
	relative := bytes.Index(content[base:rangeEnd], []byte(value))
	if relative < 0 {
		return 0, 0, false
	}
	return base + relative, base + relative + len(value), true
}

func scanByRegex(
	filePath string,
	originalContent []byte,
	scanContent []byte,
	pattern *regexp.Regexp,
	captureIndex int,
	skipPredicate func(line []byte) bool,
) []*Occurrence {
	if len(scanContent) != len(originalContent) {
		panic("scanByRegex: длины scanContent и originalContent не совпадают")
	}
	matches := pattern.FindAllSubmatchIndex(scanContent, -1)
	var occurrences []*Occurrence
	for _, match := range matches {
		captureStart := match[2*captureIndex]
		captureEnd := match[2*captureIndex+1]
		if captureStart < 0 {
			continue
		}
		if skipPredicate != nil && skipPredicate(lineAt(originalContent, captureStart)) {
			continue
		}
		rawValue := string(originalContent[captureStart:captureEnd])
		parsedRef, ok := tryParseImageRef(rawValue)
		if !ok {
			continue
		}
		occurrences = append(occurrences, &Occurrence{
			Image:     parsedRef,
			File:      filePath,
			Line:      lineNumberAt(originalContent, captureStart),
			Kind:      FullReference,
			StartByte: captureStart,
			EndByte:   captureEnd,
		})
	}
	return occurrences
}

func lineAt(content []byte, offset int) []byte {
	lineStart := bytes.LastIndexByte(content[:offset], '\n') + 1
	lineEnd := offset + bytes.IndexByte(content[offset:], '\n')
	if lineEnd < offset {
		lineEnd = len(content)
	}
	return content[lineStart:lineEnd]
}

func commentLineSkip(line []byte) bool {
	trimmed := bytes.TrimLeft(line, " \t")
	if len(trimmed) == 0 || trimmed[0] == '#' {
		return true
	}
	return len(trimmed) >= 2 && trimmed[0] == '/' && trimmed[1] == '/'
}

func looksImagedAssignment(keyName, rawValue string) bool {
	keyUpper := strings.ToUpper(keyName)
	if strings.Contains(keyUpper, "IMAGE") ||
		strings.Contains(keyUpper, "DOCKER") ||
		strings.Contains(keyUpper, "CONTAINER") {
		return true
	}
	return strings.ContainsAny(rawValue, ":/")
}
