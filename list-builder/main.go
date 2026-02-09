package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

const (
	dictionaryLocation = "../dictionaries"
)

type (
	scraping struct {
		htmlStructure []string
		classStructure []string
		idStructure []string
	}
	listStruct struct {
		wordUrls []string
		fileName string
	}
)

var (
	wikiScrape = scraping{
		htmlStructure: []string{
			"html",
			"body",
		},
		classStructure: []string{
			"mw-page-container",
			"mw-page-container-inner",
			"mw-content-container",
			"mw-body",
			"mw-category-generated",
			"mw-content-ltr",
			"mw-category mw-category-columns",
		},
		idStructure: []string{
			"bodyContent",
			"mw-content-text",
			"mw-pages",
		},
	}
	delistInfo = listStruct{
		wordUrls: []string{"https://en.wiktionary.org/w/index.php?title=Category:German_verbs", "https://en.wiktionary.org/w/index.php?title=Category:German_nouns"},
		fileName: "de",
	}
	jplistInfo = listStruct{
		wordUrls: []string{"https://en.wiktionary.org/w/index.php?title=Category:Japanese_verbs", "https://en.wiktionary.org/w/index.php?title=Category:Japanese_nouns"},
		fileName: "jp",
	}
	enListInfo = listStruct{
		wordUrls: []string{"https://en.wiktionary.org/w/index.php?title=Category:English_verbs", "https://en.wiktionary.org/w/index.php?title=Category:English_nouns"},
		fileName: "en",
	}
	esListInfo = listStruct{
		wordUrls: []string{"https://en.wiktionary.org/w/index.php?title=Category:Spanish_verbs", "https://en.wiktionary.org/w/index.php?title=Category:Spanish_nouns"},
		fileName: "es",
	}
)

func findListNodesWiki(nextPage *string, doc *html.Node) (string, *html.Node) {
	for node := range doc.ChildNodes() {
		if slices.Contains(wikiScrape.htmlStructure, node.Data) && node.Type == html.ElementNode {
			return findListNodesWiki(nextPage, node)
		}
		for _, a := range node.Attr {
			if a.Key == "class" && slices.Contains(wikiScrape.classStructure, a.Val) {
				return findListNodesWiki(nextPage, node)
			}
			if a.Key == "id" {
				if a.Val == "mw-pages" {
					for elm := range node.ChildNodes() {
						if elm.Type == html.ElementNode && elm.Data == "a" && elm.FirstChild.Data == "next page" {
							val := "https://en.wiktionary.org"+elm.Attr[0].Val
							nextPage = &val
							break
						}
					}
				}
				if slices.Contains(wikiScrape.idStructure, a.Val) {
					return findListNodesWiki(nextPage, node)
				}
			}
		}
	}
	return *nextPage, doc
}

func wiktionaryScraper(urls ...string) ([]string, error) {
	var list []string
	client := &http.Client{}
	wordCount := make([]int, len(urls))
	for i, nextPage := range urls {
		var count int
		for nextPage != "" {
			count++
			req, err := http.NewRequest("GET", nextPage, nil)
			if err != nil {
				return nil, err
			}

			req.Header.Set("User-Agent", "VerbAndNounParser/0.0 (KCkingcreation@proton.me) generic-library/0.0")

			response, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer response.Body.Close() //nolint

			doc, err := html.Parse(response.Body)
			if err != nil {
				return nil, err
			}

			fmt.Print(count, ": ", nextPage, "\n")

			nextPage = ""
			nextPage, doc = findListNodesWiki(&nextPage, doc)

			// the lists are sometimes broken into categories so we need to look through those too
			for group := range doc.ChildNodes() {
				if group.Type == html.ElementNode && group.Data == "div" {
					for node := range group.ChildNodes() {
						if node.Type == html.ElementNode && node.Data == "ul" {
							for li := range node.ChildNodes() {
								if li.Type == html.ElementNode {
									list = append(list, li.FirstChild.FirstChild.Data)
								}
							}
						}
					}
				}
			}
		}

		var totalWords int
		for _, num := range wordCount {
			totalWords += num
		}
		wordCount[i] = len(list)-totalWords
	}

	fmt.Println(len(list), "words", wordCount[0], "verbs", wordCount[1], "nouns")
	return list, nil
}

// lists built from wiktionary
func buildListSet1() error {
	listSet := []listStruct{delistInfo, jplistInfo, esListInfo, enListInfo}
	for _, d := range listSet {
		fileLocation := path.Join(dictionaryLocation, d.fileName+".list")
		if _, err := os.Open(fileLocation); !os.IsNotExist(err) {
			continue
		}

		list, err := wiktionaryScraper(d.wordUrls...)
		if err != nil {
			return err
		}

		file, err := os.Create(fileLocation)
		if err != nil {
			return err
		}
		defer file.Close() //nolint

		if _, err := file.Write([]byte(strings.Join(list, "\n"))); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := buildListSet1(); err != nil {
		fmt.Println("Error: ", err)
	}
}
