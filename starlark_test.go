package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StarlarkSuite struct {
	strategySuite
}

func TestStarlarkSuite(t *testing.T) {
	suite.Run(t, &StarlarkSuite{
		strategySuite: strategySuite{
			strategy:       &Starlark{},
			fixturesSubdir: "starlark",
			basicFile:      "Tiltfile",
			expectedFile:   "Tiltfile.expected",
		},
	})
}

func (s *StarlarkSuite) TestMatch() {
	s.True(s.strategy.Match("Tiltfile"))
	s.True(s.strategy.Match("path/to/Tiltfile"))
	s.True(s.strategy.Match("BUILD"))
	s.True(s.strategy.Match("BUILD.bazel"))
	s.True(s.strategy.Match("WORKSPACE"))
	s.True(s.strategy.Match("rules.bzl"))
	s.False(s.strategy.Match("tiltfile.txt"))
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *StarlarkSuite) TestParseFindsImagesInBothQuoteStyles() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// Одинарные кавычки
	s.Contains(keys, "docker.io/myorg/frontend:0.1.0")
	// Двойные кавычки
	s.Contains(keys, "docker.io/myorg/backend:0.2.0")
	// Custom registry
	s.Contains(keys, "gcr.io/myproj/api:3.0.0")
}

func (s *StarlarkSuite) TestParseSkipsSplitImageTag() {
	_, occurrences := s.parseBasic()
	// container_pull(image="docker.io/library/alpine", tag="3.18") —
	// у "image" нет тега, поэтому Starlark эту строку не подбирает
	for _, occurrence := range occurrences {
		s.NotEqual("docker.io/library/alpine", occurrence.Image.SourceName)
	}
}

func (s *StarlarkSuite) TestParseSkipsCommentedLines() {
	_, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.NotContains(occurrence.Image.SourceName, "commented")
	}
}

func (s *StarlarkSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
