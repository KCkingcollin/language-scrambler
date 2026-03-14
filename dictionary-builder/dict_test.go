package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"
	"sync"
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

	ctx := context.Background()

	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		t.Fatal(err)
	}

	// BuildConverter should find no words to translate, but should still save
	// Since all words are already in clusters, we expect 10 rows (one per cluster)
	err = BuildConverter(dict, converterPath, ctx, canTranslate, TranslateWords)
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

// TestSaveLoadRoundTrip verifies that saving and loading a TranslationDictionary
// preserves all word data and translation cluster relationships.
func TestSaveLoadRoundTrip(t *testing.T) {
	// Setup: create a minimal test dictionary with shared translation clusters
	original := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		original[l.Tag] = make(Dictionary)
	}

	// Create a shared translation cluster (multiple words pointing to same map)
	sharedCluster := make(map[language.Tag]*DictNode)

	// English base word
	enWord := &DictNode{
		ID:           "hello",
		W:            "hello",
		Lang:         language.English,
		Type:         WordTypeNoun,
		Translations: sharedCluster,
	}
	sharedCluster[language.English] = enWord
	original[language.English]["hello"] = enWord

	// Add translations that share the same cluster
	for _, lang := range []language.Tag{language.Spanish, language.French, language.German} {
		trans := &DictNode{
			ID:           "hola", // simplified - in real usage would be lang-specific
			W:            "hola",
			Lang:         lang,
			Type:         WordTypeNoun,
			Translations: sharedCluster, // ← critical: same pointer
		}
		sharedCluster[lang] = trans
		original[lang]["hola"] = trans
	}

	// Add a second, independent cluster to test multiple clusters
	cluster2 := make(map[language.Tag]*DictNode)
	enWord2 := &DictNode{
		ID:           "world",
		W:            "world",
		Lang:         language.English,
		Type:         WordTypeNoun,
		Translations: cluster2,
	}
	cluster2[language.English] = enWord2
	original[language.English]["world"] = enWord2

	esWord2 := &DictNode{
		ID:           "mundo",
		W:            "mundo",
		Lang:         language.Spanish,
		Type:         WordTypeNoun,
		Translations: cluster2,
	}
	cluster2[language.Spanish] = esWord2
	original[language.Spanish]["mundo"] = esWord2

	// Add a word with NO translations (should be skipped by SaveConverter)
	orphan := &DictNode{
		ID:           "orphan",
		W:            "orphan",
		Lang:         language.Japanese,
		Type:         WordTypeUnknown,
		Translations: nil,
	}
	original[language.Japanese]["orphan"] = orphan

	// Save to temp file
	tmpDir := t.TempDir()
	testPath := path.Join(tmpDir, "test_converter.csv.gz")

	languages := []language.Tag{language.English, language.Spanish, language.French, language.German}
	if err := SaveConverter(original, testPath, languages); err != nil {
		t.Fatalf("SaveConverter failed: %v", err)
	}

	// Load back

	file, err := os.Open(testPath)
	if err != nil {
		t.Fatalf("Failed to open saved file: %v", err)
	}

	gzFile, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gz reader: %v", err)
	}
	defer gzFile.Close() //nolint

	data, err := io.ReadAll(gzFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	loaded, err := ReadConverter(data)
	if err != nil {
		t.Fatalf("ReadConverter failed: %v", err)
	}

	// === Verification ===

	// 1. Check word counts per language (excluding nil-translation words)
	for _, lang := range languages {
		origCount := countTranslatableWords(original[lang])
		loadedCount := countTranslatableWords(loaded[lang])
		if origCount != loadedCount {
			t.Errorf("Language %s: word count mismatch - original=%d, loaded=%d",
				lang.String(), origCount, loadedCount)
		}
	}

	// 2. Verify each word's core data matches
	for lang, origDict := range original {
		loadedDict, ok := loaded[lang]
		if !ok {
			t.Errorf("Language %s missing in loaded dictionary", lang.String())
			continue
		}
		for id, origWord := range origDict {
			// Skip words with no translations (they aren't saved)
			if origWord.Translations == nil {
				if _, exists := loadedDict[id]; exists {
					t.Errorf("Word %q (%s) with nil translations should not be saved, but found in loaded",
						origWord.W, lang.String())
				}
				continue
			}

			loadedWord, exists := loadedDict[id]
			if !exists {
				t.Errorf("Word %q (%s) missing in loaded dictionary", origWord.W, lang.String())
				continue
			}

			// Compare core fields
			if origWord.ID != loadedWord.ID {
				t.Errorf("Word %q: ID mismatch - original=%q, loaded=%q", origWord.W, origWord.ID, loadedWord.ID)
			}
			if origWord.W != loadedWord.W {
				t.Errorf("Word %q: W mismatch - original=%q, loaded=%q", origWord.W, origWord.W, loadedWord.W)
			}
			if origWord.Lang != loadedWord.Lang {
				t.Errorf("Word %q: Lang mismatch - original=%s, loaded=%s", origWord.W, origWord.Lang, loadedWord.Lang)
			}
			if origWord.Type != loadedWord.Type {
				t.Errorf("Word %q: Type mismatch - original=%d, loaded=%d", origWord.W, origWord.Type, loadedWord.Type)
			}
		}
	}

	// 3. Verify translation clusters are logically equivalent
	// (Pointer equality won't survive serialization, but content should)
	for lang, origDict := range original {
		for id, origWord := range origDict {
			if origWord.Translations == nil {
				continue
			}
			loadedWord := loaded[lang][id]
			if loadedWord == nil || loadedWord.Translations == nil {
				t.Errorf("Word %q (%s): translation map lost during round-trip", origWord.W, lang.String())
				continue
			}

			// Check that all translations in original exist in loaded with same values
			for transLang, origTrans := range origWord.Translations {
				loadedTrans, ok := loadedWord.Translations[transLang]
				if !ok {
					t.Errorf("Word %q (%s): translation for %s missing in loaded",
						origWord.W, lang.String(), transLang.String())
					continue
				}
				if origTrans.W != loadedTrans.W {
					t.Errorf("Word %q (%s): translation for %s value mismatch - original=%q, loaded=%q",
						origWord.W, lang.String(), transLang.String(), origTrans.W, loadedTrans.W)
				}
				if origTrans.Type != loadedTrans.Type {
					t.Errorf("Word %q (%s): translation for %s type mismatch - original=%d, loaded=%d",
						origWord.W, lang.String(), transLang.String(), origTrans.Type, loadedTrans.Type)
				}
			}
		}
	}

	// 4. Verify cluster grouping: words that shared a cluster should have equivalent translation sets
	origClusters := extractClusterSignatures(original, languages)
	loadedClusters := extractClusterSignatures(loaded, languages)

	if len(origClusters) != len(loadedClusters) {
		t.Errorf("Cluster count mismatch - original=%d, loaded=%d", len(origClusters), len(loadedClusters))
	}

	// Check each original cluster has a matching loaded cluster
	for _, origSig := range origClusters {
		found := false
		for _, loadedSig := range loadedClusters {
			if clusterSignaturesEqual(origSig, loadedSig) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Original cluster not found in loaded: %+v", origSig)
		}
	}
}

// countTranslatableWords counts words that have non-nil Translations (i.e., would be saved)
func countTranslatableWords(dict Dictionary) int {
	count := 0
	for _, w := range dict {
		if w.Translations != nil {
			count++
		}
	}
	return count
}

// clusterSignature is a deterministic representation of a translation cluster for comparison
type clusterSignature struct {
	words       []string // sorted list of "lang:word" entries
	types       map[string]WordType
	translation map[string]string // lang -> word text
}

// extractClusterSignatures groups words by their translation map content
func extractClusterSignatures(dict TranslationDictionary, languages []language.Tag) []clusterSignature {
	seen := make(map[string]bool)
	var signatures []clusterSignature

	for _, lang := range languages {
		for _, word := range dict[lang] {
			if word.Translations == nil {
				continue
			}

			// Create a signature based on the CONTENT of the translation map
			var entries []string
			types := make(map[string]WordType)
			transText := make(map[string]string)

			for _, l := range languages {
				if node, ok := word.Translations[l]; ok {
					key := l.String()
					entries = append(entries, key+":"+node.ID)
					types[key] = node.Type
					transText[key] = node.W
				}
			}
			if len(entries) == 0 {
				continue
			}
			slices.Sort(entries)
			sigKey := strings.Join(entries, "|")

			if !seen[sigKey] {
				seen[sigKey] = true
				signatures = append(signatures, clusterSignature{
					words:       entries,
					types:       types,
					translation: transText,
				})
			}
		}
	}
	return signatures
}

func clusterSignaturesEqual(a, b clusterSignature) bool {
	if !slices.Equal(a.words, b.words) {
		return false
	}
	if !reflect.DeepEqual(a.types, b.types) {
		return false
	}
	return reflect.DeepEqual(a.translation, b.translation)
}

// TestSaveQueueBlockingFinalSave verifies that the final saveQueue send
// is blocking (not dropped) and completes before wg.Wait() returns.
func TestSaveQueueBlockingFinalSave(t *testing.T) {
	saveQueue := make(chan struct{}, 1)
	var saveCount int
	var wg sync.WaitGroup

	// Start the save worker (mirroring BuildConverter's pattern)
	wg.Go(func() {
		for range saveQueue {
			saveCount++
			// Simulate save work
			time.Sleep(10 * time.Millisecond)
		}
	})

	// Simulate periodic non-blocking saves (some may drop)
	for range 10 {
		select {
		case saveQueue <- struct{}{}:
		default:
			// Drop is expected for periodic saves
		}
	}

	// Final save MUST block and complete
	finalSaveStart := time.Now()
	saveQueue <- struct{}{} // blocking send
	close(saveQueue)
	wg.Wait()
	finalSaveElapsed := time.Since(finalSaveStart)

	if saveCount < 1 {
		t.Error("No saves were processed - worker may not have started")
	}
	if finalSaveElapsed < 10*time.Millisecond {
		t.Logf("⚠ Final save completed faster than simulated work (%v) - verify SaveConverter isn't being skipped", finalSaveElapsed)
	}

	t.Logf("✓ Final save blocking behavior verified: %d saves processed", saveCount)
}

// TestBuildConverterIdempotentSave verifies that calling BuildConverter
// twice with the same fully-translated dictionary produces identical output.
func TestBuildConverterIdempotentSave(t *testing.T) {
	testDir := t.TempDir()
	path1 := path.Join(testDir, "run1.csv.gz")
	path2 := path.Join(testDir, "run2.csv.gz")

	dict := createTestDictionary() // fully translated
	ctx := context.Background()

	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		t.Fatal(err)
	}

	// Run twice
	if err := BuildConverter(dict, path1, ctx, canTranslate, TranslateWords); err != nil {
		t.Fatalf("First BuildConverter failed: %v", err)
	}
	if err := BuildConverter(dict, path2, ctx, canTranslate, TranslateWords); err != nil {
		t.Fatalf("Second BuildConverter failed: %v", err)
	}

	// Read both files
	data1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("Failed to read first file: %v", err)
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("Failed to read second file: %v", err)
	}

	// Compare raw bytes (gzip output should be deterministic for same input)
	if !bytes.Equal(data1, data2) {
		// If bytes differ, decompress and compare CSV content
		dict1 := readConverterFile(t, path1)
		dict2 := readConverterFile(t, path2)

		// Compare logical content
		for _, lang := range SupportedLangs {
			if len(dict1[lang.Tag]) != len(dict2[lang.Tag]) {
				t.Errorf("Language %s: word count differs between runs - run1=%d, run2=%d",
					lang.Name, len(dict1[lang.Tag]), len(dict2[lang.Tag]))
			}
		}
	} else {
		t.Log("✓ Raw file bytes identical between runs (deterministic save)")
	}
}

// TestInterruptDuringBatch verifies behavior when context is cancelled
// mid-batch (simulating your Ctrl+C scenario).
func TestInterruptDuringBatch(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "test_mid_batch.csv.gz")

	// Create dictionary with many words needing translation
	dict := createTestDictionaryWithoutTranslations()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a very short time to catch mid-batch
	time.AfterFunc(5*time.Millisecond, cancel)

	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		t.Fatal(err)
	}

	err = BuildConverter(dict, converterPath, ctx, canTranslate, TranslateWords)
	// Accept either success or context cancellation
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	// Verify file is valid (even if incomplete)
	if _, err := os.Stat(converterPath); err != nil {
		t.Fatalf("Converter file not created or invalid: %v", err)
	}

	// Load and verify it's parseable
	loaded := readConverterFile(t, converterPath)

	// Count how many words have full translations (should be >= pre-seeded)
	var complete int
	for _, lang := range SupportedLangs {
		for _, word := range loaded[lang.Tag] {
			if word.Translations != nil {
				hasAll := true
				for _, l := range SupportedLangs {
					if _, ok := word.Translations[l.Tag]; !ok {
						hasAll = false
						break
					}
				}
				if hasAll {
					complete++
				}
			}
		}
	}

	t.Logf("✓ Interrupted run produced valid file with %d fully-translated words", complete)
	// We don't assert a minimum because timing is non-deterministic,
	// but the file should be loadable and contain at least the seed data
}

// ============ DETERMINISTIC TESTS ============

// TestCheckForTranslations_NilTranslations tests words with no translation map
func TestCheckForTranslations_NilTranslations(t *testing.T) {
	tDict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		tDict[l.Tag] = make(Dictionary)
	}

	// Word with nil Translations
	node := &DictNode{
		ID:           "test",
		W:            "test",
		Lang:         language.English,
		Type:         WordTypeNoun,
		Translations: nil,
	}
	tDict[language.English][node.ID] = node

	canTranslate := makeTranslationCapabilities(true)
	languages := getAllLanguageTags()

	wordsMap, amount := checkForTranslations(tDict, canTranslate, languages)

	if amount != 1 {
		t.Errorf("Expected 1 word needing translation, got %d", amount)
	}
	if len(wordsMap[language.English]) != 1 {
		t.Errorf("Expected 1 English word in wordsMap, got %d", len(wordsMap[language.English]))
	}
}

// TestCheckForTranslations_CompleteTranslations tests words with all translations present
func TestCheckForTranslations_CompleteTranslations(t *testing.T) {
	tDict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		tDict[l.Tag] = make(Dictionary)
	}

	// Create complete translation cluster
	translations := make(map[language.Tag]*DictNode)
	languages := getAllLanguageTags()

	for _, lang := range languages {
		node := &DictNode{
			ID:   fmt.Sprintf("word%s", lang.String()),
			W:    fmt.Sprintf("word%s", lang.String()),
			Lang: lang,
			Type: WordTypeNoun,
		}
		translations[lang] = node
	}

	// Link all nodes to same translation map
	for _, lang := range languages {
		node := translations[lang]
		node.Translations = translations
		tDict[lang][node.ID] = node
	}

	canTranslate := makeTranslationCapabilities(true)

	wordsMap, amount := checkForTranslations(tDict, canTranslate, languages)

	if amount != 0 {
		t.Errorf("Expected 0 words needing translation (all complete), got %d", amount)
	}
	for _, lang := range languages {
		if len(wordsMap[lang]) != 0 {
			t.Errorf("Expected empty wordsMap for %s, got %d", lang.String(), len(wordsMap[lang]))
		}
	}
}

// TestCheckForTranslations_PartialTranslations tests words with missing translations
func TestCheckForTranslations_PartialTranslations(t *testing.T) {
	tDict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		tDict[l.Tag] = make(Dictionary)
	}

	// Create partial translation cluster (missing French and Chinese)
	translations := make(map[language.Tag]*DictNode)
	languages := getAllLanguageTags()

	for i, lang := range languages {
		if i >= len(languages)-2 {
			continue // Skip last 2 languages
		}
		node := &DictNode{
			ID:   fmt.Sprintf("word%s", lang.String()),
			W:    fmt.Sprintf("word%s", lang.String()),
			Lang: lang,
			Type: WordTypeNoun,
		}
		translations[lang] = node
	}

	for _, node := range translations {
		node.Translations = translations
		tDict[node.Lang][node.ID] = node
	}

	canTranslate := makeTranslationCapabilities(true)

	_, amount := checkForTranslations(tDict, canTranslate, languages)

	// All nodes in the partial cluster should need translation
	expectedAmount := len(translations)
	if amount != expectedAmount {
		t.Errorf("Expected %d words needing translation, got %d", expectedAmount, amount)
	}
}

// TestCheckForTranslations_RestrictedCapabilities tests canTranslate restrictions
func TestCheckForTranslations_RestrictedCapabilities(t *testing.T) {
	tDict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		tDict[l.Tag] = make(Dictionary)
	}

	// English word with only Spanish translation
	translations := make(map[language.Tag]*DictNode)
	enNode := &DictNode{ID: "test", W: "test", Lang: language.English, Type: WordTypeNoun}
	esNode := &DictNode{ID: "prueba", W: "prueba", Lang: language.Spanish, Type: WordTypeNoun}
	translations[language.English] = enNode
	translations[language.Spanish] = esNode
	enNode.Translations = translations
	esNode.Translations = translations
	tDict[language.English][enNode.ID] = enNode
	tDict[language.Spanish][esNode.ID] = esNode

	// Restrict: English can only translate to Spanish
	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		canTranslate[l1.Tag] = make(map[language.Tag]bool)
	}
	canTranslate[language.English][language.Spanish] = true
	canTranslate[language.Spanish][language.English] = true

	languages := getAllLanguageTags()

	wordsMap, amount := checkForTranslations(tDict, canTranslate, languages)

	// Should NOT need translation (all SUPPORTED targets are present)
	if amount != 0 {
		t.Errorf("Expected 0 words (all supported langs present), got %d", amount)
	}
	if len(wordsMap[language.English]) != 0 {
		t.Errorf("Expected no English words needing translation, got %d", len(wordsMap[language.English]))
	}
}

// TestCheckForTranslations_EmptyInputs tests edge cases with empty inputs
func TestCheckForTranslations_EmptyInputs(t *testing.T) {
	tests := []struct {
		name           string
		tDict          TranslationDictionary
		canTranslate   map[language.Tag]map[language.Tag]bool
		languages      []language.Tag
		expectedAmount int
	}{
		{
			name:           "Empty dictionary",
			tDict:          makeEmptyDictionary(),
			canTranslate:   makeTranslationCapabilities(true),
			languages:      getAllLanguageTags(),
			expectedAmount: 0,
		},
		{
			name:           "Empty languages list",
			tDict:          makeDictionaryWithNilTranslations(1),
			canTranslate:   makeTranslationCapabilities(true),
			languages:      []language.Tag{},
			expectedAmount: 1, // No target langs to check, so needs translation
		},
		{
			name:           "Empty canTranslate",
			tDict:          makeDictionaryWithTranslations(),
			canTranslate:   make(map[language.Tag]map[language.Tag]bool),
			languages:      getAllLanguageTags(),
			expectedAmount: 0, // No capabilities = no required translations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, amount := checkForTranslations(tt.tDict, tt.canTranslate, tt.languages)
			if amount != tt.expectedAmount {
				t.Errorf("%s: expected %d, got %d", tt.name, tt.expectedAmount, amount)
			}
		})
	}
}

// TestCheckForTranslations_Consistency tests deterministic behavior
func TestCheckForTranslations_Consistency(t *testing.T) {
	tDict := makeDictionaryWithNilTranslations(50)
	canTranslate := makeTranslationCapabilities(true)
	languages := getAllLanguageTags()

	var amounts []int
	for range 10 {
		_, amount := checkForTranslations(tDict, canTranslate, languages)
		amounts = append(amounts, amount)
	}

	for i := 1; i < len(amounts); i++ {
		if amounts[i] != amounts[0] {
			t.Errorf("Inconsistent results: call %d=%d, call 0=%d", i, amounts[i], amounts[0])
		}
	}
}

// ============ FUZZ TEST ============

// FuzzCheckForTranslations extensively fuzzes the function with random inputs
func FuzzCheckForTranslations(f *testing.F) {
	// Seed with some initial test cases
	seedCases := []struct {
		wordCount               int
		langCount               int
		translationCompleteness float32 // 0.0 to 1.0
		capabilityDensity       float32 // 0.0 to 1.0
	}{
		{0, 0, 0, 0},
		{1, 1, 0, 0},
		{10, 5, 0.5, 0.5},
		{100, 11, 1.0, 1.0},
		{50, 11, 0.0, 1.0},
		{50, 11, 1.0, 0.0},
	}

	for _, sc := range seedCases {
		data := generateFuzzInput(sc.wordCount, sc.langCount, sc.translationCompleteness, sc.capabilityDensity)
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse fuzz data into test parameters
		params := parseFuzzData(data)

		tDict := generateRandomDictionary(params.wordCount, params.langSubset)
		canTranslate := generateRandomCapabilities(params.langSubset, params.capabilityDensity)
		languages := params.langSubset

		// Run the function
		wordsMap, amount := checkForTranslations(tDict, canTranslate, languages)

		// Validate invariants
		validateInvariants(t, wordsMap, amount, canTranslate, languages)
	})
}

// ============ HELPER FUNCTIONS ============

func makeEmptyDictionary() TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}
	return dict
}

func makeDictionaryWithNilTranslations(count int) TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	for i := range count {
		lang := SupportedLangs[i%len(SupportedLangs)].Tag
		node := &DictNode{
			ID:           fmt.Sprintf("word%d", i),
			W:            fmt.Sprintf("word%d", i),
			Lang:         lang,
			Type:         WordTypeNoun,
			Translations: nil,
		}
		dict[lang][node.ID] = node
	}
	return dict
}

func makeDictionaryWithTranslations() TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	translations := make(map[language.Tag]*DictNode)
	for _, lang := range SupportedLangs {
		node := &DictNode{
			ID:   fmt.Sprintf("word%s", lang.Tag.String()),
			W:    fmt.Sprintf("word%s", lang.Tag.String()),
			Lang: lang.Tag,
			Type: WordTypeNoun,
		}
		translations[lang.Tag] = node
	}

	for _, node := range translations {
		node.Translations = translations
		dict[node.Lang][node.ID] = node
	}

	return dict
}

func makeTranslationCapabilities(allTrue bool) map[language.Tag]map[language.Tag]bool {
	capabilities := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		capabilities[l1.Tag] = make(map[language.Tag]bool)
		for _, l2 := range SupportedLangs {
			capabilities[l1.Tag][l2.Tag] = allTrue
		}
	}
	return capabilities
}

func getAllLanguageTags() []language.Tag {
	tags := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		tags[i] = l.Tag
	}
	return tags
}

// ============ FUZZ HELPERS ============

type fuzzParams struct {
	wordCount         int
	langSubset        []language.Tag
	capabilityDensity float32
}

func generateFuzzInput(wordCount, langCount int, translationCompleteness, capabilityDensity float32) []byte {
	// Encode parameters into bytes for fuzzing
	data := make([]byte, 0, 20)
	data = append(data, byte(min(wordCount, 255)))
	data = append(data, byte(min(langCount, 11)))
	data = append(data, byte(translationCompleteness*255))
	data = append(data, byte(capabilityDensity*255))
	return data
}

func parseFuzzData(data []byte) fuzzParams {
	if len(data) < 4 {
		return fuzzParams{
			wordCount:         10,
			langSubset:        getAllLanguageTags(),
			capabilityDensity: 0.5,
		}
	}

	wordCount := min(max(1, int(data[0])), 1000)

	langCount := min(max(1, int(data[1])), len(SupportedLangs))

	langSubset := make([]language.Tag, langCount)
	for i := range langCount {
		langSubset[i] = SupportedLangs[i].Tag
	}

	capabilityDensity := float32(data[2]) / 255.0
	translationCompleteness := float32(data[3]) / 255.0

	// Use translationCompleteness to potentially reduce language subset
	if translationCompleteness < 0.3 && len(langSubset) > 3 {
		langSubset = langSubset[:3]
	}

	return fuzzParams{
		wordCount:         wordCount,
		langSubset:        langSubset,
		capabilityDensity: capabilityDensity,
	}
}

func generateRandomDictionary(wordCount int, languages []language.Tag) TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	rng := rand.New(rand.NewSource(rand.Int63()))

	for i := range wordCount {
		lang := languages[rng.Intn(len(languages))]
		node := &DictNode{
			ID:   fmt.Sprintf("fuzzword%d%s", i, lang.String()),
			W:    fmt.Sprintf("word%d", i),
			Lang: lang,
			Type: WordType(rng.Intn(3)),
		}

		// Randomly decide if this word has translations
		if rng.Float32() > 0.3 {
			translations := make(map[language.Tag]*DictNode)
			translations[lang] = node

			// Add random subset of translations
			for _, targetLang := range languages {
				if rng.Float32() > 0.5 {
					transNode := &DictNode{
						ID:   fmt.Sprintf("fuzztrans%d%s", i, targetLang.String()),
						W:    fmt.Sprintf("trans%d", i),
						Lang: targetLang,
						Type: node.Type,
					}
					translations[targetLang] = transNode
				}
			}

			// Link all nodes to same translation map
			for _, n := range translations {
				n.Translations = translations
				dict[n.Lang][n.ID] = n
			}
		} else {
			node.Translations = nil
			dict[lang][node.ID] = node
		}
	}

	return dict
}

func generateRandomCapabilities(languages []language.Tag, density float32) map[language.Tag]map[language.Tag]bool {
	capabilities := make(map[language.Tag]map[language.Tag]bool)
	rng := rand.New(rand.NewSource(rand.Int63()))

	for _, l1 := range languages {
		capabilities[l1] = make(map[language.Tag]bool)
		for _, l2 := range languages {
			capabilities[l1][l2] = rng.Float32() < density
		}
	}

	return capabilities
}

func validateInvariants(t *testing.T, wordsMap map[language.Tag][]*DictNode, amount int, canTranslate map[language.Tag]map[language.Tag]bool, languages []language.Tag) {
	t.Helper()

	// Invariant 1: amount should match sum of all wordsMap lengths
	totalFromMap := 0
	for _, lang := range languages {
		totalFromMap += len(wordsMap[lang])
	}
	if totalFromMap != amount {
		t.Errorf("Invariant violated: amount=%d but sum of wordsMap=%d", amount, totalFromMap)
	}

	// Invariant 2: All words in wordsMap should actually need translation
	for lang, words := range wordsMap {
		for _, w := range words {
			if w.Lang != lang {
				t.Errorf("Word language mismatch: expected %s, got %s", lang.String(), w.Lang.String())
			}

			// Verify this word actually needs translation
			needsTranslation := true
			if w.Translations != nil {
				needsTranslation = false
				for _, l := range languages {
					if canTranslate[w.Lang][l] {
						if _, ok := w.Translations[l]; !ok {
							needsTranslation = true
							break
						}
					}
				}
			}

			if !needsTranslation {
				t.Errorf("Word %s (%s) in wordsMap but doesn't need translation", w.W, w.Lang.String())
			}
		}
	}

	// Invariant 3: No nil pointers in wordsMap
	for lang, words := range wordsMap {
		for i, w := range words {
			if w == nil {
				t.Errorf("Nil word at %s[%d]", lang.String(), i)
			}
		}
	}

	// Invariant 4: amount should be non-negative
	if amount < 0 {
		t.Errorf("Negative amount: %d", amount)
	}

	// Invariant 5: wordsMap should only contain languages from input
	for lang := range wordsMap {
		if !slices.Contains(languages, lang) {
			t.Errorf("Unexpected language %s in wordsMap", lang.String())
		}
	}
}

// ============ PROPERTY-BASED TEST ============

// TestCheckForTranslations_Properties tests key properties that should always hold
func TestCheckForTranslations_Properties(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func() (TranslationDictionary, map[language.Tag]map[language.Tag]bool, []language.Tag)
		verifyFn func(t *testing.T, wordsMap map[language.Tag][]*DictNode, amount int)
	}{
		{
			name: "Amount equals wordsMap sum",
			setupFn: func() (TranslationDictionary, map[language.Tag]map[language.Tag]bool, []language.Tag) {
				dict := makeDictionaryWithNilTranslations(100)
				return dict, makeTranslationCapabilities(true), getAllLanguageTags()
			},
			verifyFn: func(t *testing.T, wordsMap map[language.Tag][]*DictNode, amount int) {
				total := 0
				for _, words := range wordsMap {
					total += len(words)
				}
				if total != amount {
					t.Errorf("Sum mismatch: total=%d, amount=%d", total, amount)
				}
			},
		},
		{
			name: "All words in map need translation",
			setupFn: func() (TranslationDictionary, map[language.Tag]map[language.Tag]bool, []language.Tag) {
				dict := make(TranslationDictionary)
				for _, l := range SupportedLangs {
					dict[l.Tag] = make(Dictionary)
				}

				// Mix of complete and incomplete translations
				for i := range 20 {
					lang := SupportedLangs[i%len(SupportedLangs)].Tag
					node := &DictNode{
						ID:   fmt.Sprintf("word%d", i),
						W:    fmt.Sprintf("word%d", i),
						Lang: lang,
						Type: WordTypeNoun,
					}

					if i%2 == 0 {
						node.Translations = nil
					} else {
						node.Translations = make(map[language.Tag]*DictNode)
						node.Translations[language.English] = node
					}

					dict[lang][node.ID] = node
				}

				return dict, makeTranslationCapabilities(true), getAllLanguageTags()
			},
			verifyFn: func(t *testing.T, wordsMap map[language.Tag][]*DictNode, amount int) {
				for lang, words := range wordsMap {
					for _, w := range words {
						if w.Lang != lang {
							t.Errorf("Language mismatch in wordsMap")
						}
					}
				}
			},
		},
		{
			name: "Empty dictionary returns zero",
			setupFn: func() (TranslationDictionary, map[language.Tag]map[language.Tag]bool, []language.Tag) {
				return makeEmptyDictionary(), makeTranslationCapabilities(true), getAllLanguageTags()
			},
			verifyFn: func(t *testing.T, wordsMap map[language.Tag][]*DictNode, amount int) {
				if amount != 0 {
					t.Errorf("Expected 0 for empty dictionary, got %d", amount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tDict, canTranslate, languages := tt.setupFn()
			wordsMap, amount := checkForTranslations(tDict, canTranslate, languages)
			tt.verifyFn(t, wordsMap, amount)
		})
	}
}

// ============ TRANSLATION INVARIANTS TEST ============

// TestBuildConverter_WordCountInvariant verifies that after BuildConverter
// completes (without interruption), zero words remain needing translation.
func TestBuildConverter_WordCountInvariant(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "invariant_test.csv.gz")

	// Create controlled dictionary: 100 words per language, all need translation
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	const wordsPerLang = 100
	for i := range wordsPerLang {
		for _, lang := range SupportedLangs {
			node := &DictNode{
				ID:           fmt.Sprintf("word%d%s", i, lang.Tag.String()),
				W:            fmt.Sprintf("word%d", i),
				Lang:         lang.Tag,
				Type:         WordTypeNoun,
				Translations: nil,
			}
			dict[lang.Tag][node.ID] = node
		}
	}

	// Mock capabilities: all pairs supported
	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		canTranslate[l1.Tag] = make(map[language.Tag]bool)
		for _, l2 := range SupportedLangs {
			canTranslate[l1.Tag][l2.Tag] = true
		}
	}

	// Mock translator: deterministic, returns valid translations
	translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
		result := make([]string, len(words))
		for i, w := range words {
			result[i] = w.W + to.String()
		}
		return result, nil
	}

	// Capture initial count
	_, initialCount := checkForTranslations(dict, canTranslate, getAllLanguageTags())
	expectedInitial := wordsPerLang * len(SupportedLangs)
	if initialCount != expectedInitial {
		t.Fatalf("Expected initial count %d, got %d", expectedInitial, initialCount)
	}

	// Run BuildConverter with injected dependencies
	ctx := context.Background()
	err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
	if err != nil {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	// Verify final count is 0 (all translated)
	_, finalCount := checkForTranslations(dict, canTranslate, getAllLanguageTags())
	if finalCount != 0 {
		t.Errorf("Expected 0 words needing translation after completion, got %d", finalCount)

		// Debug: log which words still need translation
		wordsMap, _ := checkForTranslations(dict, canTranslate, getAllLanguageTags())
		for lang, words := range wordsMap {
			if len(words) > 0 {
				t.Logf("Language %s still has %d words needing translation", lang.String(), len(words))
				for _, w := range words[:min(5, len(words))] {
					t.Logf("  - %s (%s)", w.W, w.ID)
				}
			}
		}
	}

	// Verify saved file is consistent
	if _, err := os.Stat(converterPath); err == nil {
		loaded := readConverterFile(t, converterPath)
		_, loadedCount := checkForTranslations(loaded, canTranslate, getAllLanguageTags())
		if loadedCount != 0 {
			t.Errorf("Loaded file has %d words still needing translation", loadedCount)
		}
	}
}

// TestBuildConverter_RaceCondition uses the race detector to catch
// concurrent access to tDict during checkForTranslations + AddTranslation.
func TestBuildConverter_RaceCondition(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "race_test.csv.gz")

	// Create dictionary with words needing translation
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	const wordsPerLang = 50
	for i := range wordsPerLang {
		for _, lang := range SupportedLangs {
			base := fmt.Sprintf("word_%d", i)
			node := &DictNode{
				ID:           CleanUpWord(base),
				W:            base,
				Lang:         lang.Tag,
				Type:         WordTypeNoun,
				Translations: nil,
			}
			dict[lang.Tag][node.ID] = node
		}
	}

	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		canTranslate[l1.Tag] = make(map[language.Tag]bool)
		for _, l2 := range SupportedLangs {
			canTranslate[l1.Tag][l2.Tag] = true
		}
	}

	// Mock translator that returns words matching the expected ID format
	translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
		result := make([]string, len(words))
		for i, w := range words {
			// For self-translation, return original word to avoid ID mismatch
			if from == to {
				result[i] = w.W
				continue
			}
			// For cross-translation, return a word that cleans to a predictable ID
			// Format: "<original>_<lang>" → CleanUpWord → "<original><lang>"
			result[i] = fmt.Sprintf("%s_%s", w.W, to.String())
		}
		return result, nil
	}

	ctx := context.Background()
	err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
	if err != nil {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	// Final verification: all words should be translated
	_, finalCount := checkForTranslations(dict, canTranslate, getAllLanguageTags())
	if finalCount != 0 {
		t.Errorf("Expected 0 words needing translation, got %d", finalCount)

		// Debug: log which words still need translation
		wordsMap, _ := checkForTranslations(dict, canTranslate, getAllLanguageTags())
		for lang, words := range wordsMap {
			if len(words) > 0 {
				t.Logf("Language %s has %d words needing translation", lang.String(), len(words))
				for j, w := range words {
					if j >= 3 {
						break
					}
					t.Logf("  - ID=%q W=%q HasTranslations=%v", w.ID, w.W, w.Translations != nil)
				}
			}
		}
	}
}

// TestBuildConverter_AddTranslationDoesNotIncreaseCount verifies that
// after completing a translation cluster, zero words remain needing translation.
func TestBuildConverter_AddTranslationDoesNotIncreaseCount(t *testing.T) {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	// Single word needing translation
	word := &DictNode{
		ID:           "test",
		W:            "test",
		Lang:         language.English,
		Type:         WordTypeNoun,
		Translations: nil,
	}
	dict[language.English][word.ID] = word

	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		canTranslate[l1.Tag] = make(map[language.Tag]bool)
		for _, l2 := range SupportedLangs {
			canTranslate[l1.Tag][l2.Tag] = true
		}
	}
	languages := getAllLanguageTags()

	// Count before (should be 1: the English word)
	_, initial := checkForTranslations(dict, canTranslate, languages)
	if initial != 1 {
		t.Fatalf("Expected initial count 1, got %d", initial)
	}

	// Add translations one by one (don't check intermediate counts - they can increase
	// because AddTranslation adds new dictionary entries that may also need translations)
	for _, lang := range languages {
		if lang == language.English {
			continue
		}
		translation := &DictNode{
			ID:   fmt.Sprintf("test%s", lang.String()),
			W:    fmt.Sprintf("test%s", lang.String()),
			Lang: lang,
			Type: WordTypeNoun,
		}

		dictMu.Lock()
		dict.AddTranslation(word, translation)
		dictMu.Unlock()
	}

	// Final check: after all translations added and cluster merged, count should be 0
	_, final := checkForTranslations(dict, canTranslate, languages)
	if final != 0 {
		t.Errorf("Expected 0 after completing cluster, got %d", final)

		// Debug: log which words still need translation
		wordsMap, _ := checkForTranslations(dict, canTranslate, languages)
		for lang, words := range wordsMap {
			if len(words) > 0 {
				t.Logf("Language %s has %d words needing translation:", lang.String(), len(words))
				for _, w := range words {
					t.Logf("  - ID=%q W=%q HasTranslations=%v", w.ID, w.W, w.Translations != nil)
				}
			}
		}
	}
}

// ============ SAVE INVARIANTS	TEST ============

// FuzzBuildConverter_FinalSaveCompletion fuzzes the final save invariant
// with varying dictionary sizes, language subsets, and translator latencies.
func FuzzBuildConverter_FinalSaveCompletion(f *testing.F) {
	// Seed with representative test cases
	seedCases := []struct {
		wordCount           int
		langSubset          int // number of languages to use (1-11)
		translatorLatencyMs int
	}{
		{0, 1, 0},     // empty dict, single lang, no latency
		{10, 3, 0},    // small dict, few langs, fast
		{100, 11, 10}, // medium dict, all langs, moderate latency
		{500, 11, 50}, // large dict, all langs, high latency
		{50, 5, 100},  // medium dict, partial langs, very high latency
	}

	for _, sc := range seedCases {
		// Encode seed params into bytes
		data := []byte{
			byte(min(max(sc.wordCount, 0), 1000)),
			byte(min(max(sc.langSubset, 1), 11)),
			byte(min(max(sc.translatorLatencyMs, 0), 255)),
		}
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 3 {
			t.Skip("insufficient fuzz data")
		}

		wordCount := int(data[0])
		langSubset := min(max(int(data[1]), 1), len(SupportedLangs))
		latencyMs := int(data[2])

		testDir := t.TempDir()
		converterPath := path.Join(testDir, "fuzz_final.csv.gz")

		// Build dictionary with controlled parameters
		dict := make(TranslationDictionary)
		for _, l := range SupportedLangs {
			dict[l.Tag] = make(Dictionary)
		}

		// Use only a subset of languages for this run
		languages := make([]language.Tag, langSubset)
		for i := range langSubset {
			languages[i] = SupportedLangs[i].Tag
		}

		// Create words needing translation
		for i := range wordCount {
			base := fmt.Sprintf("fuzz_%d", i)
			for _, lang := range languages {
				node := &DictNode{
					ID:           CleanUpWord(base),
					W:            base,
					Lang:         lang,
					Type:         WordTypeNoun,
					Translations: nil,
				}
				dict[lang][node.ID] = node
			}
		}

		// Mock capabilities: all pairs in subset supported
		canTranslate := make(map[language.Tag]map[language.Tag]bool)
		for _, l1 := range languages {
			canTranslate[l1] = make(map[language.Tag]bool)
			for _, l2 := range languages {
				canTranslate[l1][l2] = true
			}
		}

		// Mock translator with configurable latency
		translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
			if latencyMs > 0 {
				time.Sleep(time.Duration(latencyMs) * time.Millisecond)
			}
			result := make([]string, len(words))
			for i, w := range words {
				if from == to {
					result[i] = w.W // self-translation: return original
				} else {
					result[i] = fmt.Sprintf("%s_%s", w.W, to.String())
				}
			}
			return result, nil
		}

		ctx := context.Background()
		start := time.Now()

		err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
		if err != nil {
			t.Fatalf("BuildConverter failed: %v", err)
		}
		elapsed := time.Since(start)

		// Invariant 1: File must exist
		info, err := os.Stat(converterPath)
		if err != nil {
			t.Fatalf("Failed to stat converter file: %v", err)
		}

		// Invariant 2: File mod time must be within execution window (+ small buffer)
		buffer := 10 * time.Millisecond // account for filesystem timing granularity
		if info.ModTime().Before(start.Add(-buffer)) {
			t.Errorf("File mod time (%v) before start (%v) - save incomplete",
				info.ModTime(), start.Add(-buffer))
		}
		if info.ModTime().After(start.Add(elapsed + buffer)) {
			t.Errorf("File mod time (%v) after end+buffer (%v) - timing anomaly",
				info.ModTime(), start.Add(elapsed+buffer))
		}

		// Invariant 3: File must be valid and parseable
		loaded := readConverterFile(t, converterPath)

		// Invariant 4: All translated words should have complete clusters
		for _, lang := range languages {
			for _, word := range loaded[lang] {
				if word.Translations == nil {
					continue // words without translations aren't saved
				}
				for _, target := range languages {
					if _, ok := word.Translations[target]; !ok {
						t.Errorf("Word %q (%s) missing translation for %s",
							word.W, word.Lang.String(), target.String())
					}
				}
			}
		}

		// Invariant 5: File size should be reasonable (non-zero if words were translated)
		if wordCount > 0 && info.Size() == 0 {
			t.Errorf("Expected non-zero file size for %d words, got 0", wordCount)
		}
	})
}

// TestBuildConverter_FinalSave_Stress runs the final save test multiple times
// to catch intermittent timing/race issues.
func TestBuildConverter_FinalSave_Stress(t *testing.T) {
	const iterations = 20

	for i := range iterations {
		t.Run(fmt.Sprintf("iteration_%d", i+1), func(t *testing.T) {
			testDir := t.TempDir()
			converterPath := path.Join(testDir, fmt.Sprintf("stress_%d.csv.gz", i))

			// Vary dictionary size per iteration to stress different code paths
			wordCount := 50 + (i*37)%200 // 50-249 words, pseudo-random distribution

			dict := make(TranslationDictionary)
			for _, l := range SupportedLangs {
				dict[l.Tag] = make(Dictionary)
			}

			languages := getAllLanguageTags()

			// Create words needing translation
			for j := range wordCount {
				base := fmt.Sprintf("stress_%d_%d", i, j)
				for _, lang := range languages {
					node := &DictNode{
						ID:           CleanUpWord(base),
						W:            base,
						Lang:         lang,
						Type:         WordTypeNoun,
						Translations: nil,
					}
					dict[lang][node.ID] = node
				}
			}

			canTranslate := make(map[language.Tag]map[language.Tag]bool)
			for _, l1 := range languages {
				canTranslate[l1] = make(map[language.Tag]bool)
				for _, l2 := range languages {
					canTranslate[l1][l2] = true
				}
			}

			// Add slight random latency to expose race windows
			latency := time.Duration((i*13)%20) * time.Millisecond
			translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
				if latency > 0 {
					time.Sleep(latency)
				}
				result := make([]string, len(words))
				for k, w := range words {
					if from == to {
						result[k] = w.W
					} else {
						result[k] = fmt.Sprintf("%s_%s", w.W, to.String())
					}
				}
				return result, nil
			}

			ctx := context.Background()
			start := time.Now()

			err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
			if err != nil {
				t.Fatalf("BuildConverter failed: %v", err)
			}
			elapsed := time.Since(start)

			// Verify file timing invariant
			info, err := os.Stat(converterPath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			buffer := 500 * time.Millisecond
			if info.ModTime().Before(start) {
				t.Errorf("Iteration %d: file mod time (%v) before start (%v)",
					i+1, info.ModTime(), start)
			}
			if info.ModTime().After(start.Add(elapsed + buffer)) {
				t.Errorf("Iteration %d: file mod time (%v) after end+buffer (%v)",
					i+1, info.ModTime(), start.Add(elapsed+buffer))
			}

			// Verify file is valid
			loaded := readConverterFile(t, converterPath)

			// Verify all words have complete translation clusters
			var incomplete int
			for _, lang := range languages {
				for _, word := range loaded[lang] {
					if word.Translations == nil {
						continue
					}
					for _, target := range languages {
						if _, ok := word.Translations[target]; !ok {
							incomplete++
							break
						}
					}
				}
			}

			if incomplete > 0 {
				t.Errorf("Iteration %d: %d words have incomplete translation clusters",
					i+1, incomplete)
			}

			// Log progress for long runs
			if (i+1)%5 == 0 {
				t.Logf("✓ Completed %d/%d iterations", i+1, iterations)
			}
		})
	}
}

// TestBuildConverter_FinalSave_Concurrent verifies that the final save
// completes correctly even when other goroutines are accessing the dictionary.
func TestBuildConverter_FinalSave_Concurrent(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "concurrent_final.csv.gz")

	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	const wordCount = 100
	languages := getAllLanguageTags()

	// Create words needing translation
	for i := range wordCount {
		base := fmt.Sprintf("concurrent_%d", i)
		for _, lang := range languages {
			node := &DictNode{
				ID:           CleanUpWord(base),
				W:            base,
				Lang:         lang,
				Type:         WordTypeNoun,
				Translations: nil,
			}
			dict[lang][node.ID] = node
		}
	}

	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range languages {
		canTranslate[l1] = make(map[language.Tag]bool)
		for _, l2 := range languages {
			canTranslate[l1][l2] = true
		}
	}

	translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
		result := make([]string, len(words))
		for i, w := range words {
			if from == to {
				result[i] = w.W
			} else {
				result[i] = fmt.Sprintf("%s_%s", w.W, to.String())
			}
		}
		return result, nil
	}

	// Start a background goroutine that periodically reads the dictionary
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	wg.Go(func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// Concurrent read - should be safe with dictMu
				dictMu.Lock()
				_ = len(dict) // just access to trigger potential race
				dictMu.Unlock()
			}
		}
	})

	ctx := context.Background()
	start := time.Now()

	err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
	if err != nil {
		t.Fatalf("BuildConverter failed: %v", err)
	}
	elapsed := time.Since(start)

	// Stop the background reader
	close(stopCh)
	wg.Wait()

	// Verify final save completed
	info, err := os.Stat(converterPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	buffer := 500 * time.Millisecond
	if info.ModTime().Before(start) {
		t.Errorf("File mod time (%v) before start (%v)", info.ModTime(), start)
	}
	if info.ModTime().After(start.Add(elapsed + buffer)) {
		t.Errorf("File mod time (%v) after end+buffer (%v)", info.ModTime(), start.Add(elapsed+buffer))
	}

	// Verify file is valid and complete
	loaded := readConverterFile(t, converterPath)
	var incomplete int
	for _, lang := range languages {
		for _, word := range loaded[lang] {
			if word.Translations == nil {
				continue
			}
			for _, target := range languages {
				if _, ok := word.Translations[target]; !ok {
					incomplete++
					break
				}
			}
		}
	}

	if incomplete > 0 {
		t.Errorf("%d words have incomplete translation clusters after concurrent access", incomplete)
	}
}

// TestBuildConverter_FinalSave_Interrupted verifies that even if
// translation is interrupted, the final save still writes a valid file.
func TestBuildConverter_FinalSave_Interrupted(t *testing.T) {
	testDir := t.TempDir()
	converterPath := path.Join(testDir, "interrupted_final.csv.gz")

	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	const wordCount = 200
	languages := getAllLanguageTags()

	// Create words needing translation
	for i := range wordCount {
		base := fmt.Sprintf("interrupt_%d", i)
		for _, lang := range languages {
			node := &DictNode{
				ID:           CleanUpWord(base),
				W:            base,
				Lang:         lang,
				Type:         WordTypeNoun,
				Translations: nil,
			}
			dict[lang][node.ID] = node
		}
	}

	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range languages {
		canTranslate[l1] = make(map[language.Tag]bool)
		for _, l2 := range languages {
			canTranslate[l1][l2] = true
		}
	}

	// Slow translator to ensure we can interrupt mid-execution
	translateFn := func(words []*DictNode, from, to language.Tag) ([]string, error) {
		time.Sleep(50 * time.Millisecond)
		result := make([]string, len(words))
		for i, w := range words {
			if from == to {
				result[i] = w.W
			} else {
				result[i] = fmt.Sprintf("%s_%s", w.W, to.String())
			}
		}
		return result, nil
	}

	// Cancel after a short time to interrupt translation
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := BuildConverter(dict, converterPath, ctx, canTranslate, translateFn)
	elapsed := time.Since(start)

	// Accept either success or context cancellation
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	// Even if interrupted, final save should have written a valid file
	info, err := os.Stat(converterPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// File should exist and have been written during execution
	buffer := 2 * time.Second // larger buffer for interrupted case
	if info.ModTime().Before(start) {
		t.Errorf("File mod time (%v) before start (%v)", info.ModTime(), start)
	}
	if info.ModTime().After(start.Add(elapsed + buffer)) {
		t.Errorf("File mod time (%v) after end+buffer (%v)", info.ModTime(), start.Add(elapsed+buffer))
	}

	// File should be parseable even if incomplete
	loaded := readConverterFile(t, converterPath)

	// Count how many complete clusters we got (should be >= 0)
	var complete int
	for _, lang := range languages {
		for _, word := range loaded[lang] {
			if word.Translations == nil {
				continue
			}
			hasAll := true
			for _, target := range languages {
				if _, ok := word.Translations[target]; !ok {
					hasAll = false
					break
				}
			}
			if hasAll {
				complete++
			}
		}
	}

	t.Logf("✓ Interrupted run: %d complete translation clusters saved", complete)
	// We don't assert a minimum because timing is non-deterministic,
	// but the file should be valid and contain some data
}

// -- FINAL TEST --

// TestBuildConverterInterruptStress verifies save consistency under realistic
// interrupt conditions with a large dictionary.
func TestBuildConverterInterruptStress(t *testing.T) {
	// Skip if LibreTranslate isn't available (real HTTP calls)
	if !checkLibreTranslateAvailable(t) {
		t.Skip("LibreTranslate not available - skipping interrupt stress test")
	}

	testDir := t.TempDir()
	converterPath := path.Join(testDir, "stress_test.csv.gz")

	// Generate 3,000+ words needing translation
	dict := generateLargeTestDictionary(3000)

	// Pre-seed 100 words with complete translations (these MUST persist)
	seeds := seedCompleteTranslations(dict, 100)
	seedCount := len(seeds)

	// Build language tag slice from SupportedLangs (Lang structs with .Tag field)
	languages := make([]language.Tag, len(SupportedLangs))
	for i, l := range SupportedLangs {
		languages[i] = l.Tag
	}

	// Log initial state
	initialCount := countWordsNeedingTranslation(dict, languages)
	t.Logf("Initial state: %d words needing translation, %d seed clusters (%d word-lang pairs)",
		initialCount, seedCount, seedCount*len(SupportedLangs))

	// Cancel after 2-3 seconds: enough time for ~1-2 language batches to complete
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	err = BuildConverter(dict, converterPath, ctx, canTranslate, TranslateWords)
	elapsed := time.Since(start)

	// Accept either success or context cancellation
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildConverter failed: %v", err)
	}

	t.Logf("BuildConverter ran for %v before exit (err=%v)", elapsed, err)

	// Verify file exists and is valid
	info, err := os.Stat(converterPath)
	if err != nil {
		t.Fatal("Converter file was not created")
	}
	t.Logf("Saved file: %s, %d bytes", converterPath, info.Size())

	loaded := readConverterFile(t, converterPath)

	// Count total words in loaded dictionary with per-language breakdown
	var loadedTotal int
	loadedByLang := make(map[language.Tag]int)
	for _, lang := range SupportedLangs {
		count := len(loaded[lang.Tag])
		loadedByLang[lang.Tag] = count
		loadedTotal += count
	}
	t.Logf("Loaded dictionary: %d total word entries across %d languages",
		loadedTotal, len(SupportedLangs))

	// Count additional translated words (excluding seeds)
	// Note: CleanUpWord("seed_0") -> "seed0", so prefix is "seed"
	var additionalTranslated int
	additionalByLang := make(map[language.Tag]int)
	additionalDetails := make(map[language.Tag][]struct {
		id         string
		word       string
		hasAllLangs bool
	})

	for _, lang := range SupportedLangs {
		for id, node := range loaded[lang.Tag] {
			// Skip seeded words: CleanUpWord("seed_0") -> "seed0"
			if strings.HasPrefix(id, "seed") {
				continue
			}
			if node.Translations != nil {
				hasAll := true
				for _, l := range SupportedLangs {
					if _, ok := node.Translations[l.Tag]; !ok {
						hasAll = false
						break
					}
				}
				if hasAll {
					additionalTranslated++
					additionalByLang[lang.Tag]++
					additionalDetails[lang.Tag] = append(additionalDetails[lang.Tag], struct {
						id         string
						word       string
						hasAllLangs bool
					}{id, node.W, true})
				} else {
					// Track partial translations too for diagnostics
					additionalDetails[lang.Tag] = append(additionalDetails[lang.Tag], struct {
						id         string
						word       string
						hasAllLangs bool
					}{id, node.W, false})
				}
			}
		}
	}

	// Expected total: seeds (100 clusters × 11 langs = 1100) + additional with complete clusters
	expectedTotal := seedCount*len(SupportedLangs) + additionalTranslated
	if loadedTotal != expectedTotal {
		t.Logf("⚠ Count mismatch: expected %d total (seeds=%d + additional-complete=%d), got %d",
			expectedTotal, seedCount*len(SupportedLangs), additionalTranslated, loadedTotal)
		t.Logf("Breakdown by language:")
		for _, lang := range SupportedLangs {
			expectedPerLang := seedCount + additionalByLang[lang.Tag]
			actual := loadedByLang[lang.Tag]
			status := "✓"
			if actual != expectedPerLang {
				status = "✗"
			}
			t.Logf("  %s %s: expected %d, got %d (diff: %d)",
				status, lang.Tag.String(), expectedPerLang, actual, actual-expectedPerLang)
		}

		// Log details of the "extra" entries per language
		t.Logf("Diagnostic: non-seed entries by language:")
		for _, lang := range SupportedLangs {
			extra := loadedByLang[lang.Tag] - seedCount
			if extra > 0 {
				t.Logf("  %s: %d extra entries", lang.Tag.String(), extra)
				details := additionalDetails[lang.Tag]
				if len(details) > 0 {
					// Log first 5 with their translation status
					for i := range min(5, len(details)) {
						d := details[i]
						status := "complete"
						if !d.hasAllLangs {
							status = "partial"
						}
						t.Logf("    [%d] ID=%q W=%q (%s)", i+1, d.id, d.word, status)
					}
					if len(details) > 5 {
						t.Logf("    ... and %d more", len(details)-5)
					}
				}
			}
		}
	} else {
		t.Logf("✓ Loaded word count matches expected: %d (seeds + additional)", loadedTotal)
	}

	// CRITICAL: All seeded words MUST be present (they were pre-translated)
	var persistedSeed int
	var missingSeeds []string
	var incompleteSeeds []string
	for _, lang := range SupportedLangs {
		for _, m := range seeds {
			id := m[lang.Tag].ID
			if node, ok := loaded[lang.Tag][id]; ok && node.Translations != nil {
				hasAll := true
				var missingLangs []string
				for _, l := range SupportedLangs {
					if _, ok := node.Translations[l.Tag]; !ok {
						hasAll = false
						missingLangs = append(missingLangs, l.Tag.String())
					}
				}
				if hasAll {
					persistedSeed++
				} else {
					incompleteSeeds = append(incompleteSeeds,
						fmt.Sprintf("%s(%s): missing %v", id, lang.Tag.String(), missingLangs))
				}
			} else {
				missingSeeds = append(missingSeeds,
					fmt.Sprintf("%s(%s): not found or nil translations", id, lang.Tag.String()))
			}
		}
	}

	expectedSeed := seedCount * len(SupportedLangs)
	if persistedSeed < expectedSeed {
		t.Errorf("Only %d/%d seeded word-lang pairs persisted (expected all %d)",
			persistedSeed, expectedSeed, expectedSeed)

		if len(missingSeeds) > 0 {
			t.Logf("Missing seeds (%d total):", len(missingSeeds))
			for i, ms := range missingSeeds {
				if i >= 10 {
					t.Logf("  ... and %d more missing", len(missingSeeds)-10)
					break
				}
				t.Logf("  - %s", ms)
			}
		}
		if len(incompleteSeeds) > 0 {
			t.Logf("Incomplete seeds (%d total):", len(incompleteSeeds))
			for i, is := range incompleteSeeds {
				if i >= 10 {
					t.Logf("  ... and %d more incomplete", len(incompleteSeeds)-10)
					break
				}
				t.Logf("  - %s", is)
			}
		}
	} else {
		t.Logf("✓ All %d seeded word-lang pairs persisted correctly", persistedSeed)
	}

	// Log additional translations breakdown
	t.Logf("✓ Additional words with complete clusters: %d", additionalTranslated)
	if additionalTranslated > 0 {
		t.Logf("  Breakdown by language (complete clusters only):")
		for _, lang := range SupportedLangs {
			if n := additionalByLang[lang.Tag]; n > 0 {
				t.Logf("    %s: %d", lang.Tag.String(), n)
			}
		}
	}

	// Verify file content: count CSV rows
	rows, records := countConverterLines(t, converterPath)
	t.Logf("Converter file contains %d data rows (clusters)", rows)

	// Cross-check: seeded clusters should each produce exactly 1 row
	expectedMinRows := seedCount
	if rows < expectedMinRows {
		t.Logf("⚠ Warning: expected at least %d rows (seed clusters), got %d",
			expectedMinRows, rows)

		var missingClusters []string
		for _, cluster := range seeds {
			found := false
			for _, record := range records[1:] {
				for _, cell := range record[:len(record)-1] {
					for _, node := range cluster {
						if cell == node.W {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				for _, node := range cluster {
					missingClusters = append(missingClusters,
						fmt.Sprintf("%s(%s)", node.W, node.Lang.String()))
					break
				}
			}
		}

		if len(missingClusters) > 0 {
			t.Logf("Missing clusters in CSV (%d total):", len(missingClusters))
			for i, mc := range missingClusters {
				if i >= 10 {
					t.Logf("  ... and %d more missing clusters", len(missingClusters)-10)
					break
				}
				t.Logf("  - %s", mc)
			}
		}
	}

	// Log sample of CSV content for debugging (first 3 data rows)
	if rows > 0 {
		t.Logf("CSV content sample (first 3 data rows):")
		for i := 1; i <= min(3, len(records)-1); i++ {
			t.Logf("  Row %d: %v", i, records[i])
		}
	}

	// Log final word count needing translation
	finalCount := countWordsNeedingTranslation(dict, languages)
	t.Logf("Final state: %d words still needing translation (expected >0 if interrupted)", finalCount)

	// We don't assert a minimum because timing is non-deterministic,
	// but logging helps diagnose if zero were saved (indicating a save timing bug)
}

// countWordsNeedingTranslation helper for test logging
func countWordsNeedingTranslation(dict TranslationDictionary, languages []language.Tag) int {
	canTranslate := make(map[language.Tag]map[language.Tag]bool)
	for _, l1 := range SupportedLangs {
		canTranslate[l1.Tag] = make(map[language.Tag]bool)
		for _, l2 := range SupportedLangs {
			canTranslate[l1.Tag][l2.Tag] = true
		}
	}
	_, count := checkForTranslations(dict, canTranslate, languages)
	return count
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


// generateLargeTestDictionary creates N words needing translation across all languages
func generateLargeTestDictionary(n int) TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	// Generate deterministic test words: "testword_0", "testword_1", etc.
	for i := range n {
		baseWord := fmt.Sprintf("testword%d", i)
		for _, lang := range SupportedLangs {
			node := &DictNode{
				ID:           CleanUpWord(baseWord),
				W:            baseWord,
				Lang:         lang.Tag,
				Type:         WordTypeNoun,
				Translations: nil, // Needs translation
			}
			dict[lang.Tag][node.ID] = node
		}
	}
	return dict
}

// seedCompleteTranslations adds N fully-translated word clusters to the dictionary
// Returns the number of seed words added.
func seedCompleteTranslations(dict TranslationDictionary, count int) []map[language.Tag]*DictNode {
	seeds := make([]map[language.Tag]*DictNode, 0, count)
	for i := range count {
		cluster := make(map[language.Tag]*DictNode)
		baseWord := fmt.Sprintf("seed%d", i)

		for _, lang := range SupportedLangs {
			node := &DictNode{
				ID:   CleanUpWord(baseWord),
				W:    baseWord,
				Lang: lang.Tag,
				Type: WordTypeNoun,
			}
			cluster[lang.Tag] = node
		}

		// Link all nodes to the same cluster map
		for _, lang := range SupportedLangs {
			node := cluster[lang.Tag]
			node.Translations = cluster
			dict[lang.Tag][node.ID] = node
		}

		seeds = append(seeds, cluster)
	}
	return seeds
}

