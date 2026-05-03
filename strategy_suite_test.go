package depbot

import (
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
)

// strategySuite — общая база для тестов всех стратегий.
//
// Каждый конкретный тест-сьют наследует её и задаёт три поля:
//   - strategy:        тестируемая стратегия;
//   - fixturesSubdir:  имя подкаталога в test-fixtures (например "yaml");
//   - basicFile:       имя входного файла (например "basic.yaml");
//   - expectedFile:    имя ожидаемого файла после ApplyUpdates до 9.9.9 + sha256.
type strategySuite struct {
	suite.Suite
	strategy       Strategy
	fixturesSubdir string
	basicFile      string
	expectedFile   string
}

// newVersion — версия, которую тесты подставляют через ApplyUpdates.
const newVersion = "9.9.9"

// newSHA256 — sha256-дайджест, который тесты подставляют через ApplyUpdates.
const newSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// fixturePath возвращает абсолютный путь к фикстуре.
func (s *strategySuite) fixturePath(name string) string {
	return filepath.Join("test-fixtures", s.fixturesSubdir, name)
}

// readFixture читает фикстуру и фейлит тест при ошибке.
func (s *strategySuite) readFixture(name string) []byte {
	content, err := os.ReadFile(s.fixturePath(name))
	s.Require().NoError(err, "не удалось прочитать фикстуру %s", name)
	return content
}

// parseBasic запускает стратегию на basicFile и возвращает Occurrence-ы.
func (s *strategySuite) parseBasic() ([]byte, []*Occurrence) {
	content := s.readFixture(s.basicFile)
	occurrences, err := s.strategy.Parse(s.fixturePath(s.basicFile), content)
	s.Require().NoError(err, "Parse() вернул ошибку")
	return content, occurrences
}

// applyAndCompare применяет ApplyUpdates с newVersion+newSHA256 и сравнивает
// результат с expectedFile.
func (s *strategySuite) applyAndCompare(content []byte, occurrences []*Occurrence) {
	version, err := semver.NewVersion(newVersion)
	s.Require().NoError(err)
	updated, err := ApplyUpdates(content, occurrences, version, newSHA256)
	s.Require().NoError(err, "ApplyUpdates() вернул ошибку")
	expected := s.readFixture(s.expectedFile)
	s.Equal(string(expected), string(updated))
}

// requireOccurrence проверяет, что среди occurrences есть один со
// значениями key/kind, и возвращает его. Удобно для проверки конкретных
// записей без жёсткой привязки к их порядку.
func (s *strategySuite) requireOccurrence(occurrences []*Occurrence, key string, kind FieldKind) *Occurrence {
	for _, occurrence := range occurrences {
		if occurrence.Image.Key() == key && occurrence.Kind == kind {
			return occurrence
		}
	}
	s.FailNowf("вхождение не найдено", "key=%q kind=%v", key, kind)
	return nil
}

// imageKeys возвращает множество Key()-ов всех Occurrence — удобно для
// проверки, что какие-то ссылки НЕ были подобраны.
func imageKeys(occurrences []*Occurrence) map[string]int {
	result := map[string]int{}
	for _, occurrence := range occurrences {
		result[occurrence.Image.Key()]++
	}
	return result
}
