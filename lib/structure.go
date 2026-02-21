package lib

import (
	"log"

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

	word struct {
		Translation string
		Frequency int
	}

	runeNode struct {
		R rune
		next map[rune]runeNode
	}

	TranslateResponse struct {
		TranslatedText string `json:"translatedText"`
	}
)

var (
	SupportedLangs = []language.Base{
		GetBase("en"), 
		GetBase("de"), 
		GetBase("ja"),
		GetBase("es"), 
		//need to make lists for these
		GetBase("ru"),
		GetBase("sv"),
		GetBase("pt"),
		GetBase("nl"),
		GetBase("it"),
		GetBase("fr"),
		GetBase("fr"),
		GetBase("eo"),
		GetBase("no"),
		GetBase("zh"),
	}
)

func GetBase(name string) language.Base {
	base, err := language.ParseBase(name)
	if err != nil {
		log.Println("error finding base for", name)
	}
	return base
}

