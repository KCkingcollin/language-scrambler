package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	. "langscram-lib" //nolint

	"golang.org/x/text/language"
)

const libreTranslateURL = "http://localhost:5000/translate"

type translateRequest struct {
	Q      []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
}

type translateResponse struct {
	TranslatedText []string `json:"translatedText"`
}

var httpClient = &http.Client{
	Timeout: 2 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
	},
}

func TranslateWords(words []*DictNode, fromLang, toLang language.Tag) ([]string, error) {
	const batchSize = 100
	results := make([]string, len(words))
	totalBatches := (len(words) + batchSize - 1) / batchSize

	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	progressCh := make(chan int, totalBatches)

	const maxWorkers = 4

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
			progressCh <- len(batch)
		}(i, batch)
	}

	// Progress printer
	doneCh := make(chan struct{})
	go func() {
		completed := 0
		for n := range progressCh {
			completed += n
			fmt.Printf("\r\033[K%.1f%%", float32(completed)/float32(len(words))*100)
		}
		fmt.Println()
		doneCh <- struct{}{}
	}()

	wg.Wait()
	close(progressCh)
	<-doneCh

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	return results, nil
}

func translateBatch(words []*DictNode, fromLang, toLang language.Tag) ([]string, error) {
	texts := make([]string, len(words))
	for i, w := range words {
		texts[i] = w.W
	}

	reqBody := translateRequest{
		Q:      texts,
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
		return nil, fmt.Errorf("libretranslate returned %s", resp.Status)
	}

	var tr translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	if len(tr.TranslatedText) != len(words) {
		return nil, fmt.Errorf(
			"translation count mismatch: sent %d, got %d",
			len(words),
			len(tr.TranslatedText),
		)
	}

	return tr.TranslatedText, nil
}
