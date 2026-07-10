# /bin/news

A minimalist, deduplicated news aggregator for open source and Linux.

## Features
- **Deduplication:** Uses Open English WordNet synonyms and Jaccard similarity to filter overlapping stories.
- **Time Window:** Only includes stories from the last 32 hours.
- **Static Generation:** Generates a purely static site with daily archives, a monthly calendar, and source links.
- **Autonomous:** Fully automated via GitHub Actions every 12 hours and deployed to GitHub Pages.

## How it works
1. Fetches feeds from `feeds.json`.
2. Normalizes and tokenizes titles using WordNet synonyms.
3. Compares story similarity using a 50% Jaccard threshold.
4. Saves current data to `public/YYYY-MM-DD.json`.
5. Regenerates the index and all historical HTML views in `public/`.

## Usage
### Local Execution
```bash
go run .
```

### WordNet Pre-processing
To rebuild `synonyms.json` from raw WordNet JSON data:
```bash
go run scripts/gen_synonyms.go
```

## Project Structure
- `feeds.json`: Source list of RSS/Atom feed URLs.
- `template.html`: The HTML layout for all generated pages.
- `public/`: The generated static site and raw daily JSON data.
- `scripts/`: Maintenance utilities.
