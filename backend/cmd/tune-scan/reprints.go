package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

// maxPairwiseCards caps the pairwise distances of one name group. A group of n
// Cards has n(n-1)/2 pairs, so a very large one (a basic energy in dozens of
// sets) would bury the rest of the report.
const maxPairwiseCards = 8

// pairKey is an unordered pair of Cards.
type pairKey [2]uuid.UUID

func unordered(a, b uuid.UUID) pairKey {
	if bytes.Compare(a[:], b[:]) > 0 {
		a, b = b, a
	}
	return pairKey{a, b}
}

func runReprints(ctx context.Context, args []string, e env) int {
	fs := flag.NewFlagSet("reprints", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.Usage = func() {
		printf(e.stderr, "usage: tune-scan reprints [--limit N] [--min-size K]\n")
		fs.PrintDefaults()
	}
	limit := fs.Int("limit", 20, "print at most this many name groups, largest first")
	minSize := fs.Int("min-size", 2, "only groups of at least this many embedded Cards (2 or more)")
	if exit, ok := parseFlags(fs, args, e); !ok {
		return exit
	}
	if *limit < 1 {
		return usageFail(e, fs, "--limit must be at least 1")
	}
	if *minSize < 2 {
		return usageFail(e, fs, "--min-size must be at least 2")
	}

	d, cleanup, err := e.open()
	if err != nil {
		return fail(e.stderr, err)
	}
	defer cleanup()
	if err := requireEmbeddings(ctx, d); err != nil {
		return fail(e.stderr, err)
	}

	groups, err := d.tuning.NameGroups(ctx, d.model)
	if err != nil {
		return fail(e.stderr, err)
	}
	var eligible []repository.NameGroup
	for _, g := range groups {
		if g.Size >= *minSize {
			eligible = append(eligible, g)
		}
	}
	shown := min(*limit, len(eligible))

	rep := &report{w: e.stdout}
	rep.printf("MATCH_REPRINT_DELTA %.3f: a pair at or under this distance is the same artwork\n", d.reprintDelta)
	rep.printf("%d of %d name group(s) with at least %d embedded Cards, largest first\n", shown, len(eligible), *minSize)
	for i, g := range eligible[:shown] {
		logger.Infof("[%d/%d] %s: %d Cards", i+1, shown, g.Name, g.Size)
		cards, err := d.tuning.GroupCards(ctx, d.model, g.Name)
		if err != nil {
			return fail(e.stderr, err)
		}
		if err := writeGroup(ctx, rep, d, g, cards); err != nil {
			return fail(e.stderr, err)
		}
	}
	if rep.err != nil {
		return fail(e.stderr, rep.err)
	}
	return 0
}

// writeGroup prints one name group's Cards pairwise: each pair's distance and
// whether it is within MATCH_REPRINT_DELTA.
func writeGroup(ctx context.Context, r *report, d deps, g repository.NameGroup, cards []repository.GroupCard) error {
	r.printf("\n%s (%d Cards)\n", g.Name, len(cards))

	paired := cards
	if len(paired) > maxPairwiseCards {
		paired = paired[:maxPairwiseCards]
	}
	ids := make([]uuid.UUID, len(paired))
	for i, c := range paired {
		ids[i] = c.ID
	}
	pairs, err := d.tuning.PairDistances(ctx, d.model, ids)
	if err != nil {
		return err
	}
	distances := make(map[pairKey]float64, len(pairs))
	for _, p := range pairs {
		distances[unordered(p.A, p.B)] = p.Distance
	}

	for i := range paired {
		for j := i + 1; j < len(paired); j++ {
			a, b := paired[i], paired[j]
			distance, ok := distances[unordered(a.ID, b.ID)]
			if !ok {
				return fmt.Errorf("no distance between %s %s and %s %s", a.SetCode, a.LocalID, b.SetCode, b.LocalID)
			}
			verdict := "beyond delta"
			if distance <= d.reprintDelta {
				verdict = "within delta"
			}
			r.printf("  %s %s  %s %s  %.3f  %s\n", a.SetCode, a.LocalID, b.SetCode, b.LocalID, distance, verdict)
		}
	}

	if len(cards) > len(paired) {
		r.printf("  pairwise shown for the first %d of %d Cards (the cap is %d)\n", len(paired), len(cards), maxPairwiseCards)
	}
	return nil
}
