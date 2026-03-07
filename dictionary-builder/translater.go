package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	. "langscram-lib" //nolint

	"golang.org/x/text/language"
)

const libreTranslateURL = "http://localhost:5000/translate"

type (
	libreLang struct {
		Code    string   `json:"code"`
		Name    string   `json:"name"`
		Targets []string `json:"targets"`
	}

	translateRequest struct {
		Q      []string `json:"q"`
		Source string   `json:"source"`
		Target string   `json:"target"`
		Format string   `json:"format"`
	}

	translateResponse struct {
		TranslatedText []string `json:"translatedText"`
	}
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     30 * time.Second,
	},
}

// creates a map describing what can be translated to what in the order of from language to language
func GetTranslationCapabilities() (map[language.Tag]map[language.Tag]bool, error) {
	supportedLangsTag := make(map[language.Tag]bool)
	for _, l := range SupportedLangs {
		supportedLangsTag[l.Tag] = true
	}

	resp, err := httpClient.Get("http://localhost:5000/languages")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("languages endpoint returned %s", resp.Status)
	}

	var langs []libreLang
	if err := json.NewDecoder(resp.Body).Decode(&langs); err != nil {
		return nil, err
	}

	caps := make(map[language.Tag]map[language.Tag]bool)

	for _, l := range langs {
		from, err := language.Parse(l.Code)
		if err != nil {
			continue
		}

		if !supportedLangsTag[from] {
			continue
		}

		if caps[from] == nil {
			caps[from] = make(map[language.Tag]bool)
		}

		for _, t := range l.Targets {
			t = strings.SplitN(t, "-", 2)[0]
			to, err := language.Parse(t)
			if err != nil {
				continue
			}

			if from == to {
				continue
			}

			if !supportedLangsTag[to] {
				continue
			}

			caps[from][to] = true
		}
	}

	return caps, nil
}

func TranslateWords(words []*DictNode, fromLang, toLang language.Tag) ([]string, error) {
	const batchSize = 100
	results := make([]string, len(words))

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	maxWorkers := runtime.NumCPU()

	sem := make(chan struct{}, maxWorkers)
	for i := 0; i < len(words); i += batchSize {
		i, end := i, min(len(words), i+batchSize)
		batch := words[i:end]

		wg.Add(1)
		go func(start int, batch []*DictNode) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			translations, err := translateBatch(batch, fromLang, toLang)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			copy(results[start:start+len(batch)], translations)
		}(i, batch)
	}
	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	return results, nil
}

func translateOnce(fromLang, toLang language.Tag, toTranslate ...string) ([]string, error) {
	reqBody := translateRequest{
		Q:      toTranslate,
		Source: fromLang.String(),
		Target: toLang.String(),
		Format: "text",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Post(
		libreTranslateURL,
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("couldn't translate: %s \nlibretranslate returned %s", strings.Join(toTranslate, ","), resp.Status)
	}

	var tr translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	return tr.TranslatedText, nil
}

func translationFailCascade(fromLang, toLang language.Tag, toTranslate ...string) ([]string, error) {
	switch len(toTranslate) {
	case 0:
		return nil, nil
	case 1:
		return translateOnce(fromLang, toLang, toTranslate...)
	}

	translations, err := translateOnce(fromLang, toLang, toTranslate...)
	if err == nil {
		return translations, err
	}

	halfSize := len(toTranslate) / 2

	partLeft, err := translationFailCascade(fromLang, toLang, toTranslate[:halfSize]...)
	if err != nil {
		return nil, err
	}

	partRight, err := translationFailCascade(fromLang, toLang, toTranslate[halfSize:]...)
	if err != nil {
		return nil, err
	}

	return append(partLeft, partRight...), nil
}

func translateBatch(words []*DictNode, fromLang, toLang language.Tag) ([]string, error) {
	texts := make([]string, len(words))
	for i, w := range words {
		texts[i] = w.W
	}

	translations, err := translationFailCascade(fromLang, toLang, texts...)
	if err != nil {
		return nil, err
	}

	if len(translations) != len(words) {
		return nil, fmt.Errorf(
			"translation count mismatch: sent %d, got %d",
			len(words),
			len(translations),
		)
	}

	return translations, nil
}
