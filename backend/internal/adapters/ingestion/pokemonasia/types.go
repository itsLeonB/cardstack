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
// page. RegulationMark is this page's own section.expansionColumn span.alpha
// field — a format-legality control (see CONTEXT.md's Regulation Mark
// entry / ADR-0010), not a rarity. The card's actual print rarity isn't on
// this page at all; it's resolved separately via the results-list rarity
// filter (see ingest.go's sweepRarities) and passed straight to
// resolveRarityID, so it never needs a field here.
type cardDetail struct {
	Name           string
	Category       string
	Tag            string
	LocalID        string
	Illustrator    string
	RegulationMark string
	Attributes     map[string]any
}
