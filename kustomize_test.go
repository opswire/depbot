package depbot

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
	// nginx с newName: my-registry/nginx → нормализуется в docker.io/my-registry/nginx
	occurrence := s.requireOccurrence(occurrences, "docker.io/my-registry/nginx:1.22.0", TagOnly)
	s.Equal("my-registry/nginx", occurrence.Image.SourceName)
}

func (s *KustomizeSuite) TestParseImagesWithDigest() {
	_, occurrences := s.parseBasic()
	// redis имеет и newTag и digest — должно быть 2 Occurrence
	s.requireOccurrence(occurrences, "docker.io/library/redis:7.2.0", TagOnly)
	s.requireOccurrence(occurrences, "docker.io/library/redis:7.2.0", DigestOnly)
}

func (s *KustomizeSuite) TestParseSkipsNonSemverNewTag() {
	_, occurrences := s.parseBasic()
	// "newTag: latest" — non-semver, не должен породить Occurrence
	for _, occurrence := range occurrences {
		s.NotEqual("myorg/app", occurrence.Image.SourceName)
	}
}

func (s *KustomizeSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
