package config

import "fmt"

// Match sets when the scan matcher calls its top candidate confident. The
// values are tuned on real scores (ticket 07), so they are settings: changing
// them needs a restart of the API, not a frontend release.
//
// Threshold is the lowest cosine similarity (0 to 1, higher is closer) the top
// candidate may have. Margin is how far the runner-up must trail it. A near-tie,
// such as one artwork reprinted in two Expansion Sets, fails the margin and so is
// never auto-picked.
//
// ReprintDelta is the largest cosine distance between two catalog images that
// still counts them as the same artwork for a Reprint (docs/adr/0023).
type Match struct {
	Threshold    float64 `default:"0.85"`
	Margin       float64 `default:"0.03"`
	ReprintDelta float64 `envconfig:"REPRINT_DELTA" default:"0.05"`
}

func (Match) Prefix() string { return "MATCH" }

// ValidateMatch refuses a threshold outside (0, 1], a margin outside [0, 1) and
// a Reprint delta outside [0, 1].
// A threshold of 0 would make any top candidate confident, and a threshold above
// 1 would make none confident; a typo in either fails the boot rather than
// silently changing which scans are added without asking.
func (m Match) ValidateMatch() error {
	if m.Threshold <= 0 || m.Threshold > 1 {
		return fmt.Errorf("MATCH_THRESHOLD must be greater than 0 and at most 1, got %v", m.Threshold)
	}
	if m.Margin < 0 || m.Margin >= 1 {
		return fmt.Errorf("MATCH_MARGIN must be at least 0 and less than 1, got %v", m.Margin)
	}
	if m.ReprintDelta < 0 || m.ReprintDelta > 1 {
		return fmt.Errorf("MATCH_REPRINT_DELTA must be at least 0 and at most 1, got %v", m.ReprintDelta)
	}
	return nil
}
