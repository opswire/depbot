package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HCLSuite struct {
	strategySuite
}

func TestHCLSuite(t *testing.T) {
	suite.Run(t, &HCLSuite{
		strategySuite: strategySuite{
			strategy:       &HCL{},
			fixturesSubdir: "hcl",
			basicFile:      "basic.tf",
			expectedFile:   "expected.tf",
		},
	})
}

func (s *HCLSuite) TestMatch() {
	s.True(s.strategy.Match("main.tf"))
	s.True(s.strategy.Match("config.hcl"))
	s.True(s.strategy.Match("job.nomad"))
	s.False(s.strategy.Match("config.tf.json"), ".tf.json должен достаться JSON-стратегии")
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *HCLSuite) TestParseFindsImagesSkippingComments() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/myorg/app:1.5.0")
	s.Contains(keys, "docker.io/envoyproxy/envoy:v1.28.0")
	s.NotContains(keys, "docker.io/fake/skipped")
	s.NotContains(keys, "docker.io/another/skipped")
	s.NotContains(keys, "docker.io/library/ubuntu")
}

func (s *HCLSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
