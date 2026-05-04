package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DockerfileSuite struct {
	strategySuite
}

func TestDockerfileSuite(t *testing.T) {
	suite.Run(t, &DockerfileSuite{
		strategySuite: strategySuite{
			strategy:       &Dockerfile{},
			fixturesSubdir: "dockerfile",
			basicFile:      "basic.Dockerfile",
			expectedFile:   "expected.Dockerfile",
		},
	})
}

func (s *DockerfileSuite) TestMatch() {
	s.True(s.strategy.Match("Dockerfile"))
	s.True(s.strategy.Match("path/to/Dockerfile"))
	s.True(s.strategy.Match("Containerfile"))
	s.True(s.strategy.Match("backend.Dockerfile"))
	s.True(s.strategy.Match("Dockerfile.dev"))
	s.False(s.strategy.Match("docker-compose.yml"))
}

func (s *DockerfileSuite) TestParse() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// Полные ссылки с явным доменом и semver-версией.
	s.Contains(keys, "docker.io/library/alpine:3.18")
	s.Contains(keys, "gcr.io/distroless/static:3.0.0")
	// Line continuation.
	s.Contains(keys, "quay.io/jetstack/cert-manager:1.13.0")
	// Стадии и шаблоны не попадают.
	s.NotContains(keys, "library/builder")
	// scratch/$BASE_IMAGE/ubuntu:latest/redis:7.0 не попадают.
	for key := range keys {
		s.NotContains(key, "scratch")
		s.NotContains(key, "ubuntu")
		s.NotContains(key, "redis:7.0")
	}
}

func (s *DockerfileSuite) TestApplyUpdatesOnlySameMajor() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
	// cert-manager:1.13.0 имеет major=1, поэтому после update до 3.99.0
	// он останется как есть. expected.Dockerfile это отражает.
}

func (s *DockerfileSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Equal(FullReference, occurrence.Kind)
		s.Greater(occurrence.Line, 0)
		s.GreaterOrEqual(occurrence.StartByte, 0)
		s.Greater(occurrence.EndByte, occurrence.StartByte)
		s.LessOrEqual(occurrence.EndByte, len(content))
	}
}
