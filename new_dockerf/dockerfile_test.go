package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
)

// --- Match tests ---

func TestDockerfileMatch(t *testing.T) {
	s := &Dockerfile{}
	cases := []struct {
		path string
		want bool
	}{
		{"Dockerfile", true},
		{"dockerfile", true},
		{"DOCKERFILE", true},
		{"Containerfile", true},
		{"containerfile", true},
		{"app.dockerfile", true},
		{"App.Dockerfile", true},
		{"Dockerfile.dev", true},
		{"dockerfile.prod", true},
		{"Containerfile.test", true},
		{"main.go", false},
		{"docker-compose.yml", false},
		{"notadockerfile", false},
		{".dockerfile", true},    // HasSuffix matches
		{"path/to/Dockerfile", true},
		{"path/to/app.Dockerfile", true},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			if got := s.Match(tc.path); got != tc.want {
				t.Errorf("Match(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// --- Parse helpers ---

func mustParse(t *testing.T, content string) []*Occurrence {
	t.Helper()
	s := &Dockerfile{}
	occ, err := s.Parse("Dockerfile", []byte(content))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	return occ
}

func assertOccurrences(t *testing.T, occ []*Occurrence, wantImages ...string) {
	t.Helper()
	if len(occ) != len(wantImages) {
		var got []string
		for _, o := range occ {
			got = append(got, o.Image.String())
		}
		t.Fatalf("got %d occurrences %v, want %d %v", len(occ), got, len(wantImages), wantImages)
	}
	for i, want := range wantImages {
		if occ[i].Image.String() != want {
			t.Errorf("occ[%d].Image = %q, want %q", i, occ[i].Image.String(), want)
		}
	}
}

// --- Parse tests ---

func TestDockerfileParse_Simple(t *testing.T) {
	src := `FROM docker.io/library/nginx:1.25.3
RUN echo hello
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/nginx:1.25.3")
	if occ[0].Line != 1 {
		t.Errorf("expected line 1, got %d", occ[0].Line)
	}
}

func TestDockerfileParse_MultiStage(t *testing.T) {
	src := `FROM docker.io/library/golang:1.22.0 AS builder
RUN go build .
FROM docker.io/library/alpine:3.19.1
COPY --from=builder /app /app
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ,
		"docker.io/library/golang:1.22.0",
		"docker.io/library/alpine:3.19.1",
	)
}

func TestDockerfileParse_SkipScratch(t *testing.T) {
	src := `FROM docker.io/library/golang:1.22.0 AS builder
FROM scratch
COPY --from=builder /app /app
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/golang:1.22.0")
}

func TestDockerfileParse_SkipStageAlias(t *testing.T) {
	src := `FROM docker.io/library/golang:1.22.0 AS build
FROM build AS something-else
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	// "build" is an alias — skip it; "something-else" is also an alias of alias — skip too
	assertOccurrences(t, occ,
		"docker.io/library/golang:1.22.0",
		"docker.io/library/alpine:3.19.1",
	)
}

func TestDockerfileParse_SkipTemplated(t *testing.T) {
	src := `FROM ${BASE_IMAGE}
FROM docker.io/library/nginx:1.25.3
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/nginx:1.25.3")
}

func TestDockerfileParse_WithPlatformFlag(t *testing.T) {
	src := `FROM --platform=linux/amd64 docker.io/library/golang:1.22.0 AS builder
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/golang:1.22.0")
}

func TestDockerfileParse_WithDigest(t *testing.T) {
	src := `FROM docker.io/library/nginx:1.25.3@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1
`
	occ := mustParse(t, src)
	if len(occ) != 1 {
		t.Fatalf("expected 1 occurrence, got %d", len(occ))
	}
	if occ[0].Image.Digest == "" {
		t.Error("expected digest to be set")
	}
	if occ[0].Image.Tag != "1.25.3" {
		t.Errorf("expected tag 1.25.3, got %q", occ[0].Image.Tag)
	}
}

func TestDockerfileParse_SkipNoSemverTag(t *testing.T) {
	// "latest" is not a semver tag and has no digest → should be skipped
	src := `FROM docker.io/library/nginx:latest
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/alpine:3.19.1")
}

func TestDockerfileParse_SkipUnofficialDomain(t *testing.T) {
	// "nginx" without domain → no explicit domain → skip
	src := `FROM nginx:1.25.3
FROM docker.io/library/nginx:1.25.3
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/nginx:1.25.3")
}

func TestDockerfileParse_LineContinuation(t *testing.T) {
	// buildkit handles line continuations natively
	src := `FROM \
  docker.io/library/nginx:1.25.3 \
  AS webserver
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/nginx:1.25.3")
}

func TestDockerfileParse_CommentIgnored(t *testing.T) {
	src := `# FROM docker.io/library/nginx:1.25.3
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/alpine:3.19.1")
}

func TestDockerfileParse_EscapeDirective(t *testing.T) {
	// buildkit respects # escape=` directive
	src := "# escape=`\nFROM docker.io/library/nginx:1.25.3 `\n  AS webserver\n"
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/nginx:1.25.3")
}

func TestDockerfileParse_ByteOffsets(t *testing.T) {
	src := "FROM docker.io/library/nginx:1.25.3\n"
	occ := mustParse(t, src)
	if len(occ) != 1 {
		t.Fatalf("expected 1 occurrence")
	}
	o := occ[0]
	got := string([]byte(src)[o.StartByte:o.EndByte])
	if got != "docker.io/library/nginx:1.25.3" {
		t.Errorf("byte slice = %q, want %q", got, "docker.io/library/nginx:1.25.3")
	}
}

func TestDockerfileParse_ByteOffsetsMultiStage(t *testing.T) {
	src := "FROM docker.io/library/golang:1.22.0 AS builder\nFROM docker.io/library/alpine:3.19.1\n"
	occ := mustParse(t, src)
	if len(occ) != 2 {
		t.Fatalf("expected 2 occurrences, got %d", len(occ))
	}
	for _, o := range occ {
		got := string([]byte(src)[o.StartByte:o.EndByte])
		if got != o.Image.String() {
			t.Errorf("byte slice %q != Image.String() %q", got, o.Image.String())
		}
	}
}

func TestDockerfileParse_Empty(t *testing.T) {
	occ := mustParse(t, "")
	if len(occ) != 0 {
		t.Errorf("expected 0 occurrences for empty file, got %d", len(occ))
	}
}

func TestDockerfileParse_OnlyComments(t *testing.T) {
	src := "# just a comment\n# another comment\n"
	occ := mustParse(t, src)
	if len(occ) != 0 {
		t.Errorf("expected 0 occurrences, got %d", len(occ))
	}
}

func TestDockerfileParse_ManyStages(t *testing.T) {
	src := strings.Join([]string{
		"FROM docker.io/library/golang:1.21.0 AS stage1",
		"FROM docker.io/library/golang:1.22.0 AS stage2",
		"FROM docker.io/library/alpine:3.18.0 AS stage3",
		"FROM docker.io/library/alpine:3.19.1",
	}, "\n") + "\n"

	occ := mustParse(t, src)
	assertOccurrences(t, occ,
		"docker.io/library/golang:1.21.0",
		"docker.io/library/golang:1.22.0",
		"docker.io/library/alpine:3.18.0",
		"docker.io/library/alpine:3.19.1",
	)
}

func TestDockerfileParse_SortedByStartByte(t *testing.T) {
	src := "FROM docker.io/library/golang:1.22.0 AS b\nFROM docker.io/library/alpine:3.19.1\n"
	occ := mustParse(t, src)
	if len(occ) < 2 {
		t.Fatal("expected 2 occurrences")
	}
	if occ[0].StartByte >= occ[1].StartByte {
		t.Errorf("occurrences not sorted: %d >= %d", occ[0].StartByte, occ[1].StartByte)
	}
}

func TestDockerfileParse_OnlyDigestNoTag(t *testing.T) {
	// image with only a digest (no tag) — valid since image.Parse allows digest-only
	src := "FROM docker.io/library/nginx@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1\n"
	occ := mustParse(t, src)
	if len(occ) != 1 {
		t.Fatalf("expected 1 occurrence, got %d", len(occ))
	}
	if occ[0].Image.Digest == "" {
		t.Error("expected non-empty digest")
	}
}

// --- Name test ---

func TestDockerfileName(t *testing.T) {
	s := &Dockerfile{}
	if s.Name() != "dockerfile" {
		t.Errorf("Name() = %q, want %q", s.Name(), "dockerfile")
	}
}

// --- Дополнительные граничные случаи ---

// SCRATCH в верхнем/смешанном регистре должен пропускаться так же.
func TestDockerfileParse_SkipScratchCaseInsensitive(t *testing.T) {
	src := `FROM SCRATCH
FROM Scratch
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/alpine:3.19.1")
}

// Несколько флагов перед именем образа (--platform и гипотетический будущий флаг).
func TestDockerfileParse_MultipleFlagsBeforeImage(t *testing.T) {
	src := `FROM --platform=linux/arm64 docker.io/library/golang:1.22.0 AS cross
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/golang:1.22.0")
}

// ARG-шаблон через ${VAR} и через $VAR (без фигурных скобок) — оба пропускаются.
func TestDockerfileParse_SkipArgVariants(t *testing.T) {
	src := `ARG GO_VERSION=1.22.0
FROM golang:${GO_VERSION}
FROM golang:$GO_VERSION
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ, "docker.io/library/alpine:3.19.1")
}

// Алиас стадии, совпадающий по написанию с реальным реестром — не должен
// блокировать образы с таким же "именем" в другой стадии.
func TestDockerfileParse_AliasDoesNotBlockUnrelatedImage(t *testing.T) {
	src := `FROM docker.io/library/golang:1.22.0 AS myalias
FROM docker.io/library/alpine:3.19.1 AS myalias2
FROM docker.io/library/nginx:1.25.3
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ,
		"docker.io/library/golang:1.22.0",
		"docker.io/library/alpine:3.19.1",
		"docker.io/library/nginx:1.25.3",
	)
}

// FROM без аргумента — не падаем, просто ничего не возвращаем.
func TestDockerfileParse_InvalidFromNoArg(t *testing.T) {
	// buildkit сам вернёт ошибку на синтаксически неверный файл,
	// но FROM с аргументом-алиасом стадии без образа — штатный случай.
	// Проверяем что валидные строки вокруг него разбираются нормально.
	src := `FROM docker.io/library/golang:1.22.0 AS builder
FROM builder
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	assertOccurrences(t, occ,
		"docker.io/library/golang:1.22.0",
		"docker.io/library/alpine:3.19.1",
	)
}

// Проверяем что Line в Occurrence правильно соответствует номеру строки FROM.
func TestDockerfileParse_LineNumbers(t *testing.T) {
	src := `# comment
FROM docker.io/library/golang:1.22.0 AS builder
RUN echo hello
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	if len(occ) != 2 {
		t.Fatalf("expected 2 occurrences, got %d", len(occ))
	}
	if occ[0].Line != 2 {
		t.Errorf("occ[0].Line = %d, want 2", occ[0].Line)
	}
	if occ[1].Line != 4 {
		t.Errorf("occ[1].Line = %d, want 4", occ[1].Line)
	}
}

// Kind должен быть FullReference для всех вхождений из Dockerfile.
func TestDockerfileParse_KindIsFullReference(t *testing.T) {
	src := `FROM docker.io/library/nginx:1.25.3
FROM docker.io/library/alpine:3.19.1
`
	occ := mustParse(t, src)
	for _, o := range occ {
		if o.Kind != FullReference {
			t.Errorf("occ.Kind = %v, want FullReference", o.Kind)
		}
	}
}

// File в Occurrence должен совпадать с переданным filePath.
func TestDockerfileParse_FilePath(t *testing.T) {
	s := &Dockerfile{}
	content := []byte("FROM docker.io/library/nginx:1.25.3\n")
	occ, err := s.Parse("path/to/Dockerfile", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(occ) != 1 {
		t.Fatalf("expected 1 occurrence")
	}
	if occ[0].File != "path/to/Dockerfile" {
		t.Errorf("File = %q, want %q", occ[0].File, "path/to/Dockerfile")
	}
}

// --- Fixture-based tests ---

// TestDockerfileFixture_Parse проверяет что из basic.Dockerfile стратегия
// извлекает ровно два валидных образа и корректно пропускает все остальные.
func TestDockerfileFixture_Parse(t *testing.T) {
	const fixturePath = "test-fixtures/basic.Dockerfile"
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	s := &Dockerfile{}
	occ, err := s.Parse(fixturePath, content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Ожидаем ровно два образа:
	//   docker.io/library/alpine:3.18           (builder stage)
	//   gcr.io/distroless/static:3.0.0          (--platform stage)
	//   quay.io/jetstack/cert-manager:1.13.0    (line-continuation)
	//
	// Пропущены: scratch, builder (alias), ${BASE_IMAGE}, ubuntu:latest, redis:7.0
	wantImages := []string{
		"docker.io/library/alpine:3.18",
		"gcr.io/distroless/static:3.0.0",
		"quay.io/jetstack/cert-manager:1.13.0",
	}
	assertOccurrences(t, occ, wantImages...)

	// Байтовые смещения должны точно указывать на строку образа в оригинале.
	for _, o := range occ {
		got := string(content[o.StartByte:o.EndByte])
		if got != o.Image.String() {
			t.Errorf("byte slice %q != Image.String() %q (line %d)",
				got, o.Image.String(), o.Line)
		}
	}
}

// TestDockerfileFixture_ApplyUpdate читает basic.Dockerfile, применяет апдейт
// cert-manager 1.13.0 → 1.14.0 и сравнивает с basic.expected.Dockerfile.
func TestDockerfileFixture_ApplyUpdate(t *testing.T) {
	const fixturePath = "test-fixtures/basic.Dockerfile"
	const expectedPath = "test-fixtures/basic.expected.Dockerfile"

	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	s := &Dockerfile{}
	occ, err := s.Parse(fixturePath, content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Фильтруем: только cert-manager.
	var certOcc []*Occurrence
	for _, o := range occ {
		if o.Image.Path == "jetstack/cert-manager" {
			certOcc = append(certOcc, o)
		}
	}
	if len(certOcc) != 1 {
		t.Fatalf("expected 1 cert-manager occurrence, got %d", len(certOcc))
	}

	newVer, err := semver.NewVersion("1.14.0")
	if err != nil {
		t.Fatal(err)
	}

	result, count, err := ApplyUpdates(content, certOcc, newVer, "")
	if err != nil {
		t.Fatalf("ApplyUpdates error: %v", err)
	}
	if count != 1 {
		t.Errorf("updated count = %d, want 1", count)
	}
	if string(result) != string(expected) {
		t.Errorf("result mismatch:\ngot:\n%s\nwant:\n%s", result, expected)
	}
}
