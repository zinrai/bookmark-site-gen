package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

type Bookmark struct {
	URL  string
	Hash string
}

func LoadBookmarks(path string) ([]Bookmark, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var urls []string
	if err := json.Unmarshal(data, &urls); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	bookmarks := make([]Bookmark, len(urls))
	for i, url := range urls {
		bookmarks[i] = Bookmark{
			URL:  url,
			Hash: hashURL(url),
		}
	}

	return bookmarks, nil
}

func hashURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:])
}
