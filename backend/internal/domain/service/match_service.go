package service

import (
	"context"
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ungerr"
)

const (
	invalidImageMsg = "upload a JPEG image"
	jpegMIME        = "image/jpeg"
)

// MatchCandidates is how many Cards a match that is not confident offers to
// choose from.
const MatchCandidates = 5

// reprintPageSize is how many Reprints one page holds.
const reprintPageSize = 5

// MatchSettings is what the matcher needs from configuration: the embedding
// model whose stored vectors it searches (rows of any other model are never
// compared), and the rule for Confident. Threshold is the lowest similarity the
// top candidate may have; Margin is how far the runner-up must trail it.
// EmptyThreshold is the similarity below which the top candidate means no card
// is in the photo at all; zero disables that rule.
//
// ReprintDelta is the largest cosine distance at which two catalog images count
// as the same artwork for a Reprint.
type MatchSettings struct {
	Model          string
	Threshold      float64
	Margin         float64
	EmptyThreshold float64
	ReprintDelta   float64
}

// MatchService matches an uploaded card photo to catalog Cards, and finds the
// Reprints of Cards the client already holds.
type MatchService interface {
	// Match validates the upload and returns ranked candidates. It writes
	// nothing and never keeps the image.
	Match(ctx context.Context, req dto.MatchRequest) (dto.MatchResult, error)
	// Reprints returns one page of the Reprints of req's Cards (docs/adr/0023).
	Reprints(ctx context.Context, req dto.ReprintRequest) (dto.ReprintResult, error)
}

type matchService struct {
	embedder   embedding.ImageEmbedder
	embeddings repository.EmbeddingRepository
	cards      repository.MatchRepository
	images     mapper.ImageHost
	settings   MatchSettings
}

// NewMatchService builds the MatchService that embeds the photo with embedder
// and searches the stored embeddings for the nearest Cards.
func NewMatchService(
	embedder embedding.ImageEmbedder,
	embeddings repository.EmbeddingRepository,
	cards repository.MatchRepository,
	images mapper.ImageHost,
	settings MatchSettings,
) MatchService {
	return &matchService{embedder: embedder, embeddings: embeddings, cards: cards, images: images, settings: settings}
}

func (s *matchService) Match(ctx context.Context, req dto.MatchRequest) (dto.MatchResult, error) {
	if !isAcceptedImage(req.Image) {
		return dto.MatchResult{}, ungerr.BadRequestError(invalidImageMsg)
	}

	vector, err := s.embedder.Embed(ctx, jpegMIME, req.Image)
	if err != nil {
		return dto.MatchResult{}, err
	}

	nearest, err := s.embeddings.NearestCards(ctx, s.settings.Model, vector, MatchCandidates)
	if err != nil {
		return dto.MatchResult{}, err
	}
	scores := ScoreNearest(nearest)

	ids := make([]uuid.UUID, len(scores))
	for i, n := range scores {
		ids[i] = n.CardID
	}
	rows, err := s.cards.CardsByIDs(ctx, ids)
	if err != nil {
		return dto.MatchResult{}, err
	}
	byID := make(map[uuid.UUID]repository.CardResult, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	candidates := make([]dto.MatchCandidate, 0, len(scores))
	for _, n := range scores {
		if row, ok := byID[n.CardID]; ok {
			candidates = append(candidates, mapper.ToMatchCandidate(s.images, row, n.Similarity))
		}
	}

	if s.settings.empty(candidates) {
		return dto.MatchResult{Empty: true, Candidates: candidates[:0]}, nil
	}
	confident := s.settings.confident(candidates)
	if confident {
		candidates = candidates[:1]
	}
	return dto.MatchResult{Confident: confident, Candidates: candidates}, nil
}

// confident is true when the top candidate clears the threshold and the
// runner-up, if any, trails it by at least the margin. Candidates are ordered
// best first. A near-tie fails the margin, so the same artwork reprinted in
// two Expansion Sets is never picked automatically.
func (m MatchSettings) confident(candidates []dto.MatchCandidate) bool {
	if len(candidates) == 0 || candidates[0].Score < m.Threshold {
		return false
	}
	return len(candidates) == 1 || candidates[0].Score-candidates[1].Score >= m.Margin
}

// empty reports whether the top candidate scores below the empty threshold, so
// the photo shows no card. A threshold of 0 turns the rule off, and a result
// with no candidates is not empty: there is no top score to judge.
func (m MatchSettings) empty(candidates []dto.MatchCandidate) bool {
	return m.EmptyThreshold > 0 && len(candidates) > 0 && candidates[0].Score < m.EmptyThreshold
}

// bestRowPerCard keeps the nearest row of each Card, since a Card may hold
// several embeddings. The rows arrive nearest first. The search asks for
// MatchCandidates rows, so a Card with more than one row can leave fewer
// candidates; every Card has one row today.
func bestRowPerCard(nearest []repository.CardDistance) []repository.CardDistance {
	seen := make(map[uuid.UUID]bool, len(nearest))
	best := make([]repository.CardDistance, 0, len(nearest))
	for _, n := range nearest {
		if !seen[n.CardID] {
			seen[n.CardID] = true
			best = append(best, n)
		}
	}
	return best
}

// CardScore is one Card of a query's nearest embeddings with the similarity
// the matcher reports for it.
type CardScore struct {
	CardID     uuid.UUID
	Similarity float64
}

// ScoreNearest is the matcher's scoring of a query's nearest embeddings: one
// row per Card, nearest first, each with the similarity Match reports. The
// tune-scan report scores through it as well, so its numbers are the matcher's.
func ScoreNearest(nearest []repository.CardDistance) []CardScore {
	best := bestRowPerCard(nearest)
	scores := make([]CardScore, len(best))
	for i, n := range best {
		scores[i] = CardScore{CardID: n.CardID, Similarity: similarity(n.Distance)}
	}
	return scores
}

// similarity turns a cosine distance (0 to 2) into the 0 to 1 score the
// response promises, where higher is closer. Opposite vectors score 0.
func similarity(distance float64) float64 {
	return math.Min(1, math.Max(0, 1-distance))
}

// isAcceptedImage sniffs the bytes rather than trusting the declared type.
// Reprints returns one page of the Reprints of req's Cards. Each source, in the
// order given, contributes its Reprints nearest first, and a Card reachable from
// two sources keeps its first place. The list is rebuilt on every call from the
// same ids, so an offset is stable without any server state.
func (s *matchService) Reprints(ctx context.Context, req dto.ReprintRequest) (dto.ReprintResult, error) {
	var ordered []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	for _, source := range req.CardIDs {
		found, err := s.cards.ReprintsOf(ctx, s.settings.Model, source, req.CardIDs, s.settings.ReprintDelta)
		if err != nil {
			return dto.ReprintResult{}, err
		}
		for _, r := range found {
			if !seen[r.CardID] {
				seen[r.CardID] = true
				ordered = append(ordered, r.CardID)
			}
		}
	}

	start := min(max(req.Offset, 0), len(ordered))
	end := min(start+reprintPageSize, len(ordered))
	page := ordered[start:end]
	rows, err := s.cards.CardsByIDs(ctx, page)
	if err != nil {
		return dto.ReprintResult{}, err
	}
	byID := make(map[uuid.UUID]repository.CardResult, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	cards := make([]dto.CardSummary, 0, len(page))
	for _, id := range page {
		if row, ok := byID[id]; ok {
			cards = append(cards, mapper.ToCardSummary(s.images, row))
		}
	}
	return dto.ReprintResult{Cards: cards, HasMore: end < len(ordered)}, nil
}

func isAcceptedImage(data []byte) bool {
	return http.DetectContentType(data) == jpegMIME
}
