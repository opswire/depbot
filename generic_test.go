package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GenericSuite struct {
	strategySuite
}

func TestGenericSuite(t *testing.T) {
	suite.Run(t, &GenericSuite{
		strategySuite: strategySuite{
			strategy:       &Generic{},
			fixturesSubdir: "generic",
			basicFile:      "basic.md",
			expectedFile:   "expected.md",
		},
	})
}

func (s *GenericSuite) TestMatchAcceptsWhitelistedExtensions() {
	s.True(s.strategy.Match("README.md"))
	s.True(s.strategy.Match("notes.txt"))
	s.True(s.strategy.Match("doc.rst"))
	s.True(s.strategy.Match("manual.adoc"))
	s.True(s.strategy.Match("server.log"))
	s.True(s.strategy.Match("config.conf"))
	s.True(s.strategy.Match("settings.cfg"))
	s.True(s.strategy.Match("setup.ini"))
	s.True(s.strategy.Match("app.properties"))
}

func (s *GenericSuite) TestMatchRejectsEverythingElse() {
	s.False(s.strategy.Match("script"))
	s.False(s.strategy.Match("README"))
	s.False(s.strategy.Match("app.js"))
	s.False(s.strategy.Match("style.css"))
	s.False(s.strategy.Match("main.go"))
	s.False(s.strategy.Match("server.py"))
	s.False(s.strategy.Match("logo.png"))
	s.False(s.strategy.Match("archive.zip"))
	s.False(s.strategy.Match("build.sh"))
}

func (s *GenericSuite) TestParse() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/library/alpine:3.19")
	s.Contains(keys, "gcr.io/myorg/app:3.5.0")
	s.Contains(keys, "quay.io/jetstack/cert-manager:3.13.0")
}

func (s *GenericSuite) TestParseSkipsCommentedShortFormHostPort() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	for key := range keys {
		s.NotContains(key, "pretend/image")
		s.NotContains(key, "ubuntu")
		s.NotContains(key, "library/nginx")
		s.NotContains(key, "registry.example.com:5000")
	}
}

func (s *GenericSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}

func (s *GenericSuite) TestEmptyContent() {
	occurrences, err := s.strategy.Parse("empty.txt", []byte(""))
	s.NoError(err)
	s.Empty(occurrences)
}
