package depbot

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
	s.False(s.strategy.Match("Dockerfile"))
	s.False(s.strategy.Match("config.yaml"))
}

func (s *JSONSuite) TestParseFindsImagesIgnoringComments() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// Должны быть найдены: devcontainer image и web. ubuntu:latest скипается.
	s.Contains(keys, "mcr.microsoft.com/devcontainers/go:1.21")
	s.Contains(keys, "docker.io/myorg/web:1.0.0")
	s.NotContains(keys, "docker.io/library/ubuntu")
}

func (s *JSONSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Equal(FullReference, occurrence.Kind)
		s.Greater(occurrence.Line, 0)
		// StartByte указывает на сам литерал имени, не на кавычку
		s.NotEqual(byte('"'), content[occurrence.StartByte])
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
