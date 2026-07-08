package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	N *Sense `json:"n,omitempty"`
	V *Sense `json:"v,omitempty"`
	A *Sense `json:"a,omitempty"`
	R *Sense `json:"r,omitempty"`
}

type Sense struct {
	Form []string `json:"form,omitempty"`
}

type Synset struct {
	Members []string `json:"members"`
}

func main() {
	dir := "english-wordnet-2025-json"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	lemmas := make(map[string]string)
	synonyms := make(map[string]string)

	// 1. Process entries for lemmas and forms
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "entries-") {
			continue
		}
		path := filepath.Join(dir, file.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		var entries map[string]Entry
		json.Unmarshal(data, &entries)

		for lemma, entry := range entries {
			l := strings.ToLower(strings.ReplaceAll(lemma, "_", " "))
			lemmas[l] = l
			
			// Collect forms
			processForms := func(s *Sense) {
				if s != nil {
					for _, f := range s.Form {
						form := strings.ToLower(strings.ReplaceAll(f, "_", " "))
						if _, exists := lemmas[form]; !exists {
							lemmas[form] = l
						}
					}
				}
			}
			processForms(entry.N)
			processForms(entry.V)
			processForms(entry.A)
			processForms(entry.R)
		}
	}

	// 2. Process synsets for canonical synonyms
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), "entries-") || file.Name() == "frames.json" {
			continue
		}
		path := filepath.Join(dir, file.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		var synsets map[string]Synset
		if err := json.Unmarshal(data, &synsets); err != nil {
			continue
		}

		for _, synset := range synsets {
			if len(synset.Members) > 1 {
				sort.Strings(synset.Members)
				canonical := strings.ToLower(strings.ReplaceAll(synset.Members[0], "_", " "))
				for _, member := range synset.Members {
					m := strings.ToLower(strings.ReplaceAll(member, "_", " "))
					if _, exists := synonyms[m]; !exists {
						synonyms[m] = canonical
					}
				}
			}
		}
	}

	// 3. Final mapping: word -> lemma -> canonical
	finalMap := make(map[string]string)
	
	// Start with all lemmas we found
	for word, lemma := range lemmas {
		target := lemma
		if can, exists := synonyms[lemma]; exists {
			target = can
		}
		finalMap[word] = target
	}
	
	// Add synonyms that might not have been in entries (unlikely but safe)
	for word, can := range synonyms {
		if _, exists := finalMap[word]; !exists {
			finalMap[word] = can
		}
	}

	output, _ := json.MarshalIndent(finalMap, "", "  ")
	ioutil.WriteFile("synonyms.json", output, 0644)
	log.Printf("Generated synonyms.json with %d entries", len(finalMap))
}
