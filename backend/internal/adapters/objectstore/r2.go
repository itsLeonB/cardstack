package objectstore

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/ungerr"
)

// immutableCacheControl lets Cloudflare and browsers keep an original for a
// year: a key always maps to the same bytes because it derives from the row id.
const immutableCacheControl = "public, max-age=31536000, immutable"

// R2Store is the Cloudflare R2 adapter, speaking R2's S3-compatible API.
type R2Store struct {
	client *s3.Client
	bucket string
}

var _ corestore.ObjectStore = (*R2Store)(nil)

// NewR2Store builds an R2Store from cfg, which the caller has checked with
// cfg.Configured(). cfg.Endpoint, when set, replaces the address derived from
// the account id.
func NewR2Store(cfg config.R2) *R2Store {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	}
	client := s3.New(s3.Options{
		Region:       "auto", // required by the SDK, ignored by R2
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
		// R2 does not accept the SDK's default trailing checksums.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &R2Store{client: client, bucket: cfg.Bucket}
}

func (s *R2Store) Put(ctx context.Context, key, contentType string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(body),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(immutableCacheControl),
	})
	if err != nil {
		return ungerr.Wrapf(err, "putting object %q", key)
	}
	return nil
}
