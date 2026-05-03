package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EnvSuite struct {
	strategySuite
}

func TestEnvSuite(t *testing.T) {
	suite.Run(t, &EnvSuite{
		strategySuite: strategySuite{
			strategy:       &EnvFile{},
			fixturesSubdir: "env",
			basicFile:      "basic.env",
			expectedFile:   "expected.env",
		},
	})
}

func (s *EnvSuite) TestMatch() {
	s.True(s.strategy.Match(".env"))
	s.True(s.strategy.Match(".env.local"))
	s.True(s.strategy.Match(".env.production"))
	s.True(s.strategy.Match("path/to/.env"))
	s.True(s.strategy.Match("dev.env"))
	s.False(s.strategy.Match("environment.txt"))
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *EnvSuite) TestParseHonorsImagedHeuristic() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// IMAGE/DOCKER в имени или ":" в значении — заберутся
	s.Contains(keys, "docker.io/myorg/app:3.1.0")
	s.Contains(keys, "docker.io/library/alpine:3.19")
	s.Contains(keys, "docker.io/library/nginx:1.25.0")
	// PORT, NODE_ENV, TIMEOUT — не заберутся (нет ни IMAGE/DOCKER в ключе,
	// ни ":" с правильной формой в значении)
	for key := range keys {
		s.NotContains(key, "8080")
	}
}

func (s *EnvSuite) TestParseSkipsTemplatedAndNonSemver() {
	_, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		// PROXY_IMAGE=traefik:latest — non-semver
		s.NotEqual("library/traefik", occurrence.Image.Name)
		// DEV_IMAGE=${BASE}:${TAG} — шаблонный
		s.NotContains(occurrence.Image.SourceName, "${")
	}
}

func (s *EnvSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
