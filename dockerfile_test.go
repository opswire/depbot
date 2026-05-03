package depbot

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
	s.False(s.strategy.Match("README.md"))
}

func (s *DockerfileSuite) TestParseFindsAllSemverImages() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Equal(2, len(occurrences), "ожидаем ровно 2 вхождения")
	s.Contains(keys, "docker.io/library/alpine:3.18")
	s.Contains(keys, "docker.io/library/nginx:1.21-alpine")
}

func (s *DockerfileSuite) TestParseSkipsScratchAliasesAndTemplates() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// scratch — magic base, скипаем
	s.NotContains(keys, "docker.io/library/scratch")
	// "FROM builder AS final" — алиас стадии, не образ
	s.NotContains(keys, "docker.io/library/builder")
	// "FROM ${BASE_IMAGE}" — шаблонный, скипаем
	for _, occurrence := range occurrences {
		s.NotContains(occurrence.Image.SourceName, "$")
	}
}

func (s *DockerfileSuite) TestParseSkipsNonSemver() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// "ubuntu:latest" — non-semver tag, не должен породить Occurrence
	for key := range keys {
		s.NotContains(key, "ubuntu")
	}
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

func (s *DockerfileSuite) TestParseSorted() {
	_, occurrences := s.parseBasic()
	for index := 1; index < len(occurrences); index++ {
		s.Less(occurrences[index-1].StartByte, occurrences[index].StartByte,
			"вхождения должны быть отсортированы по StartByte")
	}
}

func (s *DockerfileSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}
