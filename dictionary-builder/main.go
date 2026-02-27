package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	. "langscram-lib" //nolint
	"log"
	"os"
	"path"
	"time"

	"golang.org/x/text/language"
)

func BuildConverter(fromList language.Tag) ([]byte, error) {
	wordsMap, err := LoadList(fromList)
	if err != nil {
		return nil, err
	}

	words := make([]Word, 0, len(wordsMap))
	for _, w := range wordsMap {
		words = append(words, w)
	}

	header := []string{fromList.String()}
	targetLangs := make([]language.Tag, 0, len(SupportedLangs)-1)

	for _, toLang := range SupportedLangs {
		if toLang.Tag == fromList {
			continue
		}
		header = append(header, toLang.Tag.String())
		targetLangs = append(targetLangs, toLang.Tag)
	}

	fmt.Println("translating", len(words), "words to", len(targetLangs), "languages")

	allTranslations := make([]map[language.Tag]string, len(words))

	for _, toLang := range targetLangs {
		timer := time.Now()

		translations, err := TranslateWords(words, fromList, toLang)
		if err != nil {
			return nil, err
		}

		if len(translations) != len(words) {
			return nil, fmt.Errorf(
				"translation length mismatch: got %d, expected %d",
				len(translations),
				len(words),
			)
		}

		for i := range words {
			if allTranslations[i] == nil {
				allTranslations[i] = make(map[language.Tag]string, len(targetLangs))
			}
			allTranslations[i][toLang] = translations[i]
		}

		log.Println(
			"translated to",
			toLang.String()+".",
			fmt.Sprintf("took %s.", time.Since(timer)),
		)
	}

	converter := make([][]string, 0, len(words)+1)
	converter = append(converter, header)

	var missingWords int

	for i, word := range words {
		row := make([]string, 0, len(header))
		row = append(row, word.W)

		for _, lang := range targetLangs {
			translation := allTranslations[i][lang]
			if translation == "" {
				missingWords++
			}
			row = append(row, translation)
		}

		converter = append(converter, row)
	}

	fmt.Println("we're missing", missingWords, "words")

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.WriteAll(converter); err != nil {
		return nil, err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func main() {
	for _, lang := range SupportedLangs {
		converterPath := path.Join(DictionaryPath, lang.Tag.String()+"-"+ConverterDictionaryName+".csv")
		_, err := os.ReadFile(converterPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Println(err)
				continue
			}

			data, err := BuildConverter(lang.Tag)
			if err != nil {
				log.Fatalln("making conversion list:", err)
			}

			file, err := os.Create(converterPath)
			if err != nil {
				log.Fatalln("making file:", err)
			}
			defer file.Close() //nolint

			if _, err := file.Write(data); err != nil {
				log.Fatalln("writing conversion list: ", err)
			}
		}
	}
}

