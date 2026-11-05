package dto

import "github.com/google/uuid"

// ScanFilter narrows a scan lookup to Expansion Sets and Rarities. An empty
// slice means no restriction on that field; unpublished sets stay excluded.
type ScanFilter struct {
	ExpansionSetIDs []uuid.UUID
	RarityIDs       []uuid.UUID
}

// MatchRequest is one uploaded card photo to match against the catalog,
// optionally limited to a Scan Filter.
type MatchRequest struct {
	ScanFilter
	Image []byte
}

// MatchCandidate is one Card the matcher considers a possible match, with how
// strongly (Score, 0 to 1, higher is closer).
type MatchCandidate struct {
	Score float64     `json:"score" minimum:"0" maximum:"1" doc:"How closely the photo matches this Card; candidates are ordered by it, highest first."`
	Card  CardSummary `json:"card"`
}

// MatchResult is POST /scan/match's response. When Confident is true,
// Candidates holds exactly the matched Card, so a client may add it without
// asking; otherwise it holds the Cards to choose from, and none may be
// picked automatically.
type MatchResult struct {
	Empty      bool             `json:"empty" doc:"True when the photo shows no card: its top Card scored below the server's empty threshold. Candidates is then empty and Confident false."`
	Confident  bool             `json:"confident" doc:"True only when the server is sure of the first candidate. The server owns the threshold."`
	Candidates []MatchCandidate `json:"candidates"`
}

// ReprintRequest asks for the Reprints of the Cards the client holds: up to
// five ids in the order the matcher offered them, and the page to start at.
type ReprintRequest struct {
	ScanFilter
	CardIDs []uuid.UUID
	Offset  int
}

// ReprintResult is GET /scan/reprints' response: one page of Reprints, and
// whether more remain past it.
type ReprintResult struct {
	Cards   []CardSummary `json:"cards" doc:"Up to five Reprints, in the order they are offered."`
	HasMore bool          `json:"hasMore" doc:"True while more Reprints remain after this page."`
}
