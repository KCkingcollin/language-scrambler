package lib

import (
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
