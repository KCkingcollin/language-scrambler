package main

import (
	. "langscram-lib" //nolint
	"path/filepath"
	"testing"

	"golang.org/x/text/language"
)

var (
	equivalentWords = GenWords(false)
	randomWords = GenWords(true)
)

func GenWords(rand bool) map[language.Tag][]struct {
	Noun *DictNode
	Verb *DictNode
} {
	out := make(map[language.Tag][]struct {Noun *DictNode; Verb *DictNode}, len(SupportedLangs)) 
	switch rand {
	case true:
		for _, l := range SupportedLangs {
			wordList, err := LoadList(l.Tag)
			if err != nil {
				panic(err)
			}

			for range 5 {
				for _, w := range wordList {
					if w.Type == WordTypeNoun {
						d := struct{Noun *DictNode; Verb *DictNode}{Noun: w}
						out[l.Tag] = append(out[l.Tag], d)
						break
					}
				}
			}
			
			var i int
			for range 5 {
				for _, w := range wordList {
					if w.Type == WordTypeVerb {
						out[l.Tag][i].Verb = w
						i++
						break
					}
				}
			}
		}
	}
	return out
}

func TestTranslationClusterStability(t *testing.T) {
	tmp := t.TempDir()

	DictionaryPath = tmp
	LoadedDictionary = make(TranslationDictionary)
	for _, l := range SupportedLangs {
		LoadedDictionary[l.Tag] = make(Dictionary)
	}

	for lang := range equivalentWords {
		list, err := LoadList(lang)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range list {
			LoadedDictionary[lang][k] = v
		}
	}

	err := BuildConverter(LoadedDictionary, filepath.Join(tmp, "conv.csv.gz"))
	if err != nil {
		t.Fatal(err)
	}

	clusters := make(map[*DictNode]bool)
	for _, dict := range LoadedDictionary {
		for _, w := range dict {
			if w.Translations == nil {
				continue
			}
			for _, n := range w.Translations {
				clusters[n] = true
			}
		}
	}

	if len(clusters) != 10 { // 5 nouns + 5 verbs
		t.Fatalf("expected 10 clusters, got %d", len(clusters))
	}
}
