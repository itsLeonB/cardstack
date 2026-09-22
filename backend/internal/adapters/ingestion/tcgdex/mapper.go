package tcgdex

import (
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/datatypes"
)

// mapCard converts a card's full detail response plus a locale->name map
// into the entity.Card row to upsert. It is pure: no I/O, no DB. ID and the
// BaseEntity timestamps are left zero for the caller's find-or-create to
// fill in.
func mapCard(resp cardResponse, expansionSetID uuid.UUID, names map[string]string) entity.Card {
	return entity.Card{
		ExpansionSetID: expansionSetID,
		LocalID:        resp.LocalID,
		Names:          mapNames(names),
		Rarity:         resp.Rarity,
		ImageURL:       resp.Image,
		Attributes:     mapAttributes(resp),
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

// The five finish values the catalog_schema migration's CHECK constraint
// allows (see the migration for the source of truth). Named here so the
// literal isn't repeated at each mapVariants call site.
const (
	finishNormal       = "normal"
	finishReverse      = "reverse"
	finishHolo         = "holo"
	finishFirstEdition = "first_edition"
	finishWPromo       = "w_promo"
)

// mapVariants converts TCGDex's per-card variants flag map into one
// entity.CardVariant per true flag, in a fixed order. CardID is left zero
// for the caller to fill in once the parent Card row's ID is known.
func mapVariants(v cardVariants) []entity.CardVariant {
	var variants []entity.CardVariant
	if v.Normal {
		variants = append(variants, entity.CardVariant{Finish: finishNormal})
	}
	if v.Reverse {
		variants = append(variants, entity.CardVariant{Finish: finishReverse})
	}
	if v.Holo {
		variants = append(variants, entity.CardVariant{Finish: finishHolo})
	}
	if v.FirstEdition {
		variants = append(variants, entity.CardVariant{Finish: finishFirstEdition})
	}
	if v.WPromo {
		variants = append(variants, entity.CardVariant{Finish: finishWPromo})
	}
	return variants
}
