package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JSONSuite struct {
	strategySuite
}

func TestJSONSuite(t *testing.T) {
	suite.Run(t, &JSONSuite{
		strategySuite: strategySuite{
			strategy:       &JSON{},
			fixturesSubdir: "json",
			basicFile:      "basic.jsonc",
			expectedFile:   "expected.jsonc",
		},
	})
}

func (s *JSONSuite) TestMatch() {
	s.True(s.strategy.Match("config.json"))
	s.True(s.strategy.Match("devcontainer.jsonc"))
	s.True(s.strategy.Match("path/to/config.JSON"))
	s.False(s.strategy.Match("config.yaml"))
}

func (s *JSONSuite) TestParseFindsImagesIgnoringComments() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "mcr.microsoft.com/devcontainers/go:3.21")
	s.Contains(keys, "docker.io/myorg/web:3.0.0")
	// Не попадают:
	for key := range keys {
		s.NotContains(key, "ubuntu")
		s.NotContains(key, "library/nginx")
	}
}

func (s *JSONSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}

func (s *JSONSuite) TestEmptyJSON() {
	occurrences, err := s.strategy.Parse("empty.json", []byte("{}"))
	s.NoError(err)
	s.Empty(occurrences)
}
