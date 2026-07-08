package main

import (
	"html"
	"regexp"
	"strings"
	"time"

	ext "github.com/mmcdole/gofeed/extensions"
)

var (
	reNonAlpha = regexp.MustCompile(`[^a-z0-9\s]+`)
	reHTML     = regexp.MustCompile(`<[^>]*>`)

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
)

func stripHTML(content string) string {
	content = reHTML.ReplaceAllString(content, "")
	return html.UnescapeString(content)
}

func calculateJaccard(set1, set2 map[string]bool) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0
	}
	intersection := 0
	for key := range set1 {
		if set2[key] {
			intersection++
		}
	}
	union := len(set1)
	for key := range set2 {
		if !set1[key] {
			union++
		}
	}
	return float64(intersection) / float64(union)
}

func parseFlexibleDate(published, updated string, feedExtensions ext.Extensions) time.Time {
	// Try standard Atom published extension (common in RSS 2.0 mixups)
	if atom, ok := feedExtensions["atom"]; ok {
		if pub, ok := atom["published"]; ok && len(pub) > 0 {
			if t, err := time.Parse(time.RFC3339, pub[0].Value); err == nil {
				return t
			}
		}
	}

	// Try raw strings
	for _, raw := range []string{published, updated} {
		if raw == "" {
			continue
		}
		for _, format := range []string{time.RFC3339, time.RFC1123Z, time.RFC1123, "2006-01-02T15:04:05Z07:00"} {
			if t, err := time.Parse(format, raw); err == nil {
				return t
			}
		}
	}

	// Last ditch: any field in extensions containing "date" or "publish"
	for _, extensions := range feedExtensions {
		for name, values := range extensions {
			lowerName := strings.ToLower(name)
			if (strings.Contains(lowerName, "publish") || strings.Contains(lowerName, "date")) && len(values) > 0 {
				if t, err := time.Parse(time.RFC3339, values[0].Value); err == nil {
					return t
				}
			}
		}
	}

	return time.Time{}
}
