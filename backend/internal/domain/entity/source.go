package entity

// Sources name the writer that created a Card or Expansion Set, or last
// updated it (docs/adr/0021). They are writers, not users.
const (
	SourcePokemonAsia  = "pokemon-asia"
	SourceTCGCollector = "tcgcollector"
	SourceAdmin        = "admin"
)
