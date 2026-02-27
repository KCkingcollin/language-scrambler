package lib

import (
	"bytes"
	"encoding/csv"
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
	LoadedDictionary = make(Dictionary)
	DictionaryPath = path.Clean(DictionaryDirName)
	WordLists      []string
	DictionaryTree *RuneNode
	MaxFrequency   int
)

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
		word := Word{W: line[0], Type: ParseWordType(line[1])}
		words[CleanUpWord(line[0])] = word
	}
	return words, nil

}

func readConverter(data []byte, toLang language.Base) (map[string]Word, error) {
	converterReader := csv.NewReader(bytes.NewReader(data))
	converter, err := converterReader.ReadAll()
	if err != nil {
		return nil, err
	}
	var langIndex int
	for i, lang := range converter[0] {
		if lang == toLang.String() {
			langIndex = i
		}
	}
	dict := make(map[string]Word)
	for _, line := range converter[1:] {
		dict[line[0]] = Word{Translation: line[langIndex], Type: ParseWordType(line[len(line)-1])}
	}
	return dict, nil
}

func BuildSearchTree(dict Dictionary) *RuneNode {
	root := &RuneNode{Next: make(map[rune]*RuneNode)}
	root.Root = root

	for s, w := range dict {
		node := root

		for _, r := range s {
			next, ok := node.Next[r]
			if !ok {
				next = &RuneNode{Root: root, Next: make(map[rune]*RuneNode)}
				node.Next[r] = next
			}
			node = next
		}

		w.W = s
		node.W = &w
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

func BuildDictionary(fromList language.Base, toLang language.Base, text string) error {
	converterPath := path.Join(DictionaryPath, fromList.String()+"-"+ConverterDictionaryName+".csv")
	data, err := os.ReadFile(converterPath)
	if err != nil {
		return err
	}
	LoadedDictionary, err = readConverter(data, toLang)
	if err != nil {
		return err
	}

	for s, w := range LoadedDictionary {
		w.Frequency = strings.Count(text, s)
		LoadedDictionary[s] = w
		if w.Frequency > MaxFrequency {
			MaxFrequency = w.Frequency
		}
	}

	DictionaryTree = BuildSearchTree(LoadedDictionary)

	return nil
}
