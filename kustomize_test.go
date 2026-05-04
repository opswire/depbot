package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type KustomizeSuite struct {
	strategySuite
}

func TestKustomizeSuite(t *testing.T) {
	suite.Run(t, &KustomizeSuite{
		strategySuite: strategySuite{
			strategy:       &Kustomize{},
			fixturesSubdir: "kustomize",
			basicFile:      "basic.yaml",
			expectedFile:   "expected.yaml",
		},
	})
}

func (s *KustomizeSuite) TestMatch() {
	s.True(s.strategy.Match("kustomization.yaml"))
	s.True(s.strategy.Match("path/to/kustomization.yml"))
	s.True(s.strategy.Match("Kustomization.yaml"))
	s.False(s.strategy.Match("kustomize.yaml"))
	s.False(s.strategy.Match("deployment.yaml"))
}

func (s *KustomizeSuite) TestParseImagesWithNewName() {
	_, occurrences := s.parseBasic()
	occurrence := s.requireOccurrence(occurrences, "gcr.io/distroless/nginx:3.22.0", TagOnly)
	s.Equal("distroless/nginx", occurrence.Image.Name)
	s.Equal("gcr.io", occurrence.Image.Domain)
}

func (s *KustomizeSuite) TestParseImagesWithDigest() {
	_, occurrences := s.parseBasic()
	s.requireOccurrence(occurrences, "docker.io/library/redis:3.2.0", TagOnly)
	s.requireOccurrence(occurrences, "docker.io/library/redis:3.2.0", DigestOnly)
}

func (s *KustomizeSuite) TestParseSkipsNonSemverAndBareNames() {
	_, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		// "myorg/app:latest" пропустить (non-semver)
		s.NotEqual("myorg/app", occurrence.Image.Name)
		// "bare-without-domain" — нет domain, пропустить
		s.NotEqual("bare-without-domain", occurrence.Image.Name)
	}
}

func (s *KustomizeSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
