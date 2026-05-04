package parser

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
	s.True(s.strategy.Match("path/to/.env"))
	s.True(s.strategy.Match("dev.env"))
	s.False(s.strategy.Match("environment.txt"))
}

func (s *EnvSuite) TestParse() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/myorg/app:3.1.0")
	s.Contains(keys, "docker.io/library/alpine:3.19")
	s.Contains(keys, "gcr.io/distroless/nginx:3.25.0")

	// PORT, NODE_ENV, TIMEOUT — не образы
	for key := range keys {
		s.NotContains(key, "8080")
		// Short-form не должен попасть
		s.NotContains(key, "library/traefik:3.0")
		// host:port не должен попасть
		s.NotContains(key, "db.example.com")
		s.NotContains(key, "registry.example.com:5000")
	}
}

func (s *EnvSuite) TestParseSkipsTemplatedAndNonSemver() {
	_, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.NotEqual("library/traefik", occurrence.Image.Name)
		s.NotContains(occurrence.Image.Domain, "$")
	}
}

func (s *EnvSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
