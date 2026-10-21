package config

import (
	"errors"
)

// ScanPhotoR2 configures the private R2 bucket that keeps the photos of scan
// corrections (docs/adr/0022). Its fields are R2's, under SCAN_PHOTO_R2_*. It
// must not be the public image bucket: a photo of a person's binder is never
// served. Unset, the upload call refuses and nothing else changes.
type ScanPhotoR2 R2

func (ScanPhotoR2) Prefix() string { return "SCAN_PHOTO_R2" }

// ValidateScanPhotoR2 refuses a partly set photo bucket, which would fail only
// on the first upload, and a photo bucket that is the public image bucket. With
// none of its settings set there is nothing to check.
func (c Config) ValidateScanPhotoR2() error {
	if R2(c.ScanPhotoR2) == (R2{}) {
		return nil
	}
	if !R2(c.ScanPhotoR2).Configured() {
		return errors.New("SCAN_PHOTO_R2_ACCOUNT_ID, SCAN_PHOTO_R2_ACCESS_KEY_ID, SCAN_PHOTO_R2_SECRET_ACCESS_KEY and SCAN_PHOTO_R2_BUCKET must all be set, or none of them")
	}
	if c.ScanPhotoR2.Bucket == c.R2.Bucket {
		return errors.New("SCAN_PHOTO_R2_BUCKET is the public image bucket R2_BUCKET: scan photos must go to a private bucket (docs/adr/0022)")
	}
	return nil
}
