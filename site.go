package main

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func generateCalendarData(viewDate time.Time, highlightDay int, jsonFiles map[string]bool) CalendarData {
	firstOfMonth := time.Date(viewDate.Year(), viewDate.Month(), 1, 0, 0, 0, 0, viewDate.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	var weeks [][]CalendarDay
	var currentWeek []CalendarDay

	startPadding := int(firstOfMonth.Weekday())
	for i := 0; i < startPadding; i++ {
		currentWeek = append(currentWeek, CalendarDay{})
	}

	for dayNumber := 1; dayNumber <= lastOfMonth.Day(); dayNumber++ {
		dayDate := time.Date(viewDate.Year(), viewDate.Month(), dayNumber, 0, 0, 0, 0, viewDate.Location())
		dateKey := dayDate.Format("2006-01-02")

		link := ""
		if jsonFiles[dateKey+".json"] {
			link = dateKey + ".html"
		}

		currentWeek = append(currentWeek, CalendarDay{
			Day:     dayNumber,
			Link:    link,
			Current: dayNumber == highlightDay,
		})

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

func synchronizeSite(items []FeedItem, sourceMap map[string]SourceInfo) {
	// Collect global list of JSON archives
	jsonFiles := make(map[string]bool)
	entries, err := os.ReadDir(PublicDir)
	if err != nil {
		log.Printf("Warning: error reading public directory: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != LatestJSONFile {
			jsonFiles[entry.Name()] = true
		}
	}

	// Prepare sorted archives list
	yearsMap := make(map[int]map[time.Month]string)
	for fileName := range jsonFiles {
		datePart := fileName[:10]
		archiveDate, err := time.Parse("2006-01-02", datePart)
		if err == nil {
			if yearsMap[archiveDate.Year()] == nil {
				yearsMap[archiveDate.Year()] = make(map[time.Month]string)
			}
			htmlName := datePart + ".html"
			existingFirst := yearsMap[archiveDate.Year()][archiveDate.Month()]
			if existingFirst == "" || htmlName < existingFirst {
				yearsMap[archiveDate.Year()][archiveDate.Month()] = htmlName
			}
		}
	}

	var archiveYears []ArchiveYear
	var sortedYears []int
	for year := range yearsMap {
		sortedYears = append(sortedYears, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sortedYears)))

	for _, year := range sortedYears {
		yearObj := ArchiveYear{Year: year}
		for month := time.January; month <= time.December; month++ {
			if firstDay, exists := yearsMap[year][month]; exists {
				yearObj.Months = append(yearObj.Months, ArchiveMonth{
					Name:     month.String()[:3],
					FirstDay: firstDay,
				})
			}
		}
		archiveYears = append(archiveYears, yearObj)
	}

	// Prepare sources list
	var sources []SourceInfo
	for _, info := range sourceMap {
		sources = append(sources, info)
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})

	parsedTemplate, err := template.ParseFiles(TemplateFile)
	if err != nil {
		log.Fatal("Error loading template:", err)
	}

	// Regenerate all historical pages
	for jsonFile := range jsonFiles {
		content, err := os.ReadFile(PublicDir + "/" + jsonFile)
		if err != nil {
			log.Printf("Warning: error reading %s: %v", jsonFile, err)
			continue
		}
		var dayItems []FeedItem
		if err := json.Unmarshal(content, &dayItems); err != nil {
			log.Printf("Warning: error unmarshaling %s: %v", jsonFile, err)
			continue
		}

		datePart := jsonFile[:10]
		archiveDate, _ := time.Parse("2006-01-02", datePart)

		htmlFile := strings.TrimSuffix(jsonFile, ".json") + ".html"
		outputFile, err := os.Create(PublicDir + "/" + htmlFile)
		if err != nil {
			log.Printf("Warning: error creating %s: %v", htmlFile, err)
			continue
		}
		formattedDate := archiveDate.Format("January 02, 2006")
		if err := parsedTemplate.Execute(outputFile, PageData{
			WindowTitle: "/bin/news - " + formattedDate,
			ViewTitle:   formattedDate,
			Items:       dayItems,
			Calendar:    generateCalendarData(archiveDate, archiveDate.Day(), jsonFiles),
			Sources:     sources,
			Archives:    archiveYears,
		}); err != nil {
			log.Printf("Warning: error executing template for %s: %v", htmlFile, err)
		}
		outputFile.Close()
	}

	// Generate index.html (latest)
	indexFile, err := os.Create(PublicDir + "/index.html")
	if err != nil {
		log.Printf("Warning: error creating index.html: %v", err)
		return
	}
	now := time.Now()
	if err := parsedTemplate.Execute(indexFile, PageData{
		WindowTitle: "/bin/news",
		ViewTitle:   "Latest Stories",
		Items:       items,
		Calendar:    generateCalendarData(now, now.Day(), jsonFiles),
		Sources:     sources,
		Archives:    archiveYears,
	}); err != nil {
		log.Printf("Warning: error executing template for index.html: %v", err)
	}
	indexFile.Close()

	log.Printf("Site synchronization complete: %d archives regenerated", len(jsonFiles))
}
