package config

// Gemini configures the Gemini API that cmd/embed-catalog,
// cmd/collect-embeddings and the API's scan match call. The API boots without
// a key (a preview or CI run has none) and logs an error, and every scan match
// then fails.
type Gemini struct {
	APIKey string `envconfig:"API_KEY"`
}

func (Gemini) Prefix() string { return "GEMINI" }

// Embedding names the model whose embeddings are stored and searched (GEMINI_*
// holds the key). Changing Model makes every card eligible for re-embedding,
// and a scan only matches cards embedded by the model it names.
type Embedding struct {
	Model string `default:"gemini-embedding-2"`
}

func (Embedding) Prefix() string { return "EMBEDDING" }
