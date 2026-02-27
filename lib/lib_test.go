package lib

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/text/language"
)

func TestBuildTree(t *testing.T) {
	os.Chdir("..") //nolint

	words, err := LoadList(language.English)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	wordsMap := make(map[string]Word)
	for _, w := range words {
		wordsMap[w.W] = w
	}
	fmt.Println("size of", language.English.String(), "list:", len(wordsMap))

	var b bool
	searchTree := BuildSearchTree(wordsMap)

	for s := range wordsMap {
		for _, r := range s {
			searchTree, b = searchTree.SearchStep(r)
			if !b {
				t.Log("Failed to find", s, "in search tree")
				t.FailNow()
			}
		}
		searchTree = searchTree.Root
	}
}

func TestBuildDictionary(t *testing.T) {
	text := "Programmieren ist eine kreative und zugleich logische Tätigkeit, bei der man Ideen in funktionierende digitale Lösungen verwandelt. Durch Code lassen sich Probleme systematisch analysieren, Prozesse automatisieren und innovative Anwendungen entwickeln, die den Alltag erleichtern oder ganze Branchen verändern. Dabei erfordert Coding nicht nur technisches Wissen, sondern auch Geduld, strukturiertes Denken und die Bereitschaft, aus Fehlern zu lernen und sich ständig weiterzuentwickeln."
	if err := os.Chdir("/home/kckingcollin/go/src/github.com/KCkingcollin/language-scrambler"); err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err := BuildDictionary(GetBase(language.German), GetBase(language.English), text)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}
