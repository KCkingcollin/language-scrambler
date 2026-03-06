package lib

import (
	"strings"

	"golang.org/x/text/language"
)

type WordType uint8

const (
	WordTypeUnknown WordType = iota
	WordTypeNoun
	WordTypeVerb
)

type (
	// Text source language will be detected
	ScramOpts struct{
		List language.Tag
		Lang language.Tag
		// should increase the complexity and replace less common words more often as the difficulty is raised
		Difficulty float32
		// what the difficulty should be by the end of the text
		DifGradient float32
	}

	DictNode struct {
		ID string
		W string
		// NOT a tree structure, DO NOT traverse
		Translations map[language.Tag]*DictNode
		Frequency int
		Type WordType
		Lang language.Tag
	}

	RuneNode struct {
		Root *RuneNode
		Next map[rune]*RuneNode
		DNode *DictNode
	}

	Lang struct {
		Name string
		Tag language.Tag
	}

	// string should be a cleaned version, where as the W variable in Word should be the original
	Dictionary map[string]*DictNode

	TranslationDictionary map[language.Tag]Dictionary
)

var (
	// Do NOT mutate
	SupportedLangs = []Lang{
		{"English", language.English},
		{"Spanish", language.Spanish},
		{"German", language.German},
		{"Japanese", language.Japanese},
		{"Russian", language.Russian},
		{"Swedish", language.Swedish},
		{"Portuguese", language.Portuguese},
		{"Dutch", language.Dutch},
		{"Italian", language.Italian},
		{"French", language.French},
		{"Chinese", language.Chinese},
	}
)

func ParseWordType(s string) WordType {
	s = strings.ToLower(s)

	switch {
	case strings.Contains(s, "noun"):
		return WordTypeNoun
	case strings.Contains(s, "verb"):
		return WordTypeVerb
	default:
		return WordTypeUnknown
	}
}

func (node *DictNode) GetType() string {
	switch node.Type {
	case WordTypeNoun:
		return "noun"
	case WordTypeVerb:
		return "verb"
	default:
		return ""
	}
}
