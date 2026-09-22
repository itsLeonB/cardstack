// Package tcgdex ingests Pokémon TCG (SV) catalog data from the public
// TCGDex REST API (https://api.tcgdex.net/v2) into the catalog schema
// (games/expansion_sets/cards/card_variants). See
// .scratch/cardstack-mvp/plans/03-catalog-schema-tcgdex-ingestion.md for
// the shapes this package was built against.
package tcgdex

// seriesResponse is GET /{locale}/series/{seriesID} — a series (e.g. "SV")
// and the sets released under it for that locale.
type seriesResponse struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Sets []seriesSetRef `json:"sets"`
}

type seriesSetRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// setResponse is GET /{locale}/sets/{setID} — one set's brief card list.
type setResponse struct {
	CardCount setCardCount `json:"cardCount"`
	Cards     []setCardRef `json:"cards"`
}

type setCardCount struct {
	Total    int `json:"total"`
	Official int `json:"official"`
}

type setCardRef struct {
	ID      string `json:"id"`
	LocalID string `json:"localId"`
	Name    string `json:"name"`
	Image   string `json:"image"`
}

// cardResponse is GET /{locale}/cards/{cardID} — full card detail. Fields
// other than Name are locale-invariant; only the id-locale response is used
// for them (see ingest.go).
type cardResponse struct {
	ID             string        `json:"id"`
	LocalID        string        `json:"localId"`
	Name           string        `json:"name"`
	Category       string        `json:"category"`
	Illustrator    string        `json:"illustrator"`
	Rarity         string        `json:"rarity"`
	Image          string        `json:"image"`
	Variants       cardVariants  `json:"variants"`
	HP             int           `json:"hp"`
	Types          []string      `json:"types"`
	Stage          string        `json:"stage"`
	Suffix         string        `json:"suffix"`
	Abilities      []cardAbility `json:"abilities"`
	Attacks        []cardAttack  `json:"attacks"`
	Weaknesses     []cardEffect  `json:"weaknesses"`
	Retreat        int           `json:"retreat"`
	RegulationMark string        `json:"regulationMark"`
	DexID          []int         `json:"dexId"`
}

// cardVariants is TCGDex's per-card print-finish flag map — one true flag
// per finish the card was actually printed in.
type cardVariants struct {
	FirstEdition bool `json:"firstEdition"`
	Holo         bool `json:"holo"`
	Normal       bool `json:"normal"`
	Reverse      bool `json:"reverse"`
	WPromo       bool `json:"wPromo"`
}

type cardAbility struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type cardAttack struct {
	Cost   []string `json:"cost"`
	Name   string   `json:"name"`
	Effect string   `json:"effect"`
	// Damage is polymorphic in TCGDex's live data: a bare number (20), a
	// suffixed string ("90+"), or omitted entirely — not always a string
	// as the plan's trimmed example suggested. any + passthrough JSON
	// marshaling into Card.Attributes handles all three without lossy
	// coercion.
	Damage any `json:"damage,omitempty"`
}

type cardEffect struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
