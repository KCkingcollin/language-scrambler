package main

import (
	"encoding/csv"
	"fmt"
	. "langscram-lib" //nolint
	"log"
	"maps"
	"os"
	"path"
	"time"

	"golang.org/x/text/language"
)

func SaveConverter(tDict TranslationDictionary, converterPath string, languages []language.Tag) error {
	header := make([]string, len(languages)+1)
	for _, toLang := range languages {
		header = append(header, toLang.String())
	}
	header = append(header, "type")

	if len(languages) < 1 {
		return fmt.Errorf("no languages provided")
	}

	converter := make([][]string, 0, len(tDict[languages[0]])+1)
	converter = append(converter, header)

	for _, w := range tDict[languages[0]] {
		line := make([]string, len(header))

		complete := true
		for _, l := range languages {
			if _, ok := w.Translations[l]; !ok {
				log.Println("skipping:", w.W, "due to insufficient information")
				complete = false
				break
			}
		}
		if !complete {
			continue
		}

		for i, l := range languages {
			word := w.Translations[l]
			line[i] = word.W
		}

		line[len(line)-1] = w.GetType()

		converter = append(converter, line)
	}

	file, err := os.Create(converterPath + ".tmp")
	if err != nil {
		return fmt.Errorf("making file: %w", err)
	}
	defer file.Close() //nolint

	writer := csv.NewWriter(file)

	if err := writer.WriteAll(converter); err != nil {
		return err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	err = os.Rename(converterPath+".tmp", converterPath)
	if err != nil {
		return err
	}

	return nil
}

func BuildConverter(tDict TranslationDictionary, converterPath string) error {
	const bulkSize = 1000

	var biggestLang language.Tag
	var maxListSize int
	for l, c := range tDict {
		if len(c) > maxListSize {
			biggestLang = l
			maxListSize = len(c)
		}
	}

	languages := []language.Tag{biggestLang}
	for _, toLang := range SupportedLangs {
		if toLang.Tag == biggestLang {
			continue
		}
		languages = append(languages, toLang.Tag)
	}

	wordsMap := make(map[language.Tag][]*DictNode)
	for _, d := range tDict {
		for _, w := range d {
			if w.Translations == nil {
				wordsMap[w.Lang] = append(wordsMap[w.Lang], w)
			}
		}
	}

	fmt.Println("translating at least", maxListSize, "words to", len(SupportedLangs), "languages")

	for _, toLang := range languages {
		for fromLang, words := range wordsMap {
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

				for i, w := range toTranslate {
					translation := &DictNode{W: translations[i], Lang: toLang, Type: w.Type}
					tDict.AddTranslation(w, translation)
				}

				// I know, I know, its slow, but its better than losing hours of translation rendering
				err = SaveConverter(tDict, converterPath, languages)
				if err != nil {
					return err
				}

			}

			log.Println(
				"translated to",
				toLang.String()+".",
				fmt.Sprintf("took %s.", time.Since(timer)),
			)
		}
	}

	return nil
}

func main() {
	for _, lang := range SupportedLangs {
		list, err := LoadList(lang.Tag)
		if err != nil {
			log.Fatalln(err)
		}
		maps.Copy(LoadedDictionary[lang.Tag], list)
	}

	converterPath := path.Join(DictionaryPath, ConverterDictionaryName+".csv")
	data, err := os.ReadFile(converterPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalln(err)
		}
	} else {
		LoadedDictionary, err = ReadConverter(data)
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = BuildConverter(LoadedDictionary, converterPath)
	if err != nil {
		log.Fatalln("making conversion list:", err)
	}
}
