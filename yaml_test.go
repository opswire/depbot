package parser

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
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/library/busybox:3.5.0")
	s.Contains(keys, "registry.example.com:5000/myorg/app:3.4.1")
	s.Contains(keys, "gcr.io/distroless/nginx:3.1.0")
}

func (s *YAMLSuite) TestParseHelmMappingProducesTagAndDigest() {
	_, occurrences := s.parseBasic()
	tagOccurrence := s.requireOccurrence(occurrences, "docker.io/bitnami/redis:3.2.0", TagOnly)
	digestOccurrence := s.requireOccurrence(occurrences, "docker.io/bitnami/redis:3.2.0", DigestOnly)
	s.NotEqual(tagOccurrence.StartByte, digestOccurrence.StartByte)
}

func (s *YAMLSuite) TestParseHelmMappingWithoutDigest() {
	_, occurrences := s.parseBasic()
	s.requireOccurrence(occurrences, "gcr.io/distroless/postgres:3.16.0", TagOnly)
}

func (s *YAMLSuite) TestParseSkipsNonSemverAndShortForm() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	for key := range keys {
		s.NotContains(key, "ubuntu")
		s.NotContains(key, "library/nginx")
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
