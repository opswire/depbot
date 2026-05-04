// Команда scan рекурсивно обходит каталог, ищет ссылки на Docker-образы
// и при необходимости обновляет их версию (и опционально дайджест) через
// parser.ApplyUpdates.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver/v3"

	"github.com/example/docker-image-tool/parser"
)

// Каталоги, которые сканировать заведомо не нужно: они полны автогенерата
// и сборочных артефактов.
var skippedDirectoryNames = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".idea": true, ".vscode": true,
	".terraform": true, "node_modules": true, "vendor": true, "target": true,
	"dist": true, "build": true, ".venv": true, "__pycache__": true,
}

// reportOccurrence — JSON-форма одного вхождения.
type reportOccurrence struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Raw    string `json:"raw"`
	SHA256 string `json:"sha256,omitempty"`
}

// reportGroup — JSON-форма одной записи в таблице images.
type reportGroup struct {
	Key         string             `json:"key"`
	Domain      string             `json:"domain,omitempty"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Occurrences []reportOccurrence `json:"occurrences"`
}

func main() {
	rootPath := flag.String("root", ".", "корневой каталог для рекурсивного сканирования")
	updateTo := flag.String("update", "", "если задано — заменить версию во всех найденных образах на эту semver-строку")
	updateSHA256 := flag.String("sha256", "", "новый sha256-дайджест (hex без префикса), применяется вместе с -update")
	jsonOutput := flag.Bool("json", false, "вывести результаты в JSON")
	flag.Parse()

	var newVersion *semver.Version
	if *updateTo != "" {
		parsedVersion, err := semver.NewVersion(*updateTo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "некорректная semver-версия %q: %v\n", *updateTo, err)
			os.Exit(2)
		}
		newVersion = parsedVersion
	}

	registry := parser.NewDefaultRegistry()
	var allOccurrences []*parser.Occurrence

	walkErr := filepath.WalkDir(*rootPath, func(currentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", currentPath, err)
			return nil
		}
		if dirEntry.IsDir() {
			if currentPath != *rootPath && skippedDirectoryNames[dirEntry.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if dirEntry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		content, readErr := os.ReadFile(currentPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", currentPath, readErr)
			return nil
		}

		// Бинарники отсекаем по нулевому байту в первых 8KiB.
		probe := content
		if len(probe) > 8192 {
			probe = probe[:8192]
		}
		if bytes.IndexByte(probe, 0) >= 0 {
			return nil
		}

		occurrences, parseErr := registry.Parse(currentPath, content)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", currentPath, parseErr)
			return nil
		}
		if len(occurrences) == 0 {
			return nil
		}

		if newVersion != nil {
			updatedContent, _, applyErr := parser.ApplyUpdates(content, occurrences, newVersion, *updateSHA256)
			if applyErr != nil {
				fmt.Fprintf(os.Stderr, "ошибка обновления %s: %v\n", currentPath, applyErr)
			} else if !bytes.Equal(content, updatedContent) {
				if writeErr := os.WriteFile(currentPath, updatedContent, 0o644); writeErr != nil {
					fmt.Fprintf(os.Stderr, "не удалось записать %s: %v\n", currentPath, writeErr)
				}
			}
		}
		allOccurrences = append(allOccurrences, occurrences...)
		return nil
	})
	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "ошибка обхода: %v\n", walkErr)
		os.Exit(1)
	}

	groups := parser.GroupByImage(allOccurrences)

	if *jsonOutput {
		report := buildJSONReport(groups)
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "ошибка JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	for _, group := range groups {
		fmt.Println(group.Key)
		sortOccurrences(group.Occurrences)
		for _, occurrence := range group.Occurrences {
			detail := occurrence.Image.String()
			fmt.Printf("    %s:%d %s\n", occurrence.File, occurrence.Line, detail)
		}
	}

	switch {
	case len(groups) == 0:
		fmt.Fprintln(os.Stderr, "ничего не найдено")
	case newVersion != nil:
		message := fmt.Sprintf("\nобновлено до %s", newVersion.Original())
		if *updateSHA256 != "" {
			message += " @sha256:" + *updateSHA256
		}
		fmt.Fprintf(os.Stderr, "%s: %d группа(ы), %d вхождение(й)\n", message, len(groups), len(allOccurrences))
	}
}

// buildJSONReport преобразует группы в JSON-структуру.
func buildJSONReport(groups []parser.ImageGroup) []reportGroup {
	report := make([]reportGroup, 0, len(groups))
	for _, group := range groups {
		entry := reportGroup{
			Key:    group.Key,
			Domain: group.Image.Domain,
			Name:   group.Image.Name,
		}
		if group.Image.Version != nil {
			entry.Version = group.Image.Version.Original()
		}
		sortOccurrences(group.Occurrences)
		for _, occurrence := range group.Occurrences {
			entry.Occurrences = append(entry.Occurrences, reportOccurrence{
				File:   occurrence.File,
				Line:   occurrence.Line,
				Raw:    occurrence.Image.String(),
				SHA256: occurrence.Image.SHA256,
			})
		}
		report = append(report, entry)
	}
	return report
}

// sortOccurrences стабильно сортирует вхождения по (file, line).
func sortOccurrences(occurrences []*parser.Occurrence) {
	sort.Slice(occurrences, func(left, right int) bool {
		if occurrences[left].File != occurrences[right].File {
			return occurrences[left].File < occurrences[right].File
		}
		return occurrences[left].Line < occurrences[right].Line
	})
}
