package lib

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

const DictionaryDirName = "dictionaries"

var (
	WordLists []string
	Dictionary = make(map[string]word)
	StarLocation, _ = os.Executable()
	DictionaryPath = path.Join(StarLocation, DictionaryDirName)
)

// makes word all lowercase and removes any extra white space
func CleanUpWord(w string) string {
	return strings.ToLower(strings.TrimSpace(w))
}

func GetListNames() error {
	dir, err := os.ReadDir(DictionaryPath)
	if err != nil {
		return err
	}
	WordLists = make([]string, 0, len(dir))
	for i := range WordLists {
		name := strings.TrimSpace(dir[i].Name())
		if strings.TrimRight(name, ".") != "" && strings.HasSuffix(name, ".list") {
			WordLists = append(WordLists, strings.TrimRight(name, "."))
		}
	}
	return nil
}

func GetWordsFromText(text ...word) ([]word, error) {
	words := make(map[string]int)
	for _, w := range text {
		words[strings.ToLower(strings.TrimSpace(w.Translation))]++
	}
	var wordFrequencies []word
	for w, frequency := range words {
		wordFrequencies = append(wordFrequencies, word{Translation: w, Frequency: frequency})
	}
	return wordFrequencies, nil
}

func NewConversionList(csvPath, fromList string, toLang language.Tag) (map[string]word, error) {
	data, err := os.ReadFile(path.Join(DictionaryPath, fromList+".list"))
	if err != nil {
		return nil, err
	}
	list := strings.Split(string(data), "\n")

	ctx := context.Background()
	client, err := translate.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	//nolint
	defer client.Close()
	translations, err := client.Translate(ctx, list, toLang, nil)
	if err != nil {
		return nil, err
	}

	converter := make([][]string, 0, len(list))
	dict := make(map[string]word, len(list))
	for i, s := range list {
		converter = append(converter, []string{CleanUpWord(s), CleanUpWord(translations[i].Text)})
		dict[CleanUpWord(s)] = word{Translation: CleanUpWord(translations[i].Text)}
	}

	file, err := os.Create(csvPath)
	if err != nil {
		return nil, err
	}
	//nolint
	defer file.Close()
	err = csv.NewWriter(file).WriteAll(converter)
	if err != nil {
		return nil, err
	}

	return dict, nil
}

func ReadConvertionList(data []byte) (map[string]word, error) {
	listReader := csv.NewReader(bytes.NewReader(data))
	list, err := listReader.ReadAll()
	if err != nil {
		return nil, err
	}
	dict := make(map[string]word)
	for _, line := range list {
		dict[CleanUpWord(line[0])] = word{Translation: CleanUpWord(line[1])}
	}
	return dict, nil
}

func BuildDictionary(fromList string, toLang language.Tag, text ...word) error {
	converterPath := path.Join(DictionaryPath, fromList+"-"+toLang.String()+".csv")
	data, err := os.ReadFile(converterPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		Dictionary, err = NewConversionList(converterPath, fromList, toLang)
		if err != nil {
			return err
		}
	}
	Dictionary, err = ReadConvertionList(data)
	if err != nil {
		return err
	}
	return nil
}
