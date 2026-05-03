package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TOMLSuite struct {
	strategySuite
}

func TestTOMLSuite(t *testing.T) {
	suite.Run(t, &TOMLSuite{
		strategySuite: strategySuite{
			strategy:       &TOML{},
			fixturesSubdir: "toml",
			basicFile:      "basic.toml",
			expectedFile:   "expected.toml",
		},
	})
}

func (s *TOMLSuite) TestMatch() {
	s.True(s.strategy.Match("project.toml"))
	s.True(s.strategy.Match("path/to/config.TOML"))
	s.False(s.strategy.Match("config.tf"))
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *TOMLSuite) TestParseFindsImagesSkippingComments() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/paketobuildpacks/builder-jammy-base:0.4.0")
	s.Contains(keys, "ghcr.io/example/tool:2.1.0")
	s.NotContains(keys, "docker.io/fake/skipped")
	s.NotContains(keys, "docker.io/library/ubuntu")
}

func (s *TOMLSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
