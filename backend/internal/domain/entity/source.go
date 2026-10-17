package entity

// Sources name the writer that created a Card or Expansion Set, or last
// updated it (docs/adr/0021). They are writers, not users. TCG Collector's
// constant arrives with its ingester.
const (
	SourcePokemonAsia = "pokemon-asia"
	SourceAdmin       = "admin"
)
