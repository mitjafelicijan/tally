package main

import (
	"html"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

func fetchFeeds(feedURLs []string) ([]FeedItem, map[string]SourceInfo) {
	var allItems []FeedItem
	sourceMap := make(map[string]SourceInfo)
	var acceptedKeywords []map[string]bool

	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	feedParser := gofeed.NewParser()

	currentTime := time.Now()
	cutoffTime := currentTime.Add(-48 * time.Hour)

	log.Printf("Fetching %d feeds...", len(feedURLs))

	for _, url := range feedURLs {
		waitGroup.Add(1)
		go func(url string) {
			defer waitGroup.Done()

			feed, err := feedParser.ParseURL(url)
			if err != nil {
				log.Printf("Warning: error parsing %s: %v", url, err)
				return
			}

			sourceName := extractDomain(feed.Link)
			if sourceName == "" {
				sourceName = extractDomain(url)
			}

			mutex.Lock()
			sourceMap[sourceName] = SourceInfo{Name: sourceName, URL: feed.Link}
			mutex.Unlock()

			log.Printf("Parsed %s (%d items)", url, len(feed.Items))

			for _, item := range feed.Items {
				// Handle date extraction with various fallbacks
				publishedAt := time.Time{}
				if item.PublishedParsed != nil {
					publishedAt = *item.PublishedParsed
				} else if item.UpdatedParsed != nil {
					publishedAt = *item.UpdatedParsed
				} else {
					publishedAt = parseFlexibleDate(item.Published, item.Updated, item.Extensions)
				}

				// Only process stories from the last 48 hours
				if !publishedAt.IsZero() && publishedAt.Before(cutoffTime) {
					continue
				}

				// Deduplication logic using Jaccard similarity and WordNet synonyms
				keywords := extractKeywords(item.Title)
				if len(keywords) == 0 {
					continue
				}

				mutex.Lock()
				isDuplicate := false
				for _, existing := range acceptedKeywords {
					if calculateJaccard(keywords, existing) > 0.5 {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					mutex.Unlock()
					continue
				}
				acceptedKeywords = append(acceptedKeywords, keywords)

				feedItem := FeedItem{
					Title:       html.UnescapeString(item.Title),
					Link:        item.Link,
					Description: stripHTML(item.Description),
					GUID:        item.GUID,
					Date:        publishedAt.Format("Jan 02, 2006 15:04"),
					PublishedAt: publishedAt,
					Source:      sourceName,
					SourceURL:   feed.Link,
				}

				// Extract image from enclosures or MediaRSS
				if item.Image != nil {
					feedItem.Image = item.Image.URL
				} else {
					for _, enclosure := range item.Enclosures {
						if strings.HasPrefix(enclosure.Type, "image/") {
							feedItem.Image = enclosure.URL
							break
						}
					}
				}

				allItems = append(allItems, feedItem)
				mutex.Unlock()
			}
		}(url)
	}

	waitGroup.Wait()
	return allItems, sourceMap
}

func extractKeywords(title string) map[string]bool {
	title = strings.ToLower(title)
	title = reNonAlpha.ReplaceAllString(title, " ")
	words := strings.Fields(title)

	keywords := make(map[string]bool)

	for _, word := range words {
		if stopWords[word] || len(word) < 2 {
			continue
		}

		// Simple suffix removal (stemming)
		if strings.HasSuffix(word, "ing") {
			word = strings.TrimSuffix(word, "ing")
		} else if strings.HasSuffix(word, "ed") {
			word = strings.TrimSuffix(word, "ed")
		} else if strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
			word = strings.TrimSuffix(word, "s")
		}

		if canonical, exists := synonyms[word]; exists {
			word = canonical
		}
		keywords[word] = true
	}
	return keywords
}
