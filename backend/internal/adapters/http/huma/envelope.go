package httpapi

// Envelope wraps a successful response body in a top-level "data" field,
// e.g. {"data": {...}}. It is used as the Body type of Huma Output structs
// for successful responses only; error responses continue to use Huma's own
// ErrorModel/huma.WriteErr and are never wrapped in an Envelope.
type Envelope[T any] struct {
	Data T `json:"data"`
}

// NewEnvelope wraps the given value in an Envelope.
func NewEnvelope[T any](data T) Envelope[T] {
	return Envelope[T]{Data: data}
}
