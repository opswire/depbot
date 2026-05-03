package depbot

import (
	"regexp"
)

// Generic — fallback-стратегия для любого текстового файла. Запускается
// последней, когда ни одна специализированная стратегия не подошла.
//
// Идея: пробежаться регексом по всем подстрокам, состоящим из символов,
// допустимых в Docker-ссылке (буквы, цифры, ".", "_", "-", "/", ":", "@"),
// и для каждого кандидата позвать tryParseImageRef.
//
// Дополнительные защиты:
//   - кандидат должен содержать ":" или "@" (без них одиночное слово
//     нормализуется distribution/reference как "library/<word>", и
//     tryParseImageRef отсекает его требованием Version != nil);
//   - закомментированные строки (# или //) пропускаются;
//   - токены, у которых тег — чистое число (вид "host:port"), отсекаются:
//     это почти всегда host:port, а не образ. Distribution/reference
//     принимает такой тег как валидный, поэтому проверку делаем здесь.
//
// Generic разрешает только пары "image:tag" — образы без тега или с
// non-semver тегом отбрасываются tryParseImageRef.
type Generic struct{}

// Name возвращает имя стратегии.
func (strategy *Generic) Name() string { return "generic" }

// Match всегда true: Generic — fallback. Реальный приоритет задаётся
// порядком регистрации в Registry — Generic ставится последней.
func (strategy *Generic) Match(filePath string) bool { return true }

// genericTokenRegexp матчит подряд идущие "Docker-валидные" символы,
// внутри которых есть хотя бы один ":" или "@" — это эвристический
// признак ссылки. Группа 1 — сам токен.
var genericTokenRegexp = regexp.MustCompile(
	`([A-Za-z0-9][A-Za-z0-9._\-/]*[:@][A-Za-z0-9._\-/:@]+)`)

// hostPortRegexp совпадает с токенами вида "host[.domain]:NNNN" (порт),
// которые distribution/reference распарсит как "library/host:NNNN".
// Признак: единственный ":" в строке, и после него только цифры.
var hostPortRegexp = regexp.MustCompile(`^[A-Za-z0-9.\-]+:[0-9]+$`)

// Parse находит все токены, которые tryParseImageRef признаёт ссылками.
func (strategy *Generic) Parse(filePath string, content []byte) ([]*Occurrence, error) {
	occurrences := scanByRegex(filePath, content, content, genericTokenRegexp, 1, commentLineSkip)
	// Доп. фильтр: убираем host:port — для них port = pure-numeric tag,
	// и distribution/reference принимает их как library/host с числовым тегом.
	filtered := occurrences[:0]
	for _, occurrence := range occurrences {
		if hostPortRegexp.MatchString(occurrence.Image.SourceName + ":" + occurrence.Image.Version.Original()) {
			continue
		}
		filtered = append(filtered, occurrence)
	}
	return filtered, nil
}
