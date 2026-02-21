package lib

import (
	"bytes"
	"encoding/csv"
	"os"
	"path"
	"strings"

	"golang.org/x/text/language"
)

const (
	DictionaryDirName       = "dictionaries"
	ConverterDictionaryName = "converter"
)

var (
	WordLists      []string
	Dictionary     = make(map[string]word)
	DictionaryTree = make(map[rune]runeNode)
	DictionaryPath = path.Clean(DictionaryDirName)
	MaxFrequency   int
)

// makes word all lowercase and removes any extra white space
func CleanUpWord(w string) string {
	return strings.ToLower(strings.TrimSpace(w))
}

func readConverter(data []byte, toLang language.Base) (map[string]word, error) {
	listReader := csv.NewReader(bytes.NewReader(data))
	list, err := listReader.ReadAll()
	if err != nil {
		return nil, err
	}
	var langIndex int
	for i, lang := range list[0] {
		if lang == toLang.String() {
			langIndex = i
		}
	}
	dict := make(map[string]word)
	for _, line := range list[1:] {
		dict[CleanUpWord(line[0])] = word{Translation: CleanUpWord(line[langIndex])}
	}
	return dict, nil
}

func BuildDictionary(fromList language.Base, toLang language.Base, text string) error {
	converterPath := path.Join(DictionaryPath, fromList.String()+"-"+ConverterDictionaryName+".csv")
	data, err := os.ReadFile(converterPath)
	if err != nil {
		return err
	}
	Dictionary, err = readConverter(data, toLang)
	if err != nil {
		return err
	}

	for s, w := range Dictionary {
		w.Frequency = strings.Count(text, s)
		Dictionary[s] = w
		if w.Frequency > MaxFrequency {
			MaxFrequency = w.Frequency
		}

		var node runeNode
		node.next = DictionaryTree
		for _, r := range s {
			newNode, ok := node.next[r]
			if !ok {
				newNode = runeNode{R: r, next: make(map[rune]runeNode)}
				node.next[r] = newNode
			}
			node = newNode
		}
	}

	return nil
}
