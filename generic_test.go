package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GenericSuite struct {
	strategySuite
}

func TestGenericSuite(t *testing.T) {
	suite.Run(t, &GenericSuite{
		strategySuite: strategySuite{
			strategy:       &Generic{},
			fixturesSubdir: "generic",
			basicFile:      "basic.sh",
			expectedFile:   "expected.sh",
		},
	})
}

func (s *GenericSuite) TestMatchAlwaysTrue() {
	s.True(s.strategy.Match("any.txt"))
	s.True(s.strategy.Match("script.sh"))
	s.True(s.strategy.Match("README.md"))
	s.True(s.strategy.Match("path/to/file"))
}

func (s *GenericSuite) TestParseFindsImagesInArbitraryText() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Contains(keys, "docker.io/library/alpine:3.19")
	s.Contains(keys, "docker.io/myorg/app:1.5.0")
	// Образ упомянутый в echo-сообщении тоже попадает
	s.Contains(keys, "docker.io/library/nginx:1.21-alpine")
}

func (s *GenericSuite) TestParseSkipsCommentedAndNonSemver() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	// Закомментированная строка пропущена
	s.NotContains(keys, "docker.io/skipped/image")
	// non-semver тег "ubuntu:latest" пропущен
	s.NotContains(keys, "docker.io/library/ubuntu")
}

func (s *GenericSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Equal(FullReference, occurrence.Kind)
		s.Greater(occurrence.Line, 0)
		s.GreaterOrEqual(occurrence.StartByte, 0)
		s.Greater(occurrence.EndByte, occurrence.StartByte)
		s.LessOrEqual(occurrence.EndByte, len(content))
	}
}

func (s *GenericSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}

func (s *GenericSuite) TestEmptyContent() {
	occurrences, err := s.strategy.Parse("empty.txt", []byte(""))
	s.NoError(err)
	s.Empty(occurrences)
}

func (s *GenericSuite) TestPlainTextNoImages() {
	occurrences, err := s.strategy.Parse("plain.txt", []byte("Just some\nrandom\ntext.\n"))
	s.NoError(err)
	s.Empty(occurrences)
}
