package main

import (
)

// func TestFixConverter(t *testing.T) {
// 	os.Chdir("..") //nolint
//
// 	dict := make(TranslationDictionary)
// 	for _, l := range SupportedLangs {
// 		dict[l.Tag] = make(Dictionary)
// 	}
//
// 	for _, lang := range SupportedLangs {
// 		list, err := LoadList(lang.Tag)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		maps.Copy(dict[lang.Tag], list)
// 	}
//
// 	converterPath := path.Join(DictionaryPath, ConverterDictionaryName+".csv")
// 	data, err := os.ReadFile(converterPath)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	converterReader := csv.NewReader(bytes.NewReader(data))
// 	converter, err := converterReader.ReadAll()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	var langIndexs []language.Tag
// 	for _, lang := range converter[0] {
// 		langTag, err := language.Parse(lang)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		langIndexs = append(langIndexs, langTag)
// 	}
//
// 	for _, line := range converter[1:] {
// 		word, ok := dict[langIndexs[0]][CleanUpWord(line[0])]
// 		if !ok {
// 			continue
// 		}
// 		for i, s := range line[1:] {
// 			translation, ok := dict[langIndexs[i+1]][CleanUpWord(s)]
// 			if !ok {
// 				translation = &DictNode{W: s, Lang: langIndexs[i+1], Type: word.Type}
// 			}
// 			dict.AddTranslation(word, translation)
// 			word = translation
// 		}
// 	}
//
// 	err = SaveConverter(dict, converterPath, langIndexs)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// } 

// func TestTranslate(t *testing.T) {
// 	text := []string{
// 		"hello", 
// 		"africa", 
// 		"Universe",
// 	}
// 	expected := []string{
// 		"hallo", 
// 		"afrika",
// 		"universum",
// 	}
// 	translation, err := TranslateWords(text, GetBase("en"), GetBase("de"))
// 	if err != nil {
// 		fmt.Println(err)
// 		t.FailNow()
// 	}
// 	if slices.Compare(translation, expected) != 0 {
// 		t.FailNow()
// 	}
//
// 	Translater.Dictionary = nil
// 	Translater.Lang = language.Base{}
// 	runtime.GC()
//
// 	text = []string{
// 		"hallo", 
// 		"afrika",
// 		"Universum",
// 	}
// 	expected = []string{
// 		"hello", 
// 		"africa", 
// 		"universe",
// 	}
// 	translation, err = TranslateWords(text, GetBase("de"), GetBase("en"))
// 	if err != nil {
// 		fmt.Println(err)
// 		t.FailNow()
// 	}
// 	if slices.Compare(translation, expected) != 0 {
// 		t.FailNow()
// 	}
// }
//
