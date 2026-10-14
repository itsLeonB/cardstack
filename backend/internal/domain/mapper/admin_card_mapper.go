package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToAdminCard converts a Card into the admin's view of it. Tags lists every
// tag of the Card, as an empty array when it has none.
func ToAdminCard(host ImageHost, c entity.Card) dto.AdminCard {
	return dto.AdminCard{
		ID:             c.ID,
		ExpansionSetID: c.ExpansionSetID,
		LocalID:        c.LocalID,
		Name:           c.Name,
		Category:       c.Category,
		Tags:           append([]string{}, c.Tags...),
		RarityID:       c.RarityID,
		Illustrator:    c.Illustrator,
		ImageURL:       host.URL(c.ImageKey),
	}
}
