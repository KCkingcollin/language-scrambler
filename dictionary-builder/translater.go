package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	. "langscram-lib" //nolint

	"golang.org/x/text/language"
)

const (
	langUrlBasePart1 = "https://raw.githubusercontent.com/open-dict-data/wikidict-"
	langUrlBasePart2 = "/refs/heads/master/data/"
	langUrlBasePart3 = "-"
	langUrlBasePart4 = "_wiki.txt"
)

type t struct {
	Dictionary map[string]map[language.Base]string
	Lang language.Base
}

var Translater t

func getLangUrl(fromLang, toLang language.Base) string {
	return langUrlBasePart1+toLang.String()+langUrlBasePart2+fromLang.String()+langUrlBasePart3+toLang.String()+langUrlBasePart4
}

func getLangDictionary(fromLang, toLang language.Base) ([][]string, error) {
    response, err := http.Get(getLangUrl(fromLang, toLang))
    if err != nil {
        return nil, err
    }
    defer response.Body.Close() //nolint

    body, err := io.ReadAll(response.Body)
    if err != nil {
        return nil, err
    }
	var output [][]string
	for s := range strings.SplitSeq(strings.TrimSpace(string(body)), "\n") {
		output = append(output, strings.Split(s, "\t"))
	}
	return output, nil
}

func BuildTranslatorDictionary(fromLang language.Base) (map[string]map[language.Base]string, error) {
	totalLoops := float32(len(SupportedLangs))
	var loopCounter float32
	translationDictionary := make(map[string]map[language.Base]string, int(1e6))
	for _, toLang := range SupportedLangs {
		loopCounter++
		if fromLang == toLang { continue }
		dictionary, err := getLangDictionary(fromLang, toLang)
		if err != nil {
			return nil, err
		}
		for i, l := range dictionary {
			if len(l) != 2 {
				return nil, fmt.Errorf("the dictionary line %d containing \"%s\" was broken, had a length of %d", i, strings.Join(l, "\t"), len(l))
			}
			word := CleanUpWord(l[0])
			translation := CleanUpWord(l[1])
			if translationDictionary[word] == nil {
				translationDictionary[word] = make(map[language.Base]string, len(SupportedLangs)-1)
			}
			translationDictionary[word][toLang] = translation
		}

		dictionary, err = getLangDictionary(toLang, fromLang)
		if err != nil {
			return nil, err
		}
		for i, l := range dictionary {
			if len(l) != 2 {
				return nil, fmt.Errorf("the dictionary line %d containing \"%s\" was broken, had a length of %d", i, strings.Join(l, "\t"), len(l))
			}
			word := CleanUpWord(l[1])
			translation := CleanUpWord(l[0])
			if translationDictionary[word] == nil {
				translationDictionary[word] = make(map[language.Base]string, len(SupportedLangs)-1)
			}
			translationDictionary[word][toLang] = translation
		}

		fmt.Printf("\033[2K\rProgress %d%%", int(loopCounter/totalLoops*100))
	} 
	fmt.Println()
	return translationDictionary, nil
}

func TranslateWords(words []string, fromLang, toLang language.Base) ([]string, error) {
	if Translater.Lang != fromLang {
		var err error
		Translater.Dictionary, err = BuildTranslatorDictionary(fromLang); 
		if err != nil {
			return nil, err
		}
		Translater.Lang = fromLang
	}
	var translations []string
	for _, w := range words {
		translations = append(translations, Translater.Dictionary[CleanUpWord(w)][toLang])
	}
	return translations, nil
}
