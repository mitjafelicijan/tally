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
	dataDirectory := "english-wordnet-2025-json"
	files, err := ioutil.ReadDir(dataDirectory)
	if err != nil {
		log.Fatal(err)
	}

	lemmas := make(map[string]string)
	synonyms := make(map[string]string)

	// Process entries for lemmas and forms
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "entries-") {
			continue
		}
		path := filepath.Join(dataDirectory, file.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		var entries map[string]Entry
		json.Unmarshal(data, &entries)

		for lemmaKey, entry := range entries {
			lemma := strings.ToLower(strings.ReplaceAll(lemmaKey, "_", " "))
			lemmas[lemma] = lemma
			
			// Collect forms
			processForms := func(sense *Sense) {
				if sense != nil {
					for _, formRaw := range sense.Form {
						form := strings.ToLower(strings.ReplaceAll(formRaw, "_", " "))
						if _, exists := lemmas[form]; !exists {
							lemmas[form] = lemma
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

	// Process synsets for canonical synonyms
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), "entries-") || file.Name() == "frames.json" {
			continue
		}
		path := filepath.Join(dataDirectory, file.Name())
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
				for _, memberRaw := range synset.Members {
					member := strings.ToLower(strings.ReplaceAll(memberRaw, "_", " "))
					if _, exists := synonyms[member]; !exists {
						synonyms[member] = canonical
					}
				}
			}
		}
	}

	// Final mapping: word -> lemma -> canonical
	finalMap := make(map[string]string)
	
	// Start with all lemmas we found
	for word, lemma := range lemmas {
		target := lemma
		if canonical, exists := synonyms[lemma]; exists {
			target = canonical
		}
		finalMap[word] = target
	}
	
	// Add synonyms that might not have been in entries (unlikely but safe)
	for word, canonical := range synonyms {
		if _, exists := finalMap[word]; !exists {
			finalMap[word] = canonical
		}
	}

	output, _ := json.MarshalIndent(finalMap, "", "  ")
	ioutil.WriteFile("synonyms.json", output, 0644)
	log.Printf("Generated synonyms.json with %d entries", len(finalMap))
}
