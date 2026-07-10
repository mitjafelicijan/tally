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
	configData, err := os.ReadFile(FeedsConfigFile)
	if err != nil {
		log.Fatal("Error reading feeds.json:", err)
	}

	var feedURLs []string
	if err := json.Unmarshal(configData, &feedURLs); err != nil {
		log.Fatal("Error unmarshaling feeds.json:", err)
	}

	synonymData, err := os.ReadFile(SynonymsFile)
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

	// Ensure public directory exists
	if err := os.MkdirAll(PublicDir, DirPerm); err != nil {
		log.Fatal("Error creating public directory:", err)
	}

	// Save latest results to JSON database
	dateString := time.Now().Format("2006-01-02")
	jsonData, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling JSON:", err)
	}

	if err := os.WriteFile(PublicDir+"/"+LatestJSONFile, jsonData, FilePerm); err != nil {
		log.Printf("Warning: error writing latest.json: %v", err)
	}
	if err := os.WriteFile(PublicDir+"/"+dateString+".json", jsonData, FilePerm); err != nil {
		log.Printf("Warning: error writing %s.json: %v", dateString, err)
	}

	log.Printf("Saved %d items. Synchronizing site...", len(items))

	// Rebuild entire static site for global consistency
	synchronizeSite(items, sourceMap)
}
