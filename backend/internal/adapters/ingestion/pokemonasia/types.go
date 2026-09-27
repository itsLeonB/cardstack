package pokemonasia

import "time"

// expansionListing is one <li class="expansion"> entry parsed from
// GET /card-search/?pageNo=N — the single source for Series name, Expansion
// Set code/name, and release date (see the plan's "Site shapes").
type expansionListing struct {
	Series      string
	Code        string
	Name        string
	ReleaseDate time.Time
}

// cardDetail is the parsed content of one GET /card-search/detail/{id}/
// page — the only page that carries a card's actual data. Regulation is not
// scraped from this page; the caller (ingest.go) fills it in from which of
// the 3 regulation-partitioned list passes found the card's id.
type cardDetail struct {
	Name        string
	Category    string
	Tag         string
	RarityCode  string
	LocalID     string
	Illustrator string
	Regulation  string
	Attributes  map[string]any
}
