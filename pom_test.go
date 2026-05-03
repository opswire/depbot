package depbot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PomSuite struct {
	strategySuite
}

func TestPomSuite(t *testing.T) {
	suite.Run(t, &PomSuite{
		strategySuite: strategySuite{
			strategy:       &PomXML{},
			fixturesSubdir: "pom",
			basicFile:      "basic.xml",
			expectedFile:   "expected.xml",
		},
	})
}

func (s *PomSuite) TestMatch() {
	s.True(s.strategy.Match("pom.xml"))
	s.True(s.strategy.Match("path/to/pom.xml"))
	s.True(s.strategy.Match("POM.XML"))
	s.False(s.strategy.Match("build.xml"))
	s.False(s.strategy.Match("Dockerfile"))
}

func (s *PomSuite) TestParseFindsAllImages() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Equal(3, len(occurrences), "ожидаем jib from + jib to + fabric8 = 3")
	s.Contains(keys, "docker.io/library/openjdk:11-jre-slim")
	s.Contains(keys, "gcr.io/myproj/app:1.0.0")
	s.Contains(keys, "example.com/another:5.0.0")
}

func (s *PomSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Equal(FullReference, occurrence.Kind)
		s.Greater(occurrence.Line, 0)
		s.GreaterOrEqual(occurrence.StartByte, 0)
		s.Greater(occurrence.EndByte, occurrence.StartByte)
		s.LessOrEqual(occurrence.EndByte, len(content))
		// StartByte должен указывать в trimmed-значение (не на whitespace)
		s.NotEqual(byte(' '), content[occurrence.StartByte])
		s.NotEqual(byte('\t'), content[occurrence.StartByte])
		s.NotEqual(byte('\n'), content[occurrence.StartByte])
	}
}

func (s *PomSuite) TestApplyUpdatesByteForByte() {
	content, occurrences := s.parseBasic()
	s.applyAndCompare(content, occurrences)
}

func (s *PomSuite) TestInvalidXMLReturnsError() {
	_, err := s.strategy.Parse("bad.xml", []byte("<unclosed>"))
	s.Error(err)
}
