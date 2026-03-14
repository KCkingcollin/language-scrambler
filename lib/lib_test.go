package lib

import (
	"fmt"
	"maps"
	"reflect"
	"testing"

	"golang.org/x/text/language"
)

// createTestDictionary creates a small test dictionary with ~10 words per language
func createTestDictionary() TranslationDictionary {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	testWords := []struct {
		en  string
		es  string
		de  string
		ja  string
		ru  string
		sv  string
		pt  string
		nl  string
		it  string
		fr  string
		zh  string
		typ WordType
	}{
		{"cat", "gato", "Katze", "猫", "кот", "katt", "gato", "kat", "gatto", "chat", "猫", WordTypeNoun},
		{"dog", "perro", "Hund", "犬", "собака", "hund", "cão", "hond", "cane", "chien", "狗", WordTypeNoun},
		{"house", "casa", "Haus", "家", "дом", "hus", "casa", "huis", "casa", "maison", "房子", WordTypeNoun},
		{"water", "agua", "Wasser", "水", "вода", "vatten", "água", "water", "acqua", "eau", "水", WordTypeNoun},
		{"book", "libro", "Buch", "本", "книга", "bok", "livro", "boek", "libro", "livre", "书", WordTypeNoun},
		{"run", "correr", "laufen", "走る", "бежать", "springa", "correr", "lopen", "correre", "courir", "跑", WordTypeVerb},
		{"eat", "comer", "essen", "食べる", "есть", "äta", "comer", "eten", "mangiare", "manger", "吃", WordTypeVerb},
		{"sleep", "dormir", "schlafen", "寝る", "спать", "sova", "dormir", "slapen", "dormire", "dormir", "睡觉", WordTypeVerb},
		{"write", "escribir", "schreiben", "書く", "писать", "skriva", "escrever", "schrijven", "scrivere", "écrire", "写", WordTypeVerb},
		{"read", "leer", "lesen", "読む", "читать", "läsa", "ler", "lezen", "leggere", "lire", "读", WordTypeVerb},
	}

	for _, tw := range testWords {
		translations := make(map[language.Tag]*DictNode)

		nodes := []*DictNode{
			{ID: CleanUpWord(tw.en), W: tw.en, Lang: language.English, Type: tw.typ},
			{ID: CleanUpWord(tw.es), W: tw.es, Lang: language.Spanish, Type: tw.typ},
			{ID: CleanUpWord(tw.de), W: tw.de, Lang: language.German, Type: tw.typ},
			{ID: CleanUpWord(tw.ja), W: tw.ja, Lang: language.Japanese, Type: tw.typ},
			{ID: CleanUpWord(tw.ru), W: tw.ru, Lang: language.Russian, Type: tw.typ},
			{ID: CleanUpWord(tw.sv), W: tw.sv, Lang: language.Swedish, Type: tw.typ},
			{ID: CleanUpWord(tw.pt), W: tw.pt, Lang: language.Portuguese, Type: tw.typ},
			{ID: CleanUpWord(tw.nl), W: tw.nl, Lang: language.Dutch, Type: tw.typ},
			{ID: CleanUpWord(tw.it), W: tw.it, Lang: language.Italian, Type: tw.typ},
			{ID: CleanUpWord(tw.fr), W: tw.fr, Lang: language.French, Type: tw.typ},
			{ID: CleanUpWord(tw.zh), W: tw.zh, Lang: language.Chinese, Type: tw.typ},
		}

		for _, node := range nodes {
			node.Translations = translations
			translations[node.Lang] = node
			dict[node.Lang][node.ID] = node
		}
	}

	return dict
}

func TestCleanUpWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Cat", "cat"},
		{"DOG!", "dog"},
		{"café", "café"},
		{"Hello, World!", "helloworld"},
		{"", ""},
		{"Test123", "test123"},
		{"日本語", "日本語"},
		{"Русский", "русский"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CleanUpWord(tt.input)
			if result != tt.expected {
				t.Errorf("CleanUpWord(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseWordType(t *testing.T) {
	tests := []struct {
		input    string
		expected WordType
	}{
		{"noun", WordTypeNoun},
		{"Noun", WordTypeNoun},
		{"verb", WordTypeVerb},
		{"VERB", WordTypeVerb},
		{"unknown", WordTypeUnknown},
		{"", WordTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseWordType(tt.input)
			if result != tt.expected {
				t.Errorf("ParseWordType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDictNodeGetType(t *testing.T) {
	tests := []struct {
		typ      WordType
		expected string
	}{
		{WordTypeNoun, "noun"},
		{WordTypeVerb, "verb"},
		{WordTypeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			node := &DictNode{Type: tt.typ}
			result := node.GetType()
			if result != tt.expected {
				t.Errorf("GetType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAddTranslation(t *testing.T) {
	dict := createTestDictionary()

	word := &DictNode{ID: "test", W: "test", Lang: language.English, Type: WordTypeNoun}
	translation := &DictNode{ID: "prueba", W: "prueba", Lang: language.Spanish, Type: WordTypeNoun}

	dict.AddTranslation(word, translation)

	if _, ok := dict[language.English]["test"]; !ok {
		t.Error("English word not added to dictionary")
	}

	if _, ok := dict[language.Spanish]["prueba"]; !ok {
		t.Error("Spanish translation not added to dictionary")
	}

	enNode := dict[language.English]["test"]
	if enNode.Translations == nil {
		t.Error("Translations map is nil")
	}

	if _, ok := enNode.Translations[language.Spanish]; !ok {
		t.Error("Spanish translation not linked")
	}
}

func initSets() (map[language.Tag]*DictNode, map[language.Tag]*DictNode) {
	initSet1 := map[language.Tag]*DictNode{
		language.English:    {ID: CleanUpWord("torun"), W: "to run", Lang: language.English, Type: WordTypeNoun},
		language.Spanish:    {ID: CleanUpWord("correr"), W: "correr", Lang: language.Spanish, Type: WordTypeVerb},
		language.German:     {ID: CleanUpWord("rennen"), W: "rennen", Lang: language.German, Type: WordTypeVerb},
		language.Japanese:   {ID: CleanUpWord("走る"), W: "走る", Lang: language.Japanese, Type: WordTypeVerb},
		language.Russian:    {ID: CleanUpWord("бегать"), W: "бегать", Lang: language.Russian, Type: WordTypeVerb},
		language.Swedish:    {ID: CleanUpWord("springa"), W: "springa", Lang: language.Swedish, Type: WordTypeVerb},
		language.Portuguese: {ID: CleanUpWord("correr"), W: "correr", Lang: language.Portuguese, Type: WordTypeVerb},
		language.Dutch:      {ID: CleanUpWord("rennen"), W: "rennen", Lang: language.Dutch, Type: WordTypeVerb},
		language.Italian:    {ID: CleanUpWord("correre"), W: "correre", Lang: language.Italian, Type: WordTypeVerb},
		language.French:     {ID: CleanUpWord("courir"), W: "courir", Lang: language.French, Type: WordTypeVerb},
		language.Chinese:    {ID: CleanUpWord("跑"), W: "跑", Lang: language.Chinese, Type: WordTypeVerb},
	}

	for _, d := range initSet1 {
		d.Translations = initSet1
	}

	initSet2 := map[language.Tag]*DictNode{
		language.English:    {ID: CleanUpWord("to eat"), W: "to eat", Lang: language.English, Type: WordTypeVerb},
		language.Spanish:    {ID: CleanUpWord("comer"), W: "comer", Lang: language.Spanish, Type: WordTypeVerb},
		language.German:     {ID: CleanUpWord("essen"), W: "essen", Lang: language.German, Type: WordTypeVerb},
		language.Japanese:   {ID: CleanUpWord("食べる"), W: "食べる", Lang: language.Japanese, Type: WordTypeVerb},
		language.Russian:    {ID: CleanUpWord("есть"), W: "есть", Lang: language.Russian, Type: WordTypeVerb},
		language.Swedish:    {ID: CleanUpWord("äta"), W: "äta", Lang: language.Swedish, Type: WordTypeNoun},
		language.Portuguese: {ID: CleanUpWord("comer"), W: "comer", Lang: language.Portuguese, Type: WordTypeVerb},
		language.Dutch:      {ID: CleanUpWord("eten"), W: "eten", Lang: language.Dutch, Type: WordTypeVerb},
		language.Italian:    {ID: CleanUpWord("mangiare"), W: "mangiare", Lang: language.Italian, Type: WordTypeVerb},
		language.French:     {ID: CleanUpWord("manger"), W: "manger", Lang: language.French, Type: WordTypeVerb},
		language.Chinese:    {ID: CleanUpWord("吃"), W: "吃", Lang: language.Chinese, Type: WordTypeVerb},
	}

	for _, d := range initSet2 {
		d.Translations = initSet2
	}

	return initSet1, initSet2
}

func TestAddTranslationMerging(t *testing.T) {
	tests := []struct {
		name string
		op   func() error
	}{
		{"FullSetTest", addTranslationFullSetTest},
		{"SplitSetTest", addTranslationSplitSetTest},
		{"SimilarSetTest", addTranslationSimilarSetTest},
		{"HalfSimilarSetTest", addTranslationHalfSimilarSetTest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if err != nil {
				t.Errorf("%s: %s", tt.name, err)
			}
		})
	}
}

func addTranslationFullSetTest() error {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	set1, set2 := initSets()
	expectedSet1 := maps.Clone(set1)
	expectedSet2 := maps.Clone(set2)

	for _, set := range []map[language.Tag]*DictNode{set1, set2} {
		for _, d := range set {
			dict[d.Lang][d.ID] = d
		}
	}

	var word, translation *DictNode

	for _, d := range set1 {
		word = d
		break
	}

	for _, d := range set2 {
		translation = d
		break
	}

	if word == nil || translation == nil {
		return fmt.Errorf("failed to select nodes from sets")
	}

	dict.AddTranslation(word, translation)

	set1Ptr := reflect.ValueOf(set1).Pointer()
	set2Ptr := reflect.ValueOf(set2).Pointer()

	for l, d := range expectedSet1 {
		if set1[l] != d {
			return fmt.Errorf("set1[%s] (%s) was not the expectedSet1[%s] (%s)", l, set1[l].ID, d.Lang, d.ID)
		}
		nodeTranslationMapPtr := reflect.ValueOf(d.Translations).Pointer()
		if nodeTranslationMapPtr != set1Ptr {
			return fmt.Errorf("set1[%s] (%s) translation map was incorrect, expected %d, got %d", l, set1[l].ID, set1Ptr, nodeTranslationMapPtr)
		}
	}

	for l, d := range expectedSet2 {
		if set2[l] != d {
			return fmt.Errorf("set2[%s] (%s) was not the expectedSet2[%s] (%s)", l, set2[l].ID, d.Lang, d.ID)
		}
		nodeTranslationMapPtr := reflect.ValueOf(d.Translations).Pointer()
		if nodeTranslationMapPtr != set2Ptr {
			return fmt.Errorf("set2[%s] (%s) translation map was incorrect, expected %d, got %d", l, set2[l].ID, set2Ptr, nodeTranslationMapPtr)
		}
	}

	return nil
}

func addTranslationSplitSetTest() error {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	initA, _ := initSets()
	set1 := make(map[language.Tag]*DictNode)
	set2 := make(map[language.Tag]*DictNode)

	var testList []*DictNode

	for _, d := range initA {
		testList = append(testList, d)
	}

	splitIndex := len(initA) / 2

	for _, d := range testList[:splitIndex] {
		d.Translations = set1
		set1[d.Lang] = d
	}

	for _, d := range testList[splitIndex:] {
		d.Translations = set2
		set2[d.Lang] = d
	}

	for _, set := range []map[language.Tag]*DictNode{set1, set2} {
		for _, d := range set {
			dict[d.Lang][d.ID] = d
		}
	}

	var word, translation *DictNode

	for _, d := range set1 {
		word = d
		break
	}

	for _, d := range set2 {
		translation = d
		break
	}

	if word == nil || translation == nil {
		return fmt.Errorf("failed to select nodes from sets")
	}

	dict.AddTranslation(word, translation)

	wordTranslationsPtr := reflect.ValueOf(word.Translations).Pointer()
	translationTranslationsPtr := reflect.ValueOf(translation.Translations).Pointer()

	if wordTranslationsPtr != translationTranslationsPtr {
		return fmt.Errorf("word translation map ptr %d was not the same as the translation's translation map ptr %d", wordTranslationsPtr, translationTranslationsPtr)
	}

	for l, d := range initA {
		if d != word.Translations[l] {
			return fmt.Errorf("translations did not contain all of initA\ninitA:\n%v\n\ntranslations:\n%v", initA, word.Translations)
		}
	}

	return nil
}

func addTranslationSimilarSetTest() error {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	initA, _ := initSets()
	set1 := make(map[language.Tag]*DictNode)
	set2 := make(map[language.Tag]*DictNode)

	var testList []*DictNode

	for _, d := range initA {
		testList = append(testList, d)
	}

	splitIndex := len(initA) / 2

	for _, d := range testList[:splitIndex] {
		d.Translations = set1
		set1[d.Lang] = d
	}

	for _, d := range testList[min(len(testList)-1, splitIndex+1):] {
		d.Translations = set2
		set2[d.Lang] = d
	}

	for _, set := range []map[language.Tag]*DictNode{set1, set2} {
		for _, d := range set {
			dict[d.Lang][d.ID] = d
		}
	}

	var word, translation *DictNode

	for _, d := range set1 {
		word = d
		break
	}

	for _, d := range set2 {
		translation = d
		break
	}

	if word == nil || translation == nil {
		return fmt.Errorf("failed to select nodes from sets")
	}

	dict.AddTranslation(word, translation)

	wordTranslationsPtr := reflect.ValueOf(word.Translations).Pointer()
	translationTranslationsPtr := reflect.ValueOf(translation.Translations).Pointer()

	if wordTranslationsPtr != translationTranslationsPtr {
		return fmt.Errorf("word translation map ptr %d was not the same as the translation's translation map ptr %d", wordTranslationsPtr, translationTranslationsPtr)
	}

	for l, d := range initA {
		if d != word.Translations[l] {
			return fmt.Errorf("translations did not contain all of initA\ninitA:\n%v\n\ntranslations:\n%v", initA, word.Translations)
		}
	}

	return nil
}

func addTranslationHalfSimilarSetTest() error {
	dict := make(TranslationDictionary)
	for _, l := range SupportedLangs {
		dict[l.Tag] = make(Dictionary)
	}

	initA, set2 := initSets()
	initB := maps.Clone(set2)

	// We use shallow clones here because we only care about pointer identity,
	// not the internal state of the nodes. This verifies that AddTranslation
	// did not replace any node pointers in the set.
	expectedSet1 := maps.Clone(initA)
	expectedSet2 := maps.Clone(initB)

	set1 := make(map[language.Tag]*DictNode)

	var testList []*DictNode

	for _, d := range initA {
		testList = append(testList, d)
	}

	splitIndex := len(initA) / 2

	insertWord := initB[language.Swedish]
	insertWord.Translations = set1

	for _, d := range testList[:splitIndex] {
		if d.Lang == insertWord.Lang {
			set1[d.Lang] = insertWord
		} else {
			d.Translations = set1
			set1[d.Lang] = d
		}
	}

	for _, d := range testList[splitIndex:] {
		expectedSet1[d.Lang] = expectedSet2[d.Lang]
	}

	for _, set := range []map[language.Tag]*DictNode{set1, set2} {
		for _, d := range set {
			dict[d.Lang][d.ID] = d
		}
	}

	var word, translation *DictNode

	for _, d := range set1 {
		word = d
		break
	}

	for _, d := range set2 {
		translation = d
		break
	}

	if word == nil || translation == nil {
		return fmt.Errorf("failed to select nodes from sets")
	}

	dict.AddTranslation(word, translation)

	wordTranslationsPtr := reflect.ValueOf(word.Translations).Pointer()
	translationTranslationsPtr := reflect.ValueOf(translation.Translations).Pointer()

	if wordTranslationsPtr == translationTranslationsPtr {
		return fmt.Errorf("word and translation points to the same map: %d", wordTranslationsPtr)
	}

	set1Ptr := reflect.ValueOf(set1).Pointer()
	set2Ptr := reflect.ValueOf(set2).Pointer()

	for l, d := range expectedSet1 {
		if set1[l] == nil {
			return fmt.Errorf("we didn't even have anything in set1[%s] >:(", l)
		}
		if set1[l] != d {
			return fmt.Errorf("set1[%s] (%s) was not the expectedSet1[%s] (%s)", l, set1[l].ID, d.Lang, d.ID)
		}
		nodeTranslationMapPtr := reflect.ValueOf(d.Translations).Pointer()
		if nodeTranslationMapPtr != set1Ptr {
			return fmt.Errorf("set1[%s] (%s) translation map was incorrect, expected %d, got %d", l, set1[l].ID, set1Ptr, nodeTranslationMapPtr)
		}
	}

	for l, d := range expectedSet2 {
		if set2[l] != d {
			return fmt.Errorf("set2[%s] (%s) was not the expectedSet2[%s] (%s)", l, set2[l].ID, d.Lang, d.ID)
		}
		nodeTranslationMapPtr := reflect.ValueOf(d.Translations).Pointer()
		if nodeTranslationMapPtr != set2Ptr {
			return fmt.Errorf("set2[%s] (%s) translation map was incorrect, expected %d, got %d", l, set2[l].ID, set2Ptr, nodeTranslationMapPtr)
		}
	}

	return nil
}

func TestMergeTranslations(t *testing.T) {
	node1 := &DictNode{ID: "cat", W: "cat", Lang: language.English, Type: WordTypeNoun}
	node2 := &DictNode{ID: "gato", W: "gato", Lang: language.Spanish, Type: WordTypeNoun}

	translations := MergeTranslations(node1, node2)

	if len(translations) != 2 {
		t.Errorf("Expected 2 translations, got %d", len(translations))
	}

	if translations[language.English] == nil {
		t.Error("English node not in merged translations")
	}

	if translations[language.Spanish] == nil {
		t.Error("Spanish node not in merged translations")
	}
}

func TestBuildSearchTree(t *testing.T) {
	dict := createTestDictionary()
	tree := BuildSearchTree(dict)

	if tree == nil {
		t.Fatal("Search tree is nil")
	}

	if tree.Next == nil {
		t.Fatal("Search tree Next map is nil")
	}

	testWord := "cat"
	node := tree
	for _, r := range testWord {
		node, _ = node.SearchStep(r)
	}

	if node.DNode == nil {
		t.Errorf("Could not find word %s in search tree", testWord)
	}
}

func TestRuneNodeSearchStep(t *testing.T) {
	dict := createTestDictionary()
	tree := BuildSearchTree(dict)

	tests := []struct {
		word     string
		expected bool
	}{
		{"cat", true},
		{"dog", true},
		{"xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			node := tree
			found := true
			for _, r := range tt.word {
				node, found = node.SearchStep(r)
				if !found {
					break
				}
			}

			if tt.expected && !found {
				t.Errorf("Word %s should be found", tt.word)
			}
		})
	}
}

func TestTranslationDictionaryStructure(t *testing.T) {
	dict := createTestDictionary()

	for _, lang := range SupportedLangs {
		if dict[lang.Tag] == nil {
			t.Errorf("Dictionary for %s is nil", lang.Name)
		}
	}

	for lang, langDict := range dict {
		for id, node := range langDict {
			if node.ID != id {
				t.Errorf("Node ID mismatch: %s != %s", node.ID, id)
			}
			if node.Lang != lang {
				t.Errorf("Node language mismatch for %s", node.W)
			}
		}
	}
}

func TestSupportedLangs(t *testing.T) {
	if len(SupportedLangs) == 0 {
		t.Error("SupportedLangs is empty")
	}

	seen := make(map[string]bool)
	for _, lang := range SupportedLangs {
		if seen[lang.Name] {
			t.Errorf("Duplicate language: %s", lang.Name)
		}
		seen[lang.Name] = true

		if lang.Tag == language.Und {
			t.Errorf("Language %s has undefined tag", lang.Name)
		}
	}

	if len(SupportedLangs) != 11 {
		t.Errorf("Expected 11 supported languages, got %d", len(SupportedLangs))
	}
}

func BenchmarkCleanUpWord(b *testing.B) {
	testWord := "Hello, World! 123"
	b.ResetTimer()
	for b.Loop() {
		_ = CleanUpWord(testWord)
	}
}

func BenchmarkBuildSearchTree(b *testing.B) {
	dict := createTestDictionary()
	b.ResetTimer()
	for b.Loop() {
		_ = BuildSearchTree(dict)
	}
}

func BenchmarkAddTranslation(b *testing.B) {
	dict := createTestDictionary()
	word := &DictNode{ID: "test", W: "test", Lang: language.English, Type: WordTypeNoun}
	translation := &DictNode{ID: "prueba", W: "prueba", Lang: language.Spanish, Type: WordTypeNoun}

	b.ResetTimer()
	for b.Loop() {
		dict.AddTranslation(word, translation)
	}
}

func BenchmarkParseWordType(b *testing.B) {
	testType := "noun"
	b.ResetTimer()
	for b.Loop() {
		_ = ParseWordType(testType)
	}
}
