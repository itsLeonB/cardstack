package dto

// MatchRequest is one uploaded card photo to match against the catalog.
type MatchRequest struct {
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
	Confident  bool             `json:"confident" doc:"True only when the server is sure of the first candidate. The server owns the threshold."`
	Candidates []MatchCandidate `json:"candidates"`
}
