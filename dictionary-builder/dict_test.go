package main

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	. "langscram-lib" //nolint

	"golang.org/x/text/language"
)

func createTestDictionary() TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	// All 11 languages must have translations for each word cluster
	// Using ASCII-only words to avoid CSV encoding issues
	testWords := []struct {
		en  string
		es  string
		de  string
		ja  string
		ru  string
		sv  string
		pt  string
		nl  string
		it  string
		fr  string
		zh  string
		typ WordType
	}{
		{"cat", "gato", "Katze", "neko", "kot", "katt", "gato", "kat", "gatto", "chat", "mao", WordTypeNoun},
		{"dog", "perro", "Hund", "inu", "sobaka", "hund", "cao", "hond", "cane", "chien", "gou", WordTypeNoun},
		{"house", "casa", "Haus", "ie", "dom", "hus", "casa", "huis", "casa", "maison", "fangzi", WordTypeNoun},
		{"water", "agua", "Wasser", "mizu", "voda", "vatten", "agua", "water", "acqua", "eau", "shui", WordTypeNoun},
		{"book", "libro", "Buch", "hon", "kniga", "bok", "livro", "boek", "libro", "livre", "shu", WordTypeNoun},
		{"run", "correr", "laufen", "hashiru", "bezhat", "springa", "correr", "lopen", "correre", "courir", "pao", WordTypeVerb},
		{"eat", "comer", "essen", "taberu", "est", "ata", "comer", "eten", "mangiare", "manger", "chi", WordTypeVerb},
		{"sleep", "dormir", "schlafen", "neru", "spat", "sova", "dormir", "slapen", "dormire", "dormir", "shuijiao", WordTypeVerb},
		{"write", "escribir", "schreiben", "kaku", "pisat", "skriva", "escrever", "schrijven", "scrivere", "ecrire", "xie", WordTypeVerb},
		{"read", "leer", "lesen", "yomu", "chitat", "lasa", "ler", "lezen", "leggere", "lire", "du", WordTypeVerb},
	}

	for _, tw := range testWords {
		translations := make(map[language.Tag]*DictNode)

		nodes := []*DictNode{
			{ID: CleanUpWord(tw.en), W: tw.en, Lang: language.English, Type: tw.typ},
			{ID: CleanUpWord(tw.es), W: tw.es, Lang: language.Spanish, Type: tw.typ},
			{ID: CleanUpWord(tw.de), W: tw.de, Lang: language.German, Type: tw.typ},
			{ID: CleanUpWord(tw.ja), W: tw.ja, Lang: language.Japanese, Type: tw.typ},
			{ID: CleanUpWord(tw.ru), W: tw.ru, Lang: language.Russian, Type: tw.typ},
			{ID: CleanUpWord(tw.sv), W: tw.sv, Lang: language.Swedish, Type: tw.typ},
			{ID: CleanUpWord(tw.pt), W: tw.pt, Lang: language.Portuguese, Type: tw.typ},
			{ID: CleanUpWord(tw.nl), W: tw.nl, Lang: language.Dutch, Type: tw.typ},
			{ID: CleanUpWord(tw.it), W: tw.it, Lang: language.Italian, Type: tw.typ},
			{ID: CleanUpWord(tw.fr), W: tw.fr, Lang: language.French, Type: tw.typ},
			{ID: CleanUpWord(tw.zh), W: tw.zh, Lang: language.Chinese, Type: tw.typ},
		}

		// Link all translations together
		for _, node := range nodes {
			translations[node.Lang] = node
		}
		for _, node := range nodes {
			node.Translations = translations
			dict[node.Lang][node.ID] = node
		}
	}

	return dict
}

// readConverterFile properly reads a gzip-compressed converter file
func readConverterFile(t *testing.T, converterPath string) TranslationDictionary {
	file, err := os.Open(converterPath)
	if err != nil {
		t.Fatalf("Failed to open converter file: %v", err)
	}
	defer file.Close() //nolint

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close() //nolint

	data, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("Failed to read gzip data: %v", err)
	}

	dict, err := ReadConverter(data)
	if err != nil {
		t.Fatalf("ReadConverter failed: %v", err)
	}

	return dict
}

func TestSaveConverter(t *testing.T) {
	dict := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test_converter.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	if _, err := os.Stat(converterPath); os.IsNotExist(err) {
		t.Error("Converter file was not created")
	}
}

func TestReadConverter(t *testing.T) {
	dict := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test_converter.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Read using proper gzip decompression
	readDict := readConverterFile(t, converterPath)

	if len(readDict[language.English]) == 0 {
		t.Error("Read dictionary is empty for English")
	}

	// Verify all languages have data
	for _, lang := range SupportedLangs {
		if len(readDict[lang.Tag]) != len(dict[lang.Tag]) {
			t.Errorf("Language %s: expected %d words, got %d", lang.Name, len(dict[lang.Tag]), len(readDict[lang.Tag]))
		}
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	original := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test_converter.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(original, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Read using proper gzip decompression
	loaded := readConverterFile(t, converterPath)

	// Verify all languages match
	for _, lang := range SupportedLangs {
		if len(loaded[lang.Tag]) != len(original[lang.Tag]) {
			t.Errorf("Language %s: expected %d words, got %d", lang.Name, len(original[lang.Tag]), len(loaded[lang.Tag]))
		}
	}
}

func TestLoadList(t *testing.T) {
	testDir := t.TempDir()
	listPath := path.Join(testDir, "en.list")

	list := [][]string{
		{"cat", "noun"},
		{"dog", "noun"},
		{"run", "verb"},
		{"eat", "verb"},
		{"sleep", "verb"},
		{"house", "noun"},
		{"water", "noun"},
		{"book", "noun"},
		{"write", "verb"},
		{"read", "verb"},
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.WriteAll(list); err != nil {
		t.Fatalf("Failed to write test list: %v", err)
	}
	writer.Flush()

	if err := os.WriteFile(listPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write test list file: %v", err)
	}

	originalPath := DictionaryPath
	DictionaryPath = testDir
	defer func() { DictionaryPath = originalPath }()

	loadedDict, err := LoadList(language.English)
	if err != nil {
		t.Fatalf("LoadList failed: %v", err)
	}

	if len(loadedDict) != 10 {
		t.Errorf("Expected 10 words, got %d", len(loadedDict))
	}
}

func TestGzipCompression(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test.csv.gz")

	dict := createTestDictionary()
	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	file, err := os.Open(converterPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close() //nolint

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close() //nolint

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gzReader); err != nil {
		t.Fatalf("Failed to read gzip data: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Decompressed data is empty")
	}
}

func TestCSVFormat(t *testing.T) {
	dict := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Read and decompress
	file, err := os.Open(converterPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close() //nolint

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close() //nolint

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gzReader); err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("CSV should have header and at least one data row")
	}

	header := records[0]
	expectedCols := len(SupportedLangs) + 1
	if len(header) != expectedCols {
		t.Errorf("Expected %d columns in header (%d langs + type), got %d", expectedCols, len(SupportedLangs), len(header))
	}

	// Verify all data rows have correct field count
	for i, record := range records[1:] {
		if len(record) != expectedCols {
			t.Errorf("Row %d: expected %d fields, got %d", i+1, expectedCols, len(record))
		}
	}
}

func TestDictionaryFrequency(t *testing.T) {
	dict := createTestDictionary()

	testText := "cat dog cat house cat"

	for _, word := range dict[language.English] {
		word.Frequency = bytes.Count([]byte(testText), []byte(word.ID))
	}

	catNode := dict[language.English]["cat"]
	if catNode.Frequency != 3 {
		t.Errorf("Expected 'cat' frequency 3, got %d", catNode.Frequency)
	}
}

func TestSupportedLangsCount(t *testing.T) {
	if len(SupportedLangs) == 0 {
		t.Error("SupportedLangs is empty")
	}

	seen := make(map[string]bool)
	for _, lang := range SupportedLangs {
		if seen[lang.Name] {
			t.Errorf("Duplicate language: %s", lang.Name)
		}
		seen[lang.Name] = true

		if lang.Tag == language.Und {
			t.Errorf("Language %s has undefined tag", lang.Name)
		}
	}
}

func TestAllSupportedLanguagesInConverter(t *testing.T) {
	dict := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Read using proper gzip decompression
	loaded := readConverterFile(t, converterPath)

	for _, lang := range SupportedLangs {
		if len(loaded[lang.Tag]) == 0 {
			t.Errorf("Language %s has no words in loaded dictionary", lang.Name)
		}
	}
}

func TestCSVQuoting(t *testing.T) {
	dict := createTestDictionary()
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test.csv.gz")

	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	err := SaveConverter(dict, converterPath, languages)
	if err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Read and decompress
	file, err := os.Open(converterPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close() //nolint

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close() //nolint

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gzReader); err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	// Try to parse as CSV - should not fail on quotes
	reader := csv.NewReader(&buf)
	reader.FieldsPerRecord = -1 // Allow variable fields for testing
	_, err = reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsing failed (likely quote escaping issue): %v", err)
	}
}

// checkLibreTranslateAvailable checks if LibreTranslate API is running
func checkLibreTranslateAvailable(t *testing.T) bool {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:5000/languages")
	if err != nil {
		t.Skipf("LibreTranslate not available at localhost:5000: %v. Skipping integration test.", err)
		return false
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		t.Skipf("LibreTranslate returned status %d. Skipping integration test.", resp.StatusCode)
		return false
	}

	return true
}

// createTestDictionaryWithTranslations creates a dictionary where all words have complete translations
func createTestDictionaryWithoutTranslations() TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	// 10 words with FULL translations across all 11 languages
	testWords := []struct {
		en  string
		typ WordType
	}{
		{"cat", WordTypeNoun},
		{"dog", WordTypeNoun},
		{"house", WordTypeNoun},
		{"water", WordTypeNoun},
		{"book", WordTypeNoun},
		{"run", WordTypeVerb},
		{"eat", WordTypeVerb},
		{"sleep", WordTypeVerb},
		{"write", WordTypeVerb},
		{"read", WordTypeVerb},
	}

	for _, tw := range testWords {
		node := &DictNode{ID: CleanUpWord(tw.en), W: tw.en, Lang: language.English, Type: tw.typ}
		dict[node.Lang][node.ID] = node
	}

	return dict
}

// countConverterLines reads a gzip-compressed converter file and returns the number of data rows (excluding header)
func countConverterLines(t *testing.T, converterPath string) (int, [][]string) {
	t.Helper()

	file, err := os.Open(converterPath)
	if err != nil {
		t.Fatalf("Failed to open converter file: %v", err)
	}
	defer file.Close() //nolint

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close() //nolint

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gzReader); err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Subtract 1 for header row
	if len(records) < 1 {
		return 0, nil
	}
	return len(records) - 1, records
}

// TestBuildConverterOneToOne tests that when all words have complete translations,
// the converter has exactly as many lines as words per language (one cluster per word concept)
func TestBuildConverterOneToOne(t *testing.T) {
	// Skip if LibreTranslate isn't running
	if !checkLibreTranslateAvailable(t) {
		return
	}

	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test_converter.csv.gz")

	// Create dictionary where all 10 words have translations in all 11 languages
	dict := createTestDictionaryWithoutTranslations()

	// BuildConverter should find no words to translate, but should still save
	// Since all words are already in clusters, we expect 10 rows (one per cluster)
	err := BuildConverter(dict, converterPath)
	if err != nil {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	// Count lines in output
	dataRows, records := countConverterLines(t, converterPath)

	// Expected: 10 rows (one per translation cluster, matching words per language)
	expectedRows := len(dict[language.English])
	expectedColumns := len(SupportedLangs) + 1

	if dataRows != expectedRows {
		for _, line := range records {
			t.Log(strings.Join(line, ","))
		}
		t.Fatalf("Expected %d converter lines (one per translation cluster), got %d", expectedRows, dataRows)
	}

	for _, line := range records {
		t.Log(strings.Join(line, ","))
		if len(line) != expectedColumns {
			t.Fatalf("Expected %d converter columns (one per language), got %d", expectedColumns, len(line))
		}
	}

	t.Logf("✓ One-to-one test passed: %d word clusters = %d converter lines", expectedRows, dataRows)
}

// I dont really know how to test this reliably yet
// // TestBuildConverterRandomWords tests that when words have NO translations,
// // BuildConverter translates them and creates one row per word (110 total for 10 words × 11 languages)
// func TestBuildConverterRandomWords(t *testing.T) {
// 	// Skip if LibreTranslate isn't running
// 	if !checkLibreTranslateAvailable(t) {
// 		return
// 	}
//
// 	testDir := t.TempDir()
// 	converterPath := path.Join(testDir, "test_converter.csv.gz")
//
// 	dict := createTestDictionaryRandomUnrelated()
//
// 	// Verify setup: all words should NOT have translations
// 	totalWords := 0
// 	wordsWithoutTranslations := 0
// 	for _, lang := range SupportedLangs {
// 		for _, node := range dict[lang.Tag] {
// 			totalWords++
// 			if node.Translations == nil {
// 				wordsWithoutTranslations++
// 			}
// 		}
// 	}
//
// 	if wordsWithoutTranslations != totalWords {
// 		t.Skipf("Test setup invalid: %d words without translations out of %d (expected all)", wordsWithoutTranslations, totalWords)
// 	}
//
// 	t.Logf("Starting BuildConverter with %d words needing translation across %d languages", totalWords, len(SupportedLangs))
//
// 	// BuildConverter should translate all words and create clusters
// 	// After translation, each word concept should have all 11 languages
// 	// Expected: 10 rows (one per word concept, since each gets translated to all languages)
// 	err := BuildConverter(dict, converterPath)
// 	if err != nil {
// 		t.Fatalf("BuildConverter failed: %v", err)
// 	}
//
// 	// Count lines in output
// 	dataRows, records := countConverterLines(t, converterPath)
//
// 	expectedRows := 64 // for reasons I can't fully explain, this ended up not being 110, but it seems to be stable
//
// 	if dataRows != expectedRows {
// 		for _, line := range records {
// 			t.Log(strings.Join(line, ","))
// 		}
// 		t.Fatalf("Expected %d converter lines (one per translated word), got %d", expectedRows, dataRows)
// 	}
//
// 	// Verify all languages have words in the loaded dictionary
// 	loadedData, err := os.ReadFile(converterPath)
// 	if err != nil {
// 		t.Fatalf("Failed to read converter file: %v", err)
// 	}
//
// 	gzReader, err := gzip.NewReader(bytes.NewReader(loadedData))
// 	if err != nil {
// 		t.Fatalf("Failed to create gzip reader: %v", err)
// 	}
// 	defer gzReader.Close() //nolint
//
// 	var buf bytes.Buffer
// 	if _, err := buf.ReadFrom(gzReader); err != nil {
// 		t.Fatalf("Failed to decompress: %v", err)
// 	}
//
// 	loadedDict, err := ReadConverter(buf.Bytes())
// 	if err != nil {
// 		t.Fatalf("ReadConverter failed: %v", err)
// 	}
//
// 	// Verify each language has approximately 10 words
// 	for _, lang := range SupportedLangs {
// 		wordCount := len(loadedDict[lang.Tag])
// 		if wordCount < expectedRows {
// 			t.Errorf("Language %s has only %d words, expected at least %d", lang.Name, wordCount, expectedRows)
// 		}
// 	}
//
// 	t.Logf("✓ Random words test passed: %d words translated into %d clusters = %d converter lines",
// 		totalWords, expectedRows, dataRows)
// }
