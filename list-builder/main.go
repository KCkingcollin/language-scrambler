package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	. "langscram-lib" //nolint
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"slices"

	"golang.org/x/net/html"
	"golang.org/x/text/language"
)

const (
	dictionaryLocation = "dictionaries"
)

type (
	scraping struct {
		htmlStructure []string
		classStructure []string
		idStructure []string
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

func wiktionaryScraper(lang language.Tag, urls ...string) (Dictionary, error) {
	list := make(Dictionary)
	client := &http.Client{}
	wordCount := make([]int, len(urls))
	for i, nextPage := range urls {
		var wordType reflect.Type
		if i < 1 {
			wordType = reflect.TypeFor[Verb]()
		} else {
			wordType = reflect.TypeFor[Noun]()
		}
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
									word := &DictNode{W: li.FirstChild.FirstChild.Data, Lang: lang, Type: wordType}
									if CleanUpWord(word.W) != "" {
										list[CleanUpWord(word.W)] = word
									}
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

	if len(list) < 1 {
		return nil, fmt.Errorf("somehow we got nothing")
	}

	log.Println(len(list), "words", wordCount[0], "verbs", wordCount[1], "nouns")
	return list, nil
}

func GetUrlList() map[language.Tag][]string {
	urlBase := "https://en.wiktionary.org/w/index.php?title=Category:"
	urlList := make(map[language.Tag][]string, len(SupportedLangs))
	for _, l := range SupportedLangs {
		urlList[l.Tag] = []string{urlBase+l.Name+"_verbs", urlBase+l.Name+"_nouns"}
	}
	return urlList
}

// lists built from wiktionary
func buildListSet1() error {
	_, err := os.ReadDir(dictionaryLocation)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s", fmt.Sprintln(dictionaryLocation, "needs to exist in the current dir"))
		} else {
			return err
		}
	}
	for langTag, urls := range GetUrlList() {
		fileLocation := path.Join(dictionaryLocation, langTag.String()+".list")
		if _, err := os.Open(fileLocation); !os.IsNotExist(err) {
			continue
		}

		wordList, err := wiktionaryScraper(langTag, urls...)
		if err != nil {
			return err
		}

		list := make([][]string, 0, len(wordList))
		for _, w := range wordList {
			list = append(list, []string{w.W, w.GetType()})
		}

		file, err := os.Create(fileLocation)
		if err != nil {
			return err
		}
		defer file.Close() //nolint

		buffer := new(bytes.Buffer)

		if err := csv.NewWriter(buffer).WriteAll(list); err != nil {
			return err
		}

		data := make([]byte, buffer.Len())
		if _, err := buffer.Read(data); err != nil {
			return err
		}

		if _, err := file.Write(data); err != nil {
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
