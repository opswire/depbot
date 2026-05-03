package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type YAMLSuite struct {
	strategySuite
}

func TestYAMLSuite(t *testing.T) {
	suite.Run(t, &YAMLSuite{
		strategySuite: strategySuite{
			strategy:       &YAML{},
			fixturesSubdir: "yaml",
			basicFile:      "basic.yaml",
			expectedFile:   "expected.yaml",
		},
	})
}

func (s *YAMLSuite) TestMatch() {
	s.True(s.strategy.Match("deployment.yaml"))
	s.True(s.strategy.Match("compose.yml"))
	s.True(s.strategy.Match("path/to/values.YAML"))
	s.False(s.strategy.Match("Dockerfile"))
	s.False(s.strategy.Match("config.json"))
}

func (s *YAMLSuite) TestParseScalarImages() {
	_, occurrences := s.parseBasic()
	// busybox с кавычками
	s.requireOccurrence(occurrences, "docker.io/library/busybox:1.35", FullReference)
	// app с registry:port
	s.requireOccurrence(occurrences, "registry.example.com:5000/myorg/app:2.4.1", FullReference)
	// uses: docker:// prefix
	s.requireOccurrence(occurrences, "docker.io/library/nginx:1.21-alpine", FullReference)
}

func (s *YAMLSuite) TestParseHelmMappingProducesTagAndDigest() {
	_, occurrences := s.parseBasic()
	// bitnami/redis должен дать ровно 2 Occurrence: TagOnly + DigestOnly
	tagOccurrence := s.requireOccurrence(occurrences, "docker.io/bitnami/redis:7.2.0", TagOnly)
	digestOccurrence := s.requireOccurrence(occurrences, "docker.io/bitnami/redis:7.2.0", DigestOnly)
	s.NotEqual(tagOccurrence.StartByte, digestOccurrence.StartByte)
}

func (s *YAMLSuite) TestParseHelmMappingWithoutDigest() {
	_, occurrences := s.parseBasic()
	// postgres имеет только tag, не digest
	s.requireOccurrence(occurrences, "docker.io/library/postgres:16.0", TagOnly)
	// Не должен быть DigestOnly для postgres
	for _, occurrence := range occurrences {
		if occurrence.Image.Name == "library/postgres" {
			s.NotEqual(DigestOnly, occurrence.Kind)
		}
	}
}

func (s *YAMLSuite) TestParseSkipsTemplated() {
	_, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.NotContains(occurrence.Image.SourceName, "{{")
		s.NotContains(occurrence.Image.SourceName, "}}")
	}
}

func (s *YAMLSuite) TestParseSkipsNonSemver() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	for key := range keys {
		s.NotContains(key, "ubuntu")
	}
}

func (s *YAMLSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Greater(occurrence.Line, 0)
		s.GreaterOrEqual(occurrence.StartByte, 0)
		s.Greater(occurrence.EndByte, occurrence.StartByte)
		s.LessOrEqual(occurrence.EndByte, len(content))
	}
}

func (s *YAMLSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}

func (s *YAMLSuite) TestEmptyDocument() {
	occurrences, err := s.strategy.Parse("empty.yaml", []byte(""))
	s.NoError(err)
	s.Empty(occurrences)
}

func (s *YAMLSuite) TestInvalidYAMLReturnsError() {
	_, err := s.strategy.Parse("bad.yaml", []byte("key: : :\n  value: [[[\n"))
	s.Error(err)
}
