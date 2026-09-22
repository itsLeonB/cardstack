package tcgdex

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/datatypes"
)

// mapCard converts a card's full detail response plus a locale->name map
// into the entity.Card row to upsert. It is pure: no I/O, no DB. ID and the
// BaseEntity timestamps are left zero for the caller's find-or-create to
// fill in. raw is the exact response body as received over the wire (see
// client.getCard) and is stored verbatim into Raw — not a remarshal of the
// already-decoded resp, which would silently drop any field cardResponse
// doesn't model (or omit a field carrying `omitempty` even when TCGDex sent
// it), defeating Raw's whole purpose of future-proofing against exactly
// that.
func mapCard(resp cardResponse, expansionSetID uuid.UUID, names map[string]string, raw json.RawMessage) entity.Card {
	return entity.Card{
		ExpansionSetID: expansionSetID,
		LocalID:        resp.LocalID,
		Names:          mapNames(names),
		Rarity:         resp.Rarity,
		ImageURL:       resp.Image,
		Attributes:     mapAttributes(resp),
		Raw:            datatypes.JSON(raw),
	}
}

// mapNames builds the Names JSONB map from a locale->name lookup, dropping
// empty entries (a locale that legitimately has no data for this card).
func mapNames(names map[string]string) datatypes.JSONMap {
	m := make(datatypes.JSONMap, len(names))
	for locale, name := range names {
		if name == "" {
			continue
		}
		m[locale] = name
	}
	return m
}

// mapAttributes builds the game-specific Attributes JSONB map (docs/adr/0001)
// from a card's full detail response, omitting fields TCGDex left zero/empty
// (e.g. non-Pokémon categories carry no HP/types/attacks).
func mapAttributes(resp cardResponse) datatypes.JSONMap {
	m := datatypes.JSONMap{}
	if resp.Category != "" {
		m["category"] = resp.Category
	}
	if resp.Illustrator != "" {
		m["illustrator"] = resp.Illustrator
	}
	if resp.HP != 0 {
		m["hp"] = resp.HP
	}
	if len(resp.Types) > 0 {
		m["types"] = resp.Types
	}
	if resp.Stage != "" {
		m["stage"] = resp.Stage
	}
	if resp.Suffix != "" {
		m["suffix"] = resp.Suffix
	}
	if len(resp.Abilities) > 0 {
		m["abilities"] = resp.Abilities
	}
	if len(resp.Attacks) > 0 {
		m["attacks"] = resp.Attacks
	}
	if len(resp.Weaknesses) > 0 {
		m["weaknesses"] = resp.Weaknesses
	}
	if resp.Retreat != 0 {
		m["retreat"] = resp.Retreat
	}
	if resp.RegulationMark != "" {
		m["regulationMark"] = resp.RegulationMark
	}
	if len(resp.DexID) > 0 {
		m["dexId"] = resp.DexID
	}
	return m
}

// The five finish codes TCGDex's cardVariants flag map can report. Named
// here so the literal isn't repeated at each mapVariants call site; also
// used as the finishes lookup table's row codes (see ingest.go's
// resolveFinishID).
const (
	finishNormal       = "normal"
	finishReverse      = "reverse"
	finishHolo         = "holo"
	finishFirstEdition = "first_edition"
	finishWPromo       = "w_promo"
)

// mapVariants converts TCGDex's per-card variants flag map into the finish
// codes present, in a fixed order, for the caller to resolve into a
// finishes row (and CardVariant) once each finish's ID is known.
func mapVariants(v cardVariants) []string {
	var codes []string
	if v.Normal {
		codes = append(codes, finishNormal)
	}
	if v.Reverse {
		codes = append(codes, finishReverse)
	}
	if v.Holo {
		codes = append(codes, finishHolo)
	}
	if v.FirstEdition {
		codes = append(codes, finishFirstEdition)
	}
	if v.WPromo {
		codes = append(codes, finishWPromo)
	}
	return codes
}
