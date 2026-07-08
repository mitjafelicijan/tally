package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"
)

var synonyms map[string]string

func main() {
	// Load feed configuration and WordNet synonyms
	configData, err := os.ReadFile("feeds.json")
	if err != nil {
		log.Fatal("Error reading feeds.json:", err)
	}

	var feedURLs []string
	if err := json.Unmarshal(configData, &feedURLs); err != nil {
		log.Fatal("Error unmarshaling feeds.json:", err)
	}

	synonymData, err := os.ReadFile("synonyms.json")
	if err == nil {
		if err := json.Unmarshal(synonymData, &synonyms); err != nil {
			log.Printf("Warning: error unmarshaling synonyms.json: %v", err)
		}
	}

	// Aggregate and deduplicate latest news
	items, sourceMap := fetchFeeds(feedURLs)

	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	// Save latest results to JSON database
	dateString := time.Now().Format("2006-01-02")
	jsonData, _ := json.MarshalIndent(items, "", "  ")

	os.WriteFile("latest.json", jsonData, 0644)
	os.WriteFile("public/latest.json", jsonData, 0644)
	os.WriteFile("public/"+dateString+".json", jsonData, 0644)

	log.Printf("Saved %d items. Synchronizing site...", len(items))

	// Rebuild entire static site for global consistency
	synchronizeSite(items, sourceMap)
}
