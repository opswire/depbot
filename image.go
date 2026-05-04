// Package image предоставляет строгий разбор Docker-ссылок без
// нормализации.
//
// В отличие от distribution/reference (который автоматически нормализует
// short-form "nginx:1.21" в "docker.io/library/nginx:1.21"), здесь
// нормализация ОТКЛЮЧЕНА: short-form считается невалидной ссылкой,
// потому что для dependbot-сценария он почти всегда — false positive
// (часть исходного кода, документации или конфига, не настоящий образ).
//
// Дополнительные требования к валидной ссылке:
//   - явно указан Domain (с точкой/портом — иначе не признаётся доменом);
//   - есть либо semver-совместимая Version, либо SHA256.
package image

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Ref — разобранная ссылка на Docker-образ.
//
// Domain   — registry с точкой или портом, например "docker.io",
// "registry.example.com:5000" или "gcr.io". Не может быть пустым у Ref,
// возвращённого Parse.
// Name     — путь внутри registry, например "myorg/app". Никогда не
// содержит префикс "library/" (он добавляется только при нормализации,
// которой здесь нет).
// Version  — тег как semver. nil допустим, если задан SHA256.
// SHA256   — hex-дайджест.
type Ref struct {
	Domain  string
	Name    string
	Version *semver.Version
	SHA256  string
}

// ErrInvalidReference означает, что строка не является валидной
// Docker-ссылкой по нашим (более строгим, чем у distribution/reference)
// критериям: либо нет Domain, либо нет ни Version, ни SHA256, либо тег
// не разбирается как semver.
var ErrInvalidReference = errors.New("невалидная ссылка на образ")

// Parse разбирает строку Docker-ссылки.
//
// Принимаются формы:
//
//	domain.tld/name:semver
//	domain.tld/name:semver@sha256:...
//	domain.tld/name@sha256:...
//	domain.tld:port/name:semver[...]
//
// Отвергаются:
//
//	nginx               — short-form, без явного домена
//	nginx:1.21          — short-form, без явного домена
//	myorg/app:1.21      — без явного домена (myorg не выглядит как домен)
//	domain.io/n:latest  — non-semver тег и нет sha256
//	domain.io/name      — нет ни тега, ни дайджеста
//
// Возвращает ErrInvalidReference во всех этих случаях.
func Parse(rawReference string) (Ref, error) {
	if rawReference == "" {
		return Ref{}, fmt.Errorf("%w: пустая строка", ErrInvalidReference)
	}

	// Отделяем дайджест.
	rest, sha256 := splitDigest(rawReference)
	if sha256 == "" && strings.Contains(rest, "@") {
		// Был "@sha512:..." или другой алгоритм — мы такой не принимаем.
		return Ref{}, fmt.Errorf("%w: неподдерживаемый дайджест", ErrInvalidReference)
	}

	// Отделяем тег.
	namePart, tag := splitTag(rest)

	// Нужен хотя бы один разделитель "/" — то есть минимум "domain/name".
	slashIndex := strings.Index(namePart, "/")
	if slashIndex < 0 {
		return Ref{}, fmt.Errorf("%w: нет разделителя domain/name", ErrInvalidReference)
	}
	domain := namePart[:slashIndex]
	name := namePart[slashIndex+1:]

	// Domain должен выглядеть как домен: содержать "." (FQDN/субдомен) или
	// ":" (порт), либо быть localhost. Иначе это просто <user>/<repo>
	// short-form Docker Hub.
	if !looksLikeDomain(domain) {
		return Ref{}, fmt.Errorf("%w: %q не выглядит как domain", ErrInvalidReference, domain)
	}
	if name == "" {
		return Ref{}, fmt.Errorf("%w: пустой name", ErrInvalidReference)
	}
	if !looksLikeName(name) {
		return Ref{}, fmt.Errorf("%w: невалидный name %q", ErrInvalidReference, name)
	}

	result := Ref{Domain: domain, Name: name, SHA256: sha256}

	if tag != "" {
		version, err := semver.NewVersion(tag)
		if err != nil {
			return Ref{}, fmt.Errorf("%w: тег %q не semver", ErrInvalidReference, tag)
		}
		result.Version = version
	}

	// Нужна хотя бы версия или дайджест.
	if result.Version == nil && result.SHA256 == "" {
		return Ref{}, fmt.Errorf("%w: нужна semver-версия или sha256", ErrInvalidReference)
	}
	return result, nil
}

// splitDigest вырезает "@sha256:hex" из конца. Если алгоритм не sha256
// (например sha512), возвращается ("", ""). Если "@" нет, дайджест "".
func splitDigest(text string) (rest, sha256 string) {
	atIndex := strings.LastIndex(text, "@")
	if atIndex < 0 {
		return text, ""
	}
	digestPart := text[atIndex+1:]
	const prefix = "sha256:"
	if !strings.HasPrefix(digestPart, prefix) {
		return text, ""
	}
	hex := digestPart[len(prefix):]
	if !isHex(hex) || len(hex) != 64 {
		return text, ""
	}
	return text[:atIndex], hex
}

// splitTag отделяет ":tag" в конце имени. Внимательно с registry:port —
// двоеточие в порте идёт ДО первого "/" (т.е. до начала name), а двоеточие
// тега всегда ПОСЛЕ последнего "/". Поэтому ищем последнее ":" и проверяем,
// что после него нет "/".
func splitTag(text string) (name, tag string) {
	colonIndex := strings.LastIndex(text, ":")
	if colonIndex < 0 {
		return text, ""
	}
	if strings.Contains(text[colonIndex:], "/") {
		// ":" оказалось внутри domain:port, а не разделителем тега.
		return text, ""
	}
	return text[:colonIndex], text[colonIndex+1:]
}

// looksLikeDomain эвристически определяет, что строка — это domain
// или host:port. Признак: содержит "." или ":" или равна "localhost".
func looksLikeDomain(text string) bool {
	if text == "localhost" {
		return true
	}
	return strings.ContainsAny(text, ".:")
}

// looksLikeName проверяет, что name синтаксически допустим как путь
// репозитория Docker-образа: разрешены [a-z0-9._-/], не пустой, не
// начинается и не заканчивается на разделитель.
func looksLikeName(text string) bool {
	if text == "" {
		return false
	}
	for _, value := range text {
		switch {
		case value >= 'a' && value <= 'z':
		case value >= '0' && value <= '9':
		case value == '.' || value == '_' || value == '-' || value == '/':
		default:
			return false
		}
	}
	if text[0] == '/' || text[len(text)-1] == '/' {
		return false
	}
	return true
}

// isHex проверяет, что строка состоит только из hex-символов.
func isHex(text string) bool {
	for _, value := range text {
		switch {
		case value >= '0' && value <= '9':
		case value >= 'a' && value <= 'f':
		case value >= 'A' && value <= 'F':
		default:
			return false
		}
	}
	return text != ""
}

// String возвращает каноничную форму "domain/name[:tag][@sha256:hex]".
func (imageRef Ref) String() string {
	var builder strings.Builder
	builder.WriteString(imageRef.Domain)
	builder.WriteString("/")
	builder.WriteString(imageRef.Name)
	if imageRef.Version != nil {
		builder.WriteString(":")
		builder.WriteString(imageRef.Version.Original())
	}
	if imageRef.SHA256 != "" {
		builder.WriteString("@sha256:")
		builder.WriteString(imageRef.SHA256)
	}
	return builder.String()
}

// Key возвращает стабильный идентификатор: "domain/name:version" или
// "domain/name@sha256:..." (если версии нет, но есть дайджест).
func (imageRef Ref) Key() string {
	prefix := imageRef.Domain + "/" + imageRef.Name
	if imageRef.Version != nil {
		return prefix + ":" + imageRef.Version.Original()
	}
	return prefix + "@sha256:" + imageRef.SHA256
}

// With возвращает копию Ref с новой версией и новым дайджестом.
func (imageRef Ref) With(newVersion *semver.Version, newSHA256 string) Ref {
	imageRef.Version = newVersion
	imageRef.SHA256 = newSHA256
	return imageRef
}
