package main

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	. "langscram-lib" //nolint
	"log"
	"os"
	"path"
	"slices"
	"sync"
	"time"

	"golang.org/x/text/language"
)

var dictMu sync.RWMutex

func SaveConverter(tDict TranslationDictionary, converterPath string, languages []language.Tag) error {
	seen := make(map[*DictNode]bool)
	clusters := make([]map[language.Tag]*DictNode, 0)

	dictMu.RLock()
	for _, dict := range tDict {
		for _, w := range dict {
			if w.Translations == nil {
				continue
			}

			if seen[w] {
				continue
			}

			for _, node := range w.Translations {
				seen[node] = true
			}

			clusters = append(clusters, w.Translations)
		}
	}
	dictMu.RUnlock()

	if len(languages) < 1 {
		return fmt.Errorf("no languages provided")
	}

	header := make([]string, 0, len(languages)+1)
	for _, toLang := range languages {
		header = append(header, toLang.String())
	}
	header = append(header, "type")

	converter := make([][]string, 0, len(clusters)+1)
	converter = append(converter, header)

	dictMu.RLock()
	for _, cluster := range clusters {
		line := make([]string, len(header))

		var word *DictNode
		for _, node := range cluster {
			word = node
			break
		}
		if word == nil {
			continue
		}

		for i, l := range languages {
			if node, ok := cluster[l]; ok {
				line[i] = node.W
			}
		}
		line[len(line)-1] = word.GetType()

		converter = append(converter, line)
	}
	dictMu.RUnlock()

	file, err := os.Create(converterPath + ".tmp")
	if err != nil {
		return fmt.Errorf("making file: %w", err)
	}
	defer file.Close() //nolint

	gzFile := gzip.NewWriter(file)
	writer := csv.NewWriter(gzFile)

	if err := writer.WriteAll(converter); err != nil {
		return err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	if err := gzFile.Close(); err != nil {
		return err
	}

	err = os.Rename(converterPath+".tmp", converterPath)
	if err != nil {
		return err
	}

	return nil
}

func BuildConverter(tDict TranslationDictionary, converterPath string) error {
	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		return err
	}

	var languages = make([]language.Tag, 0, len(SupportedLangs))
	for _, toLang := range SupportedLangs {
		languages = append(languages, toLang.Tag)
	}
	slices.SortFunc(languages, func(a, b language.Tag) int {
		return len(tDict[b]) - len(tDict[a])
	})

	const bulkSize = 1000

	var wg sync.WaitGroup
	saveQueue := make(chan struct{}, 1)

	wg.Go(func() {
		for range saveQueue {
			err := SaveConverter(tDict, converterPath, languages)
			if err != nil {
				log.Println("Save failed:", err)
			}
		}
	})

	var amountOfWords int
	wordsMap := make(map[language.Tag][]*DictNode)
	for _, d := range tDict {
		for _, w := range d {
			if w.Translations == nil {
				wordsMap[w.Lang] = append(wordsMap[w.Lang], w)
				amountOfWords++
			}
		}
	}

	fmt.Println("translating roughly", amountOfWords, "words to", len(SupportedLangs), "languages")

	for _, toLang := range languages {
		for fromLang, words := range wordsMap {
			if len(words) == 0 {
				continue
			}

			if !canTranslate[fromLang][toLang] {
				log.Printf("%s to %s is not supported", fromLang.String(), toLang.String())
				continue
			}

			fmt.Println("translating", len(words), "words from", fromLang.String(), "to", toLang.String())

			timer := time.Now()
			for i := 0; i < len(words); i += bulkSize {
				buffer := words[i:min(i+bulkSize, len(words))]

				var toTranslate []*DictNode
				for _, w := range buffer {
					if w.Translations == nil {
						toTranslate = append(toTranslate, w)
						continue
					}
					if _, ok := w.Translations[toLang]; !ok {
						toTranslate = append(toTranslate, w)
					}
				}

				if len(toTranslate) == 0 {
					continue
				}

				translations, err := TranslateWords(toTranslate, fromLang, toLang)
				if err != nil {
					return err
				}

				if len(translations) != len(toTranslate) {
					return fmt.Errorf(
						"translation length mismatch: got %d, expected %d",
						len(translations),
						len(toTranslate),
					)
				}

				dictMu.Lock()
				for i, w := range toTranslate {
					translation := &DictNode{ID: CleanUpWord(translations[i]), W: translations[i], Lang: toLang, Type: w.Type}
					tDict.AddTranslation(w, translation)
				}
				dictMu.Unlock()

				select {
				case saveQueue <- struct{}{}:
				default:
				}
				progress := float32(min(i+bulkSize, len(words))) / float32(len(words)) * 100
				fmt.Printf("\r\033[K%.1f%%", progress)
			}
			fmt.Println()

			log.Println(
				"translated to",
				toLang.String()+".",
				fmt.Sprintf("took %s.", time.Since(timer)),
			)
		}
	}

	select {
	case saveQueue <- struct{}{}:
	default:
	}
	close(saveQueue)
	wg.Wait()

	return nil
}

func main() {
	converterPath := path.Join(DictionaryPath, ConverterDictionaryName+".csv.gz")
	file, err := os.Open(converterPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalln(err)
		}
	} else {
		defer file.Close() //nolint

		gzFile, err := gzip.NewReader(file)
		if err != nil {
			log.Fatalln(err)
		}
		defer gzFile.Close() //nolint

		data, err := io.ReadAll(gzFile)
		if err != nil {
			log.Fatalln(err)
		}

		LoadedDictionary, err = ReadConverter(data)
		if err != nil && !errors.Is(err, ErrEmptyConverter) {
			log.Fatalln(err)
		}

		for _, dict := range LoadedDictionary {
			for key, node := range dict {
				if CleanUpWord(node.W) != key {
					panic("identity drift detected")
				}
			}
		}
	}

	for _, lang := range SupportedLangs {
		list, err := LoadList(lang.Tag)
		if err != nil {
			log.Fatalln(err)
		}
		for s, w := range list {
			if _, ok := LoadedDictionary[w.Lang][s]; !ok {
				LoadedDictionary[w.Lang][s] = w
			}
		}
	}

	err = BuildConverter(LoadedDictionary, converterPath)
	if err != nil {
		log.Fatalln("making conversion list:", err)
	}
}
