package tcgdex

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestMapVariants(t *testing.T) {
	tests := []struct {
		name     string
		variants cardVariants
		want     []string // finish codes, in the order mapVariants must return them
	}{
		{
			name:     "single holo variant",
			variants: cardVariants{Holo: true},
			want:     []string{"holo"},
		},
		{
			name: "multi-variant card",
			variants: cardVariants{
				FirstEdition: true,
				Holo:         true,
				Normal:       true,
				Reverse:      true,
				WPromo:       true,
			},
			want: []string{"normal", "reverse", "holo", "first_edition", "w_promo"},
		},
		{
			name:     "no true flags",
			variants: cardVariants{},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapVariants(tt.variants)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapCard(t *testing.T) {
	expansionSetID := uuid.New()

	tests := []struct {
		name  string
		resp  cardResponse
		names map[string]string
		want  entity.Card
	}{
		{
			name: "full detail with all locale names present",
			resp: cardResponse{
				ID:             "SV1V-008",
				LocalID:        "008",
				Name:           "Spidops ex",
				Category:       "Pokemon",
				Illustrator:    "takuyoa",
				Rarity:         "Double rare",
				Image:          "https://assets.tcgdex.net/id/SV/SV1V/008",
				Variants:       cardVariants{Holo: true},
				HP:             260,
				Types:          []string{"Grass"},
				Stage:          "Stage1",
				Suffix:         "EX",
				Abilities:      []cardAbility{{Type: "Ability", Name: "Trap Territory", Effect: "..."}},
				Attacks:        []cardAttack{{Cost: []string{"Grass", "Colorless"}, Name: "Wire Hang", Effect: "...", Damage: "90+"}},
				Weaknesses:     []cardEffect{{Type: "Fire", Value: "×2"}},
				Retreat:        2,
				RegulationMark: "G",
				DexID:          []int{918},
			},
			names: map[string]string{
				"id":    "Spidops ex",
				"ja":    "ワナイダーex",
				"zh-tw": "操陷蛛ex",
				"th":    "วาไนเดอร์ex",
			},
			want: entity.Card{
				ExpansionSetID: expansionSetID,
				LocalID:        "008",
				Names: map[string]any{
					"id":    "Spidops ex",
					"ja":    "ワナイダーex",
					"zh-tw": "操陷蛛ex",
					"th":    "วาไนเดอร์ex",
				},
				Rarity:   "Double rare",
				ImageURL: "https://assets.tcgdex.net/id/SV/SV1V/008",
				Attributes: map[string]any{
					"category":       "Pokemon",
					"illustrator":    "takuyoa",
					"hp":             260,
					"types":          []string{"Grass"},
					"stage":          "Stage1",
					"suffix":         "EX",
					"abilities":      []cardAbility{{Type: "Ability", Name: "Trap Territory", Effect: "..."}},
					"attacks":        []cardAttack{{Cost: []string{"Grass", "Colorless"}, Name: "Wire Hang", Effect: "...", Damage: "90+"}},
					"weaknesses":     []cardEffect{{Type: "Fire", Value: "×2"}},
					"retreat":        2,
					"regulationMark": "G",
					"dexId":          []int{918},
				},
			},
		},
		{
			name: "card missing some locale names (en legitimately 404s)",
			resp: cardResponse{
				LocalID:  "001",
				Category: "Pokemon",
				Rarity:   "Common",
				Image:    "https://assets.tcgdex.net/id/SV/SV1V/001",
				Variants: cardVariants{Normal: true},
			},
			names: map[string]string{
				"id": "Pineco",
				"ja": "",
				"th": "",
			},
			want: entity.Card{
				ExpansionSetID: expansionSetID,
				LocalID:        "001",
				Names: map[string]any{
					"id": "Pineco",
				},
				Rarity:   "Common",
				ImageURL: "https://assets.tcgdex.net/id/SV/SV1V/001",
				Attributes: map[string]any{
					"category": "Pokemon",
				},
			},
		},
		{
			name: "trainer card with no Pokémon-specific attributes",
			resp: cardResponse{
				LocalID:  "150",
				Category: "Trainer",
				Rarity:   "Uncommon",
				Image:    "https://assets.tcgdex.net/id/SV/SV1V/150",
				Variants: cardVariants{Normal: true, Reverse: true},
			},
			names: map[string]string{"id": "Ultra Ball"},
			want: entity.Card{
				ExpansionSetID: expansionSetID,
				LocalID:        "150",
				Names:          map[string]any{"id": "Ultra Ball"},
				Rarity:         "Uncommon",
				ImageURL:       "https://assets.tcgdex.net/id/SV/SV1V/150",
				Attributes:     map[string]any{"category": "Trainer"},
			},
		},
	}

	// raw is passed through to every case below unchanged — mapCard doesn't
	// derive it from resp, it just threads it into Card.Raw. What raw
	// actually preserves (including fields resp doesn't model) is covered
	// by TestMapCard_RawIsExactResponseBytes.
	raw := json.RawMessage(`{"stub":true}`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Raw = string(raw)

			got := mapCard(tt.resp, expansionSetID, tt.names, raw)

			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMapCard_RawIsExactResponseBytes confirms Raw stores the exact bytes
// mapCard was given, not a remarshal of the decoded cardResponse — a
// remarshal would silently drop futureField below (cardResponse has no
// field for it), defeating Raw's purpose of future-proofing against
// exactly that.
func TestMapCard_RawIsExactResponseBytes(t *testing.T) {
	raw := json.RawMessage(`{"id":"SV1V-008","localId":"008","rarity":"Double rare","futureField":{"nested":true}}`)
	resp := cardResponse{ID: "SV1V-008", LocalID: "008", Rarity: "Double rare"}

	got := mapCard(resp, uuid.New(), map[string]string{"id": "Spidops ex"}, raw)

	assert.Equal(t, string(raw), got.Raw, "raw must be stored byte-for-byte, not reformatted")
}
