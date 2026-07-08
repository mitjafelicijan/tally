package main

import "time"

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

type SourceInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
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
