# docker-image-tool

Утилита и Go-библиотека для поиска ссылок на Docker-образы в произвольных
файлах репозитория и точечного обновления версии (и опционально дайджеста)
без потери форматирования.

Архитектура заточена под dependbot-сценарий: модель данных группируется
по `(domain, name, version)` и ложится в БД-таблицу `images`, файлы — в
`dockerfiles`, связь — в `dockerfile_image`.

## Что считается валидной ссылкой на образ

`image.Parse` принимает только полные ссылки. Все более слабые формы
отвергаются — они слишком часто ловятся как false positive в исходном
коде, документации и логах.

| Форма | Принимается |
| --- | :---: |
| `domain.tld/name:semver` | ✓ |
| `domain.tld/name:semver@sha256:hex` | ✓ |
| `domain.tld/name@sha256:hex` (без тега) | ✓ |
| `domain.tld:port/name:semver` | ✓ |
| `localhost/name:semver` | ✓ |
| `nginx:1.21` (short-form) | ✗ нет явного домена |
| `myorg/app:1.21` | ✗ `myorg` не выглядит как домен (нет точки/порта) |
| `domain.io/name:latest` | ✗ тег не semver |
| `domain.io/name` (без тега и дайджеста) | ✗ нечего обновлять |
| `db.example.com:5432` (host:port) | ✗ нет `/name` после домена |

Доменом считается строка, содержащая `.` или `:` (порт), либо равная
`localhost`. Тег обязан разбираться `Masterminds/semver/v3` (включая
формы `v1.2.3`, `1.21-alpine`).

## Правило обновления

`CanUpdate(currentVersion, newVersion)` возвращает `true` только при:

* same major;
* `newVersion.Minor > current.Minor` ИЛИ `(Minor=) AND (newVersion.Patch > current.Patch)`.

Образы без версии (только sha256) обновляются всегда — сравнивать нечего.

## Поддерживаемые стратегии

Стратегии перебираются по порядку, первая прошедшая `Match` забирает файл
себе. Generic стоит последним — это fallback для произвольных текстовых
файлов из whitelist расширений.

| # | Стратегия | Файлы | Основа парсинга |
| :- | --- | --- | --- |
| 1 | **Dockerfile** | `Dockerfile`, `Containerfile`, `*.Dockerfile`, `Dockerfile.*` | [`github.com/moby/buildkit/frontend/dockerfile/parser`](https://pkg.go.dev/github.com/moby/buildkit/frontend/dockerfile/parser) |
| 2 | **PomXML** | `pom.xml` | `encoding/xml` с отслеживанием `Decoder.InputOffset()` |
| 3 | **YAML** | `*.yaml`, `*.yml` | `gopkg.in/yaml.v3` (узловой AST с позициями) |
| 4 | **JSON / JSONC** | `*.json`, `*.jsonc` | `tidwall/jsonc.ToJSON` для стриппа комментариев + регекс по очищенной копии |
| 5 | **HCL** | `*.tf`, `*.hcl`, `*.nomad` (исключая `*.tf.json`) | регекс `^\s*image\s*=\s*"..."` |
| 6 | **TOML** | `*.toml` | тот же регекс что HCL |
| 7 | **Starlark** | `Tiltfile`, `BUILD`, `BUILD.bazel`, `WORKSPACE`, `*.bzl` | регекс по строковым литералам |
| 8 | **Generic** | `.md`, `.txt`, `.rst`, `.adoc`, `.log`, `.conf`, `.cfg`, `.ini`, `.properties`, `.env`, `.env.*`, `*.env` | регекс по docker-валидным токенам |

### Особенности по стратегиям

**Dockerfile.** Через AST buildkit'а. Корректно обрабатывает:
line-continuations (`\<newline>`), `--platform=` флаги, AS-алиасы стадий
(включая mixed-case `as`), `# escape=` директивы. Пропускает `FROM scratch`,
`FROM ${ARG}` (шаблонные основы), `FROM <stage-alias>`.

**PomXML.** Поддерживает обе формы: jib-maven-plugin (`<from><image>`,
`<to><image>`) и fabric8 docker-maven-plugin (`<images><image><name>`).

**YAML.** Распознаёт:

1. scalar `image: domain/name:tag`;
2. Helm-mapping `image: { registry, repository, tag, digest }` — порождает
   до двух Occurrence одной группы (`TagOnly` + `DigestOnly`);
3. GitHub Actions `uses: docker://name:tag`.

Поддерживаются мультидокументные YAML-файлы (`---` разделители).
Шаблоны `{{ ... }}` пропускаются, не разламывая парсинг.

**JSON.** Сканирует пары `"image": "..."`. Комментарии `//` и `/* */`
заменяются на пробелы через `tidwall/jsonc.ToJSON` с сохранением байтовых
оффсетов, поэтому позиции в очищенной копии напрямую применимы к оригиналу.

**HCL/TOML.** Из-за совпадения нужной нам части грамматики (`image = "..."`
в начале строки) реализованы общим регексом. Закомментированные строки
(`#` или `//`) пропускаются.

**Starlark.** Принимает только полные ссылки в строковых литералах
(`docker_build('domain/name:tag', ...)`). Изолированные `image="..."` без
тега в той же строке игнорируются — в Bazel они часто разнесены по
разным аргументам функции (`oci_pull(image=..., tag=...)`).

**Generic.** Работает по whitelist расширений, где имеет смысл искать
ссылки на образы fallback-парсером. Включает `.env`-подобные форматы:
`KEY=value` присваивания распознаются как обычные scalar-ссылки. Регекс
ищет токены вида `domain/name:tag`; коротких форм (`nginx:1.21`) и
`host:port` он не поймает, потому что строгий `image.Parse` их отвергает.

## Использование как CLI

```bash
# Поиск (группированный вывод по logical image):
go run ./cmd/scan -root ./repo

# JSON-выход для записи в БД:
go run ./cmd/scan -root ./repo -json

# Обновление до 3.99.0 + новый sha256.
# Применится только к образам с major=3.
go run ./cmd/scan -root ./repo -update 3.99.0 \
    -sha256 abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789
```

Пример JSON-вывода (структура соответствует БД-схеме):

```json
[
  {
    "key": "docker.io/library/alpine:3.18",
    "domain": "docker.io",
    "name": "library/alpine",
    "version": "3.18",
    "occurrences": [
      { "file": "Dockerfile",   "line": 1, "raw": "docker.io/library/alpine:3.18" },
      { "file": "values.yaml",  "line": 5, "raw": "docker.io/library/alpine:3.18" }
    ]
  }
]
```

## Использование как библиотеки

```go
package main

import (
	"fmt"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/example/docker-image-tool/parser"
)

func main() {
	registry := parser.NewDefaultRegistry()

	content, _ := os.ReadFile("deployment.yaml")
	occurrences, _ := registry.Parse("deployment.yaml", content)

	// Группировка для записи в БД.
	for _, group := range parser.GroupByImage(occurrences) {
		fmt.Printf("%s (%d вхождений)\n", group.Key, len(group.Occurrences))
		// group.Image.Domain / Name / Version.Original() / SHA256 — поля
		// для INSERT INTO images ... ON CONFLICT DO NOTHING.
	}

	// Применить апдейт ко всем вхождениям в одном файле.
	// CanUpdate-фильтр применяется автоматически: образы вне same-major
	// или с понижением версии останутся нетронутыми.
	newVersion := semver.MustParse("3.99.0")
	newSHA256 := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	updated, count, err := parser.ApplyUpdates(content, occurrences, newVersion, newSHA256)
	if err != nil {
		panic(err)
	}
	fmt.Printf("обновлено %d уникальных образов\n", count)
	_ = os.WriteFile("deployment.yaml", updated, 0o644)
}
```

## API в двух словах

```go
// image/image.go
type Ref struct {
    Domain  string  // "docker.io", "gcr.io", "registry.example.com:5000"
    Name    string  // "library/nginx", "myorg/app"
    Version *semver.Version
    SHA256  string  // hex без префикса "sha256:"
}

func Parse(rawReference string) (Ref, error)
func (imageRef Ref) String() string             // "domain/name:tag@sha256:hex"
func (imageRef Ref) Key() string                // "domain/name:version" или "domain/name@sha256:..."
func (imageRef Ref) With(*semver.Version, string) Ref

// parser/parser.go
type FieldKind int
const (
    FullReference FieldKind = iota
    TagOnly
    DigestOnly
)

type Occurrence struct {
    Image     image.Ref
    File      string
    Line      int
    Kind      FieldKind
    StartByte int
    EndByte   int
}

type Strategy interface {
    Name() string
    Match(filePath string) bool
    Parse(filePath string, content []byte) ([]*Occurrence, error)
}

func CanUpdate(currentVersion, newVersion *semver.Version) bool
func ApplyUpdates(content []byte, occurrences []*Occurrence, newVersion *semver.Version, newSHA256 string) ([]byte, int, error)

type ImageGroup struct {
    Key         string
    Image       image.Ref
    Occurrences []*Occurrence
}

func GroupByImage(occurrences []*Occurrence) []ImageGroup
```

`Update` и `ApplyUpdates` возвращают новый байтовый контент и **не**
трогают диск — запись файла остаётся за вызывающим кодом.

## Соответствие БД-схеме

```
images:           (domain, name, version, sha256)   ← одна строка на ImageGroup
dockerfiles:      (path)                            ← одна строка на уникальный Occurrence.File
dockerfile_image: (dockerfile_id, image_id)         ← одна строка на Occurrence
```

`ImageGroup.Key()` = `"domain/name:version"` (или `"domain/name@sha256:..."`
для образов без версии) — стабильный естественный ключ для UPSERT.

## Семантика обновления `(newVersion, newSHA256)`

* **scalar-форматы** (Dockerfile, scalar YAML, JSON, HCL, TOML, Generic,
  Starlark): вся ссылка переписывается через `Ref.String()` с обновлёнными
  полями. Старый дайджест убирается всегда; новый вставляется, если
  передан непустой `newSHA256`.
* **Helm-mapping `image: {tag, digest}`** в YAML: tag-узел и digest-узел
  дают **отдельные** Occurrence одной группы. Tag обновляется на
  `newVersion.Original()`. Digest обновляется на `sha256:<newSHA256>`,
  если переданный непустой; если пустой — узел не трогается.
* Если digest-узла **не было** в источнике, он **не вставляется** —
  это бы поломало YAML-форматирование. Хотите всегда обновлять digest —
  добавьте узел вручную, следующий update подхватит.

## Зависимости

* `github.com/Masterminds/semver/v3` — semver для поля `Version`.
* `github.com/moby/buildkit/frontend/dockerfile/parser` — AST Dockerfile
  (тот же, что использует `docker build`).
* `gopkg.in/yaml.v3` — YAML с node-API и позициями.
* `github.com/tidwall/jsonc` — стриппер JSONC-комментов.
* `github.com/stretchr/testify` (test only) — testify/suite.

`distribution/reference` сознательно НЕ используется: его автоматическая
нормализация short-form (`nginx → docker.io/library/nginx`) противоречит
нашему требованию "только явный домен".

## Граничные случаи

* **Шаблонные значения** (`{{ .x }}`, `${VAR}`, `$(VAR)`) автоматически
  отбрасываются.
* **Helm-mapping без tag-узла** (только `repository: nginx`) — Occurrence
  не порождается (нечего обновлять без semver).
* **YAML block scalar** (`image: |<newline> nginx:1.21`) не парсится
  стратегией — это редкая форма, для image-ссылки бессмысленна.
* **Bazel split args** (`oci_pull(image="domain/name", tag="1.21")`) —
  Starlark не подбирает, потому что в одном литерале нет тега. Это
  компромисс ради надёжности: иначе пришлось бы парсить вызовы функций.
* **Pom XML с комментарием внутри `<image>`-узла** (например
  `<image>nginx<!-- c -->:1.21</image>`) — байтовый диапазон может уйти,
  потому что CharData события склеиваются. На практике не встречается.
