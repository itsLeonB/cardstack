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

const invalidImageMsg = "upload a JPEG, PNG or WebP image"

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
	// unsure (three to five candidates); the real matcher replaces this.
	confident := rand.IntN(2) == 0
	count := 1
	if !confident {
		count = 3 + rand.IntN(3)
	}

	rows, err := s.repo.RandomCards(ctx, count)
	if err != nil {
		return dto.MatchResult{}, err
	}

	score := 0.9 + rand.Float64()*0.09
	if !confident {
		score = 0.5 + rand.Float64()*0.3
	}
	candidates := make([]dto.MatchCandidate, len(rows))
	for i, row := range rows {
		candidates[i] = mapper.ToMatchCandidate(s.images, row, score)
		score -= 0.03 + rand.Float64()*0.07
	}

	return dto.MatchResult{Confident: confident, Candidates: candidates}, nil
}

// isAcceptedImage sniffs the bytes rather than trusting the declared type.
func isAcceptedImage(data []byte) bool {
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}
