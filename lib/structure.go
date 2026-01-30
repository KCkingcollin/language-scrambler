package lib

import "golang.org/x/text/language"

// Text source language will be detected
type ScramOpts struct{
	Lang language.Tag
	// should increase the complexity and replace less common words more often as the difficulty is raised
	Difficulty float32
	// what the difficulty should be by the end of the text
	DifGradient float32
}

type word struct {
	Translation string
	Frequency int
}
