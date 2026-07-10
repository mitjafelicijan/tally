package main

import "time"

const (
	// File and directory configurations
	PublicDir       = "public"
	FeedsConfigFile = "feeds.json"
	SynonymsFile    = "synonyms.json"
	TemplateFile    = "template.html"
	LatestJSONFile  = "latest.json"

	// Processing configurations
	StoryCutoffDuration = 32 * time.Hour
	SimilarityThreshold = 0.5

	// File permissions
	DirPerm  = 0755
	FilePerm = 0644
)
