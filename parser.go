// Package parser находит ссылки на Docker-образы в произвольных файлах
// и позволяет точечно обновлять версию и дайджест, сохраняя исходное
// форматирование, комментарии и порядок ключей.
package parser

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"

	"github.com/example/docker-image-tool/image"
)

// FieldKind описывает, что именно покрывает диапазон [StartByte, EndByte)
// у Occurrence.
type FieldKind int

const (
	// FullReference — диапазон содержит всю ссылку "domain/name[:tag][@sha256:hex]".
	FullReference FieldKind = iota
	// TagOnly — диапазон содержит только тег (без двоеточия), как
	// отдельный YAML-узел в Helm-mapping или Kustomize.
	TagOnly
	// DigestOnly — диапазон содержит только дайджест "sha256:hex" как
	// отдельный YAML-узел.
	DigestOnly
)

// Occurrence — найденная ссылка на образ внутри файла.
//
// Image полностью разобран и удовлетворяет строгим требованиям image.Parse:
// есть явный Domain (с точкой/портом или localhost), и есть либо Version,
// либо SHA256.
type Occurrence struct {
	Image     image.Ref
	File      string
	Line      int
	Kind      FieldKind
	StartByte int
	EndByte   int
}

// Strategy — парсер одного формата файла.
type Strategy interface {
	Name() string
	Match(filePath string) bool
	Parse(filePath string, content []byte) ([]*Occurrence, error)
}

// CanUpdate проверяет правило апдейта версии: same major + (minor↑ или
// (minor=) и patch↑). Образы без Version (только sha256) обновляются
// всегда — сравнивать нечего.
func CanUpdate(currentVersion, newVersion *semver.Version) bool {
	if newVersion == nil {
		return false
	}
	if currentVersion == nil {
		// Образ без версии (только sha256) — раз пришёл новый sha256,
		// обновляем.
		return true
	}
	if currentVersion.Major() != newVersion.Major() {
		return false
	}
	if newVersion.Minor() > currentVersion.Minor() {
		return true
	}
	if newVersion.Minor() == currentVersion.Minor() && newVersion.Patch() > currentVersion.Patch() {
		return true
	}
	return false
}

// ApplyUpdates применяет обновление (newVersion, newSHA256) к Occurrence-ам,
// прошедшим CanUpdate. Возвращает новые байты и количество фактически
// обновлённых уникальных образов (по Image.Key()).
//
// Вхождения сортируются по убыванию StartByte, чтобы сдвиги от ранних
// замен не делали оффсеты соседних вхождений невалидными.
func ApplyUpdates(content []byte, occurrences []*Occurrence, newVersion *semver.Version, newSHA256 string) ([]byte, int, error) {
	if newVersion == nil {
		return nil, 0, errors.New("newVersion = nil")
	}

	// Отбираем только те вхождения, образ которых разрешён к обновлению.
	var eligible []*Occurrence
	for _, occurrence := range occurrences {
		if CanUpdate(occurrence.Image.Version, newVersion) {
			eligible = append(eligible, occurrence)
		}
	}
	sort.Slice(eligible, func(left, right int) bool {
		return eligible[left].StartByte > eligible[right].StartByte
	})

	updatedKeys := map[string]bool{}
	result := content
	for _, occurrence := range eligible {
		replacement, ok := buildReplacement(occurrence, newVersion, newSHA256)
		if !ok {
			continue
		}
		if occurrence.StartByte < 0 || occurrence.EndByte > len(result) || occurrence.StartByte > occurrence.EndByte {
			return nil, 0, fmt.Errorf("%s:%d: некорректный диапазон [%d:%d]",
				occurrence.File, occurrence.Line, occurrence.StartByte, occurrence.EndByte)
		}
		updated := make([]byte, 0, len(result)+len(replacement)-(occurrence.EndByte-occurrence.StartByte))
		updated = append(updated, result[:occurrence.StartByte]...)
		updated = append(updated, replacement...)
		updated = append(updated, result[occurrence.EndByte:]...)
		result = updated
		updatedKeys[occurrence.Image.Key()] = true
	}
	return result, len(updatedKeys), nil
}

// buildReplacement формирует текст подстановки. Возвращает ok=false для
// DigestOnly-вхождений без переданного newSHA256 (тогда вхождение
// пропускается без ошибки).
func buildReplacement(occurrence *Occurrence, newVersion *semver.Version, newSHA256 string) (string, bool) {
	switch occurrence.Kind {
	case FullReference:
		return occurrence.Image.With(newVersion, newSHA256).String(), true
	case TagOnly:
		return newVersion.Original(), true
	case DigestOnly:
		if newSHA256 == "" {
			return "", false
		}
		return "sha256:" + newSHA256, true
	default:
		return "", false
	}
}

// ImageGroup — все вхождения одного логического образа в одном файле.
// Используется при записи в БД: одна группа = одна запись в images.
type ImageGroup struct {
	Key         string
	Image       image.Ref
	Occurrences []*Occurrence
}

// GroupByImage группирует вхождения по Image.Key(). Группы отсортированы
// по Key для устойчивости вывода.
func GroupByImage(occurrences []*Occurrence) []ImageGroup {
	byKey := map[string]*ImageGroup{}
	for _, occurrence := range occurrences {
		key := occurrence.Image.Key()
		group, exists := byKey[key]
		if !exists {
			group = &ImageGroup{Key: key, Image: occurrence.Image}
			byKey[key] = group
		}
		group.Occurrences = append(group.Occurrences, occurrence)
	}
	groups := make([]ImageGroup, 0, len(byKey))
	for _, group := range byKey {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(left, right int) bool {
		return groups[left].Key < groups[right].Key
	})
	return groups
}
