package lib

import (
	"fmt"
	"os"
	"testing"
)

func TestBuildDictionary(t *testing.T) {
	text := "Programmieren ist eine kreative und zugleich logische Tätigkeit, bei der man Ideen in funktionierende digitale Lösungen verwandelt. Durch Code lassen sich Probleme systematisch analysieren, Prozesse automatisieren und innovative Anwendungen entwickeln, die den Alltag erleichtern oder ganze Branchen verändern. Dabei erfordert Coding nicht nur technisches Wissen, sondern auch Geduld, strukturiertes Denken und die Bereitschaft, aus Fehlern zu lernen und sich ständig weiterzuentwickeln."
	if err := os.Chdir("/home/kckingcollin/go/src/github.com/KCkingcollin/language-scrambler"); err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err := BuildDictionary(GetBase("de"), GetBase("en"), text)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

