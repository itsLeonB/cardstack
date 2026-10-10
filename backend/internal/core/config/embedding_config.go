package config

// Gemini configures the Gemini API that cmd/embed-catalog and
// cmd/collect-embeddings call. Only those commands read it, so the API boots
// without it.
type Gemini struct {
	APIKey string `envconfig:"API_KEY"`
}

func (Gemini) Prefix() string { return "GEMINI" }

// Embedding names the model whose embeddings are stored (GEMINI_* holds the
// key). Changing Model makes every card eligible for re-embedding.
type Embedding struct {
	Model string `default:"gemini-embedding-2"`
}

func (Embedding) Prefix() string { return "EMBEDDING" }
