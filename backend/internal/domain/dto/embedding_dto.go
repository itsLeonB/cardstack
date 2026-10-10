package dto

import "time"

// SubmitCatalogRequest scopes one submit run to the Expansion Set with Set as
// its code, or to every card when Set is empty.
type SubmitCatalogRequest struct {
	Set string
}

// SubmitCatalogSummary reports one submit run. Total, NoImage, AlreadyEmbedded
// and InFlight describe the scope before the run. Submitted and Batches count
// this run's work, and Failed counts the cards that could not be submitted.
type SubmitCatalogSummary struct {
	Total           int64
	NoImage         int64
	AlreadyEmbedded int64
	InFlight        int64
	Submitted       int
	Batches         int
	Failed          int
	Elapsed         time.Duration
}

// CollectBatchesSummary reports one collect run over the submitted batches.
// Collected batches had their vectors stored, Running ones are left for a later
// run, and FailedBatches failed at the provider, which makes their cards
// pending again. Embedded counts the vectors stored. Failed counts the cards
// that did not embed and the batches whose check errored; either makes the
// command exit non-zero.
type CollectBatchesSummary struct {
	Batches       int
	Collected     int
	Running       int
	FailedBatches int
	Embedded      int
	Failed        int
	Elapsed       time.Duration
}
