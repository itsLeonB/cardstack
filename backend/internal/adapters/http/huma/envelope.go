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

// EnvelopeWithMeta wraps a successful, paginated response body in top-level
// "data" and "meta" fields, e.g. {"data": [...], "meta": {...}}. It's the
// paginated counterpart to Envelope, for routes (e.g. GET /catalog/cards)
// whose pagination bookkeeping (total/page/limit) travels alongside the data
// rather than nested inside it.
type EnvelopeWithMeta[T, M any] struct {
	Data T `json:"data"`
	Meta M `json:"meta"`
}

// NewEnvelopeWithMeta wraps the given data and meta in an EnvelopeWithMeta.
func NewEnvelopeWithMeta[T, M any](data T, meta M) EnvelopeWithMeta[T, M] {
	return EnvelopeWithMeta[T, M]{Data: data, Meta: meta}
}
