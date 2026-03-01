package lib

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"unicode"

	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

const (
	DictionaryDirName       = "dictionaries"
	ConverterDictionaryName = "converter"
)

var (
	LoadedDictionary = func() TranslationDictionary {
		dict := make(TranslationDictionary)
		for _, l := range SupportedLangs {
			dict[l.Tag] = make(Dictionary)
		}
		return dict
	}()
	DictionaryPath = path.Clean(DictionaryDirName)
	WordLists      []string
	DictionaryTree *RuneNode
	MaxFrequency   int
)

// adds the translation to the translations map in the dictionaries node, as well as sets the translation map for the translation node as well, updates the word and translation nodes accordingly
//
// YOUR NODES ARE NOT SAFE HERE
func (dict TranslationDictionary) AddTranslation(word, translation *DictNode) {
	var translations map[language.Tag]*DictNode

	wordKey := CleanUpWord(word.W)
	translationKey := CleanUpWord(translation.W)

	wordNode, wordOk := dict[word.Lang][wordKey]
	translationNode, translationOk := dict[translation.Lang][translationKey]

	switch {
	case wordOk && wordNode.Translations != nil:
		translations = wordNode.Translations
	case translationOk && translationNode.Translations != nil:
		translations = translationNode.Translations
	default:
		translations = make(map[language.Tag]*DictNode)
	}

	if !translationOk {
		translationNode = &DictNode{Lang: translation.Lang, W: translation.W, Translations: translations, Type: translation.Type}
	} else {
		translationNode.Translations = translations
	}

	if !wordOk {
		wordNode = &DictNode{Lang: word.Lang, W: word.W, Translations: translations, Type: word.Type}
	} else {
		wordNode.Translations = translations
	}

	translations[wordNode.Lang] = wordNode
	translations[translationNode.Lang] = translationNode

	dict[word.Lang][wordKey] = wordNode
	dict[translation.Lang][translationKey] = translationNode

	*word = *wordNode
	*translation = *translationNode
}

func GetBase(lang language.Tag) language.Base {
	base, conf := lang.Base()
	if conf != language.High {
		log.Println("error finding base for", lang.String())
	}
	return base
}

// makes word all lowercase and removes any non-alphabet chars, and normalizes it
func CleanUpWord(w string) string {
	normalized := w
	if !norm.NFC.IsNormalString(normalized) {
		normalized = norm.NFC.String(normalized)
	}

	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, char := range normalized {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			builder.WriteRune(unicode.ToLower(char))
		}
	}

	return builder.String()
}

func LoadList(fromLang language.Tag) (Dictionary, error) {
	listPath := path.Join(DictionaryPath, fromLang.String()+".list")
	listData, err := os.ReadFile(listPath)
	if err != nil {
		return nil, err
	}

	listReader := csv.NewReader(bytes.NewReader(listData))
	list, err := listReader.ReadAll()
	if err != nil {
		return nil, err
	}

	words := make(Dictionary, len(list))
	for _, line := range list {
		if len(line) < 2 {
			return nil, fmt.Errorf("only one column? List may be corrupted")
		}
		word := &DictNode{W: line[0], Lang: fromLang, Type: ParseWordType(line[1])}
		words[CleanUpWord(line[0])] = word
	}

	return words, nil
}

func ReadConverter(data []byte) (TranslationDictionary, error) {
	converterReader := csv.NewReader(bytes.NewReader(data))
	converter, err := converterReader.ReadAll()
	if err != nil {
		return nil, err
	}

	var langIndexs []language.Tag
	for _, lang := range converter[0][:len(converter[0])-1] {
		langTag, err := language.Parse(lang)
		if err != nil {
			return nil, err
		}
		langIndexs = append(langIndexs, langTag)
	}

	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	// This is syntactic, not semantic
	for _, line := range converter[1:] {
		t := ParseWordType(line[len(line)-1])
		word := &DictNode{W: line[0], Lang: langIndexs[0], Type: t}
		for i, s := range line[1 : len(line)-1] {
			translation := &DictNode{W: s, Lang: langIndexs[i+1], Type: t}
			dict.AddTranslation(word, translation)
			word = translation
		}
	}

	return dict, nil
}

func BuildSearchTree(tDict TranslationDictionary) *RuneNode {
	root := &RuneNode{Next: make(map[rune]*RuneNode)}
	root.Root = root

	for _, dict := range tDict {
		for s, wNode := range dict {
			rNode := root

			for _, r := range s {
				next, ok := rNode.Next[r]
				if !ok {
					next = &RuneNode{Root: root, Next: make(map[rune]*RuneNode)}
					rNode.Next[r] = next
				}
				rNode = next
			}

			rNode.DNode = wNode
		}
	}

	return root
}

// Searches one step into the tree
//
// Returns the next node if not nil and true, if nil it returns the root node, and false
func (node *RuneNode) SearchStep(r rune) (*RuneNode, bool) {
	next, ok := node.Next[r]
	if ok {
		node = next
		return node, true
	} else {
		node = node.Root
		return node, false
	}
}

func BuildDictionary(text ...string) error {
	converterPath := path.Join(DictionaryPath, ConverterDictionaryName+".csv")
	data, err := os.ReadFile(converterPath)
	if err != nil {
		return err
	}
	LoadedDictionary, err = ReadConverter(data)
	if err != nil {
		return err
	}

	for l, dict := range LoadedDictionary {
		for s, w := range dict {
			w.Frequency = strings.Count(strings.Join(text, ""), s)
			LoadedDictionary[l][s] = w
			if w.Frequency > MaxFrequency {
				MaxFrequency = w.Frequency
			}
		}
	}

	DictionaryTree = BuildSearchTree(LoadedDictionary)

	return nil
}
