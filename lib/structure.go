package lib

import (
	"reflect"
	"strings"

	"golang.org/x/text/language"
)

type (
	// Text source language will be detected
	ScramOpts struct{
		List language.Base
		Lang language.Base
		// should increase the complexity and replace less common words more often as the difficulty is raised
		Difficulty float32
		// what the difficulty should be by the end of the text
		DifGradient float32
	}

	Noun struct{}
	Verb struct{}

	Word struct {
		W string
		Translation string
		Frequency int
		Type reflect.Type
	}

	RuneNode struct {
		Root *RuneNode
		Next map[rune]*RuneNode
		W *Word
	}

	Lang struct {
		Name string
		Tag language.Tag
	}

	// string should be a cleaned version, where as the W variable in Word should be the original
	Dictionary map[string]Word
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
