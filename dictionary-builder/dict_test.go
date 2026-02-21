package main

import (
	"fmt"
	"runtime"
	"slices"
	"testing"

	"golang.org/x/text/language"
	. "langscram-lib" //nolint
)

func TestTranslate(t *testing.T) {
	text := []string{
		"hello", 
		"africa", 
		"Universe",
	}
	expected := []string{
		"hallo", 
		"afrika",
		"universum",
	}
	translation, err := TranslateWords(text, GetBase("en"), GetBase("de"))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if slices.Compare(translation, expected) != 0 {
		t.FailNow()
	}

	Translater.Dictionary = nil
	Translater.Lang = language.Base{}
	runtime.GC()

	text = []string{
		"hallo", 
		"afrika",
		"Universum",
	}
	expected = []string{
		"hello", 
		"africa", 
		"universe",
	}
	translation, err = TranslateWords(text, GetBase("de"), GetBase("en"))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	if slices.Compare(translation, expected) != 0 {
		t.FailNow()
	}
}

