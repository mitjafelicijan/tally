package main

import (
	"encoding/json"
	"html"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedItem struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	GUID        string    `json:"guid"`
	Date        string    `json:"date"`
	PublishedAt time.Time `json:"published_at"`
	Image       string    `json:"image,omitempty"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"source_url"`
}

var (
	synonyms map[string]string
	stopWords = map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "with": true,
		"by": true, "of": true, "about": true, "as": true, "into": true, "like": true,
		"through": true, "after": true, "over": true, "between": true, "out": true,
		"against": true, "during": true, "without": true, "before": true, "under": true,
		"around": true, "among": true, "never": true, "ne'er": true, "goes": true, "go": true, "behind": true,
		"that": true, "this": true, "from": true, "it": true, "its": true, "has": true,
		"have": true, "had": true, "will": true, "can": true, "could": true, "would": true,
		"up": true, "down": true, "more": true, "less": true, "now": true, "just": true,
		"their": true, "they": true, "them": true, "his": true, "her": true, "who": true,
		"which": true, "where": true, "when": true, "why": true, "how": true, "all": true,
		"any": true, "both": true, "each": true, "few": true, "some": true, "such": true,
		"no": true, "nor": true, "not": true, "only": true, "own": true, "same": true,
		"so": true, "than": true, "too": true, "very": true, "s": true, "t": true,
		"don": true, "should": true,
		"plans": true, "releases": true, "released": true, "new": true, "latest": true,
	}
	reNonAlpha = regexp.MustCompile(`[^a-z0-9\s]+`)
	reHTML     = regexp.MustCompile(`<[^>]*>`)
)

func stripHTML(s string) string {
	s = reHTML.ReplaceAllString(s, "")
	return html.UnescapeString(s)
}

func getKeywords(title string) map[string]bool {
	title = strings.ToLower(title)
	title = reNonAlpha.ReplaceAllString(title, " ")
	words := strings.Fields(title)

	keywords := make(map[string]bool)

	for _, w := range words {
		if stopWords[w] || len(w) < 2 {
			continue
		}

		// Simple suffix removal
		if strings.HasSuffix(w, "ing") {
			w = strings.TrimSuffix(w, "ing")
		} else if strings.HasSuffix(w, "ed") {
			w = strings.TrimSuffix(w, "ed")
		} else if strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
			w = strings.TrimSuffix(w, "s")
		}

		// Replace with synonym if exists
		if canonical, exists := synonyms[w]; exists {
			w = canonical
		}
		keywords[w] = true
	}
	return keywords
}

func calculateJaccard(set1, set2 map[string]bool) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0
	}
	intersection := 0
	for k := range set1 {
		if set2[k] {
			intersection++
		}
	}
	union := len(set1)
	for k := range set2 {
		if !set1[k] {
			union++
		}
	}
	return float64(intersection) / float64(union)
}

type CalendarDay struct {
	Day     int
	Link    string
	Current bool
}

type CalendarData struct {
	MonthName string
	Weeks     [][]CalendarDay
}

type SourceInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ArchiveMonth struct {
	Name     string
	FirstDay string
}

type ArchiveYear struct {
	Year   int
	Months []ArchiveMonth
}

type PageData struct {
	ViewTitle string
	Items     []FeedItem
	Calendar  CalendarData
	Sources   []SourceInfo
	Archives  []ArchiveYear
}

func generateCalendarData(viewDate time.Time, highlightDay int, jsonFiles map[string]bool) CalendarData {
	firstOfMonth := time.Date(viewDate.Year(), viewDate.Month(), 1, 0, 0, 0, 0, viewDate.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	var weeks [][]CalendarDay
	var currentWeek []CalendarDay
	startPadding := int(firstOfMonth.Weekday())
	for i := 0; i < startPadding; i++ {
		currentWeek = append(currentWeek, CalendarDay{})
	}

	for d := 1; d <= lastOfMonth.Day(); d++ {
		dayDate := time.Date(viewDate.Year(), viewDate.Month(), d, 0, 0, 0, 0, viewDate.Location())
		dateKey := dayDate.Format("2006-01-02")
		link := ""
		if jsonFiles[dateKey+".json"] {
			link = dateKey + ".html"
		}
		currentWeek = append(currentWeek, CalendarDay{Day: d, Link: link, Current: d == highlightDay})
		if len(currentWeek) == 7 {
			weeks = append(weeks, currentWeek)
			currentWeek = []CalendarDay{}
		}
	}
	if len(currentWeek) > 0 {
		for len(currentWeek) < 7 {
			currentWeek = append(currentWeek, CalendarDay{})
		}
		weeks = append(weeks, currentWeek)
	}
	return CalendarData{
		MonthName: viewDate.Month().String() + " " + viewDate.Format("2006"),
		Weeks:     weeks,
	}
}

func main() {
	// 1. Read feeds.json and synonyms.json
	data, err := os.ReadFile("feeds.json")
	if err != nil {
		log.Fatal("Error reading feeds.json:", err)
	}

	var feedURLs []string
	if err := json.Unmarshal(data, &feedURLs); err != nil {
		log.Fatal("Error unmarshaling feeds.json:", err)
	}

	synData, err := os.ReadFile("synonyms.json")
	if err == nil {
		if err := json.Unmarshal(synData, &synonyms); err != nil {
			log.Printf("Warning: error unmarshaling synonyms.json: %v", err)
		}
	} else {
		log.Printf("Warning: synonyms.json not found, proceeding without it")
	}

	// 2. Parse feeds
	var items []FeedItem
	var acceptedKeywords []map[string]bool
	sourceMap := make(map[string]SourceInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup
	fp := gofeed.NewParser()

	log.Printf("Starting to fetch %d feeds...", len(feedURLs))

	for _, url := range feedURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			log.Printf("Fetching: %s", url)
			feed, err := fp.ParseURL(url)
			if err != nil {
				log.Printf("Warning: error parsing %s: %v", url, err)
				return
			}

			mu.Lock()
			sourceMap[feed.Title] = SourceInfo{Name: feed.Title, URL: feed.Link}
			mu.Unlock()

			log.Printf("Successfully parsed %s (%d items found)", url, len(feed.Items))

			now := time.Now()
			cutoff := now.Add(-48 * time.Hour)

			for _, item := range feed.Items {
				pubAt := time.Time{}
				if item.PublishedParsed != nil {
					pubAt = *item.PublishedParsed
				} else if item.UpdatedParsed != nil {
					pubAt = *item.UpdatedParsed
				} else {
					// Manual extraction from extensions as a last resort
					rawDate := item.Published
					if rawDate == "" {
						rawDate = item.Updated
					}
					if rawDate == "" {
						if atom, ok := item.Extensions["atom"]; ok {
							if pub, ok := atom["published"]; ok && len(pub) > 0 {
								rawDate = pub[0].Value
							}
						}
					}
					if rawDate == "" {
						for _, ext := range item.Extensions {
							for name, values := range ext {
								if strings.Contains(strings.ToLower(name), "publish") || strings.Contains(strings.ToLower(name), "date") {
									if len(values) > 0 {
										rawDate = values[0].Value
										break
									}
								}
							}
							if rawDate != "" {
								break
							}
						}
					}

					// Try to parse the rawDate we found
					if rawDate != "" {
						t, err := time.Parse(time.RFC3339, rawDate)
						if err != nil {
							t, err = time.Parse(time.RFC1123Z, rawDate)
						}
						if err != nil {
							t, err = time.Parse(time.RFC1123, rawDate)
						}
						if err == nil {
							pubAt = t
						}
					}
				}

				// Skip stories older than 48 hours
				if !pubAt.IsZero() && pubAt.Before(cutoff) {
					continue
				}

				keywords := getKeywords(item.Title)
				if len(keywords) == 0 {
					continue
				}

				mu.Lock()
				isDuplicate := false
				for _, existing := range acceptedKeywords {
					if calculateJaccard(keywords, existing) > 0.5 {
						isDuplicate = true
						break
					}
				}
				if isDuplicate {
					log.Printf("Duplicate topic caught: %s", item.Title)
					mu.Unlock()
					continue
				}
				acceptedKeywords = append(acceptedKeywords, keywords)

				fi := FeedItem{
					Title:       html.UnescapeString(item.Title),
					Link:        item.Link,
					Description: stripHTML(item.Description),
					GUID:        item.GUID,
					Date:        pubAt.Format("Jan 02, 2006 15:04"),
					PublishedAt: pubAt,
					Source:      feed.Title,
					SourceURL:   feed.Link,
				}

				// Try to find an image
				if item.Image != nil {
					fi.Image = item.Image.URL
				} else if len(item.Enclosures) > 0 {
					for _, enc := range item.Enclosures {
						if enc.Type == "image/jpeg" || enc.Type == "image/png" || enc.Type == "image/gif" {
							fi.Image = enc.URL
							break
						}
					}
				}

				items = append(items, fi)
				mu.Unlock()
			}
		}(url)
	}

	wg.Wait()

	// Sort items by PublishedAt descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})

	// Collect and sort sources
	var sources []SourceInfo
	for _, s := range sourceMap {
		sources = append(sources, s)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})

	// 3. Save current JSON
	os.MkdirAll("public", 0755)
	dateStr := time.Now().Format("2006-01-02")
	output, _ := json.MarshalIndent(items, "", "  ")
	
	// Save to root and public
	os.WriteFile("latest.json", output, 0644)
	os.WriteFile("public/latest.json", output, 0644)
	os.WriteFile("public/"+dateStr+".json", output, 0644)

	log.Printf("Regenerating all historical pages for total sync...")

	// 4. Scan for ALL JSON files to build sidebar
	jsonFiles := make(map[string]bool)
	entries, _ := os.ReadDir("public")
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "latest.json" {
			jsonFiles[e.Name()] = true
		}
	}

	yearsMap := make(map[int]map[time.Month]string)
	for fileName := range jsonFiles {
		datePart := fileName[:10]
		t, err := time.Parse("2006-01-02", datePart)
		if err == nil {
			if yearsMap[t.Year()] == nil {
				yearsMap[t.Year()] = make(map[time.Month]string)
			}
			existingFirst := yearsMap[t.Year()][t.Month()]
			htmlName := datePart + ".html"
			if existingFirst == "" || htmlName < existingFirst {
				yearsMap[t.Year()][t.Month()] = htmlName
			}
		}
	}

	var archives []ArchiveYear
	var sortedYears []int
	for y := range yearsMap {
		sortedYears = append(sortedYears, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedYears)))
	for _, y := range sortedYears {
		yearObj := ArchiveYear{Year: y}
		for m := time.January; m <= time.December; m++ {
			if firstDay, exists := yearsMap[y][m]; exists {
				yearObj.Months = append(yearObj.Months, ArchiveMonth{Name: m.String()[:3], FirstDay: firstDay})
			}
		}
		archives = append(archives, yearObj)
	}

	// 5. Load template and render EVERYTHING
	tmpl, err := template.ParseFiles("template.html")
	if err != nil {
		log.Fatal(err)
	}

	for jsonFile := range jsonFiles {
		content, _ := os.ReadFile("public/" + jsonFile)
		var dayItems []FeedItem
		json.Unmarshal(content, &dayItems)

		datePart := jsonFile[:10]
		archiveDate, _ := time.Parse("2006-01-02", datePart)

		htmlFile := strings.TrimSuffix(jsonFile, ".json") + ".html"
		f, _ := os.Create("public/" + htmlFile)
		tmpl.Execute(f, PageData{
			ViewTitle: archiveDate.Format("January 02, 2006"),
			Items:     dayItems,
			Calendar:  generateCalendarData(archiveDate, archiveDate.Day(), jsonFiles),
			Sources:   sources,
			Archives:  archives,
		})
		f.Close()
		log.Printf("Generated archive: %s", htmlFile)
	}

	// Finally, render index.html (latest)
	f, _ := os.Create("public/index.html")
	now := time.Now()
	tmpl.Execute(f, PageData{
		ViewTitle: "Latest Stories",
		Items:     items,
		Calendar:  generateCalendarData(now, now.Day(), jsonFiles),
		Sources:   sources,
		Archives:  archives,
	})
	f.Close()
	log.Printf("Generated index.html")

	log.Printf("Successfully synchronized %d archive pages and index.html", len(jsonFiles))
}
