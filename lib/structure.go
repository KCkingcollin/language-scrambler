package lib

import (
	"reflect"
	"strings"

	"golang.org/x/text/language"
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

	Noun struct{}
	Verb struct{}

	DictNode struct {
		W string
		Translations map[language.Tag]*DictNode
		Frequency int
		Type reflect.Type
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

func ParseWordType(s string) reflect.Type {
	switch {
	case strings.Contains(s, "noun"):
		return reflect.TypeFor[Noun]()
	case strings.Contains(s, "verb"):
		return reflect.TypeFor[Verb]()
	default:
		return nil
	}
}

func (node *DictNode) GetType() string {
	switch node.Type {
	case reflect.TypeFor[Noun]():
		return "noun"
	case reflect.TypeFor[Verb]():
		return "verb"
	default:
		return ""
	}
}
