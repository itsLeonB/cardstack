package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToAdminCard converts a Card into the admin's view of it. Tag is the Card's
// first tag, if any.
func ToAdminCard(host ImageHost, c entity.Card) dto.AdminCard {
	tag := ""
	if len(c.Tags) > 0 {
		tag = c.Tags[0]
	}
	return dto.AdminCard{
		ID:             c.ID,
		ExpansionSetID: c.ExpansionSetID,
		LocalID:        c.LocalID,
		Name:           c.Name,
		Category:       c.Category,
		Tag:            tag,
		RarityID:       c.RarityID,
		Illustrator:    c.Illustrator,
		ImageURL:       host.URL(c.ImageKey),
	}
}
