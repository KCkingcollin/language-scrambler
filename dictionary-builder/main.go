package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	. "langscram-lib" //nolint
	"log"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/text/language"
)

func GetListNames() error {
	dir, err := os.ReadDir(DictionaryPath)
	if err != nil {
		return err
	}
	WordLists = make([]string, 0, len(dir))
	for _, file := range dir {
		name := CleanUpWord(file.Name())
		if strings.TrimRight(name, ".") != "" && strings.HasSuffix(name, ".list") {
			WordLists = append(WordLists, strings.TrimSuffix(name, ".list"))
		}
	}
	return nil
}

func BuildConverter(fromList language.Base, listPath, csvPath string) ([]byte, error) {
	listData, err := os.ReadFile(listPath)
	if err != nil {
		return nil, err
	}
	words := strings.Split(string(listData), "\n")

	converterMap := make(map[string]map[language.Base]string, len(words))

	fmt.Println("translating", len(words), "words to", len(SupportedLangs)-1, "languages")
	allTranslations := make([]map[language.Base]string, len(words))
	for _, toLang := range SupportedLangs {
		if toLang == fromList {
			continue
		}

		timer := time.Now()
		translations, err := TranslateWords(words, fromList, toLang)
		if err != nil {
			return nil, err
		}

		for i := range allTranslations {
			if allTranslations[i] == nil {
				allTranslations[i] = make(map[language.Base]string, len(SupportedLangs)-1)
			}
			allTranslations[i][toLang] = translations[i]
		}

		log.Println("translated to", toLang.String()+".", fmt.Sprintf("took %s.", time.Since(timer)))
	}

	for i, m := range allTranslations {
		converterMap[CleanUpWord(words[i])] = m
	}

	converter := make([][]string, len(words)+1)
	converter[0] = append(converter[0], fromList.String())
	for _, lang := range SupportedLangs {
		if lang == fromList {
			continue
		}
		converter[0] = append(converter[0], lang.String())
	}
	var missingWords int
	for i, word := range words {
		converter[i+1] = []string{word}
		for _, l := range converter[0][1:] {
			lang, _ := language.ParseBase(l)
			translation := converterMap[word][lang]
			if translation == "" {
				missingWords++
			}
			converter[i+1] = append(converter[i+1], translation)
		}
	}
	fmt.Println("we're missing", missingWords, "words")

	buffer := new(bytes.Buffer)

	file, err := os.Create(csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint
	
	if err := csv.NewWriter(buffer).WriteAll(converter); err != nil {
		return nil, err
	}

	data := make([]byte, buffer.Len())
	if _, err := buffer.Read(data); err != nil {
		return nil, err
	}

	if _, err := file.Write(data); err != nil {
		return nil, err
	}

	return data, nil
}


func main() {
	if err := GetListNames(); err != nil {
		log.Println(err)
	}
	fmt.Println(WordLists)
	for _, listName := range WordLists {
		converterPath := path.Join(DictionaryPath, listName+"-"+ConverterDictionaryName+".csv")
		_, err := os.ReadFile(converterPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Println(err)
				continue
			}
			base, _ := language.ParseBase(listName)
			_, err = BuildConverter(base, path.Join(DictionaryPath, listName+".list"), converterPath)
			if err != nil {
				log.Println("making conversion list: %w", err)
			}
		}
	}
}

