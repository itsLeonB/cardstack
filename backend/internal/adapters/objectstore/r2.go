package objectstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/ungerr"
)

// immutableCacheControl lets Cloudflare and browsers keep an original for a
// year: a key always maps to the same bytes because it derives from the row id.
const immutableCacheControl = "public, max-age=31536000, immutable"

// maxGetBytes caps what Get reads into memory: a hosted original is a few
// hundred KB, so anything near this is not an image we put there.
const maxGetBytes = 32 << 20

// R2Store is the Cloudflare R2 adapter, speaking R2's S3-compatible API.
type R2Store struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
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
	return &R2Store{client: client, presigner: s3.NewPresignClient(client), bucket: cfg.Bucket}
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

// PresignPut signs a PUT of key whose Content-Type header must equal
// contentType. No cache-control is signed: an admin re-upload replaces the same
// key, so the object must not be marked immutable.
func (s *R2Store) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, time.Time, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", time.Time{}, ungerr.Wrapf(err, "presigning put of %q", key)
	}
	return req.URL, time.Now().Add(ttl), nil
}

func (s *R2Store) ReadHead(ctx context.Context, key string, n int) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Range:  aws.String(fmt.Sprintf("bytes=0-%d", n-1)),
	})
	if err != nil {
		if apiErr, ok := errors.AsType[smithy.APIError](err); ok && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
			return nil, corestore.ErrNotFound
		}
		return nil, ungerr.Wrapf(err, "reading head of object %q", key)
	}
	defer func() {
		if err := out.Body.Close(); err != nil {
			logger.Errorf("closing object %q: %v", key, err)
		}
	}()
	head, err := io.ReadAll(io.LimitReader(out.Body, int64(n)))
	if err != nil {
		return nil, ungerr.Wrapf(err, "reading head of object %q", key)
	}
	return head, nil
}

func (s *R2Store) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		if apiErr, ok := errors.AsType[smithy.APIError](err); ok && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
			return 0, corestore.ErrNotFound
		}
		return 0, ungerr.Wrapf(err, "reading size of object %q", key)
	}
	return aws.ToInt64(out.ContentLength), nil
}

func (s *R2Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return ungerr.Wrapf(err, "deleting object %q", key)
	}
	return nil
}

func (s *R2Store) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, ungerr.Wrapf(err, "getting object %q", key)
	}
	defer func() {
		if err := out.Body.Close(); err != nil {
			logger.Errorf("closing object %q: %v", key, err)
		}
	}()
	body, err := io.ReadAll(io.LimitReader(out.Body, maxGetBytes+1))
	if err != nil {
		return nil, ungerr.Wrapf(err, "reading object %q", key)
	}
	if len(body) > maxGetBytes {
		return nil, ungerr.Unknownf("object %q is larger than %d bytes", key, maxGetBytes)
	}
	return body, nil
}
