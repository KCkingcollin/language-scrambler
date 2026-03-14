package main

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	. "langscram-lib" //nolint
	"log"
	"os"
	"os/signal"
	"path"
	"reflect"
	"slices"
	"sync"
	"syscall"
	"time"

	"golang.org/x/text/language"
)

var dictMu sync.Mutex

func SaveConverter(tDict TranslationDictionary, converterPath string, languages []language.Tag) error {
	seenClusters := make(map[uintptr]bool)
	clusters := make([]map[language.Tag]*DictNode, 0)
	dictMu.Lock()

	for _, dict := range tDict {
		for _, w := range dict {
			if w.Translations == nil {
				continue
			}

			clusterPtr := reflect.ValueOf(w.Translations).Pointer()
			if seenClusters[clusterPtr] {
				continue
			}
			seenClusters[clusterPtr] = true

			clusters = append(clusters, w.Translations)
		}
	}

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

	dictMu.Unlock()

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

func checkForTranslations(tDict TranslationDictionary, canTranslate map[language.Tag]map[language.Tag]bool, languages []language.Tag) (map[language.Tag][]*DictNode, int) {
	var amountOfWords int
	wordsMap := make(map[language.Tag][]*DictNode, len(languages))
	dictMu.Lock()
	for _, d := range tDict {
		for _, w := range d {
			needsTranslation := true
			if w.Translations != nil {
				needsTranslation = false
				var langs []language.Tag
				for _, l := range languages {
					if canTranslate[w.Lang][l] {
						langs = append(langs, l)
					}
				}
				for _, l := range langs {
					if _, ok := w.Translations[l]; !ok {
						needsTranslation = true
						break
					}
				}
			}
			if needsTranslation {
				wordsMap[w.Lang] = append(wordsMap[w.Lang], w)
				amountOfWords++
			}
		}
	}
	dictMu.Unlock()
	return wordsMap, amountOfWords
}

func BuildConverter(
	tDict TranslationDictionary, 
	converterPath string, 
	ctx context.Context,
	canTranslate map[language.Tag]map[language.Tag]bool,
	translateFn func([]*DictNode, language.Tag, language.Tag) ([]string, error),
) error {
	var languages = make([]language.Tag, 0, len(SupportedLangs))
	for _, toLang := range SupportedLangs {
		languages = append(languages, toLang.Tag)
	}
	slices.SortFunc(languages, func(a, b language.Tag) int {
		return len(tDict[b]) - len(tDict[a])
	})

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

	wordsMap, amountOfWords := checkForTranslations(tDict, canTranslate, languages)

	fmt.Println("translating roughly", amountOfWords, "words to", len(SupportedLangs), "languages")

	var translationInterrupted bool
	const bulkSize = 1000

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
				if ctx.Err() != nil {
					translationInterrupted = true
					break
				}

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

				translations, err := translateFn(toTranslate, fromLang, toLang)
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
			if translationInterrupted {
				break
			}

			fmt.Println()
			log.Println(
				"translated to",
				toLang.String()+".",
				fmt.Sprintf("took %s.", time.Since(timer)),
			)
		}

		if translationInterrupted {
			fmt.Println()
			break
		}
	}

	log.Println("Running final save...")
	saveQueue <- struct{}{}
	close(saveQueue)
	wg.Wait()

	fmt.Println("Checking for missing translations...")
	_, amountOfWords = checkForTranslations(tDict, canTranslate, languages)

	if amountOfWords != 0 {
		log.Printf("%d words did not get translated", amountOfWords)
		if !translationInterrupted {
			return BuildConverter(tDict, converterPath, ctx, canTranslate, translateFn)
		}
	}

	return nil
}

func main() {
	fmt.Println("Loading dictionaries...")
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

	converterWords := 0
	for _, d := range LoadedDictionary {
		converterWords += len(d)
	}
	log.Printf("After ReadConverter: %d words", converterWords)

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

	totalWords := 0
	for _, d := range LoadedDictionary {
		totalWords += len(d)
	}
	log.Printf("After LoadList: %d words (%d added from lists)", totalWords, totalWords-converterWords)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\nInterrupt received, shutting down translation...")
		cancel()
	}()

	canTranslate, err := GetTranslationCapabilities()
	if err != nil {
		log.Fatal(err)
	}

	err = BuildConverter(LoadedDictionary, converterPath, ctx, canTranslate, TranslateWords)
	if err != nil {
		log.Fatalln("making conversion list:", err)
	}
}
