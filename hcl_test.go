package parser

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
	s.False(s.strategy.Match("config.tf.json"))
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *HCLSuite) TestParseFindsImagesSkippingCommentsAndShortForm() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/myorg/app:3.5.0")
	s.Contains(keys, "gcr.io/envoyproxy/envoy:3.28.0")
	for key := range keys {
		s.NotContains(key, "skipped")
		s.NotContains(key, "ubuntu")
		s.NotContains(key, "library/nginx")
	}
}

func (s *HCLSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
