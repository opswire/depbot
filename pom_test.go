package parser

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
}

func (s *PomSuite) TestParseFindsAllImages() {
	_, occurrences := s.parseBasic()
	keys := imageKeys(occurrences)
	s.Equal(3, len(occurrences))
	s.Contains(keys, "docker.io/library/openjdk:3.11.1")
	s.Contains(keys, "gcr.io/myproj/app:3.0.0")
	s.Contains(keys, "example.com/another:3.5.0")
}

func (s *PomSuite) TestOccurrenceFields() {
	content, occurrences := s.parseBasic()
	for _, occurrence := range occurrences {
		s.Equal(FullReference, occurrence.Kind)
		s.Greater(occurrence.Line, 0)
		s.NotEqual(byte(' '), content[occurrence.StartByte])
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
