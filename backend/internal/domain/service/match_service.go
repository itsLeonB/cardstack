package service

import (
	"context"
	"math/rand/v2"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/ungerr"
)

const invalidImageMsg = "upload a JPEG image"

// Stub score shaping. A confident match scores confidentFloor plus up to
// confidentRange; an unsure list starts at unsureFloor plus up to unsureRange
// and each next candidate drops by stepFloor plus up to stepRange.
const (
	confidentFloor = 0.9
	confidentRange = 0.09
	unsureFloor    = 0.5
	unsureRange    = 0.3
	stepFloor      = 0.03
	stepRange      = 0.07
)

// MatchService matches an uploaded card photo to catalog Cards. Its current
// implementation is a stub that returns random Cards; the real matcher
// replaces it behind this same interface and response shape.
type MatchService interface {
	// Match validates the upload and returns ranked candidates. It writes
	// nothing and never keeps the image.
	Match(ctx context.Context, req dto.MatchRequest) (dto.MatchResult, error)
}

type matchService struct {
	repo   repository.MatchRepository
	images mapper.ImageHost
}

// NewMatchService builds the stub MatchService.
func NewMatchService(repo repository.MatchRepository, images mapper.ImageHost) MatchService {
	return &matchService{repo: repo, images: images}
}

func (s *matchService) Match(ctx context.Context, req dto.MatchRequest) (dto.MatchResult, error) {
	if !isAcceptedImage(req.Image) {
		return dto.MatchResult{}, ungerr.BadRequestError(invalidImageMsg)
	}

	// ponytail: stub. Half the calls are confident (one match), the rest are
	// unsure (three to five candidates). Replaced by the real matcher in
	// ticket 06; the interface and response shape stay.
	confident := rand.IntN(2) == 0
	count := 1
	if !confident {
		count = 3 + rand.IntN(3)
	}

	rows, err := s.repo.RandomCards(ctx, count)
	if err != nil {
		return dto.MatchResult{}, err
	}

	score := confidentFloor + rand.Float64()*confidentRange
	if !confident {
		score = unsureFloor + rand.Float64()*unsureRange
	}
	candidates := make([]dto.MatchCandidate, len(rows))
	for i, row := range rows {
		candidates[i] = mapper.ToMatchCandidate(s.images, row, score)
		score -= stepFloor + rand.Float64()*stepRange
	}

	return dto.MatchResult{Confident: confident, Candidates: candidates}, nil
}

// isAcceptedImage sniffs the bytes rather than trusting the declared type.
func isAcceptedImage(data []byte) bool {
	return http.DetectContentType(data) == "image/jpeg"
}
