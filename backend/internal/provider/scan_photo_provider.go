package provider

import (
	objectstoreadapter "github.com/itsLeonB/cardstack/backend/internal/adapters/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
)

// ScanPhotoStore is the private photo bucket's store (docs/adr/0022). It is a
// type of its own so wire can tell it from the public image bucket's
// ObjectStore.
type ScanPhotoStore objectstore.ObjectStore

// ProvideScanPhotoStore builds the R2 store of the private photo bucket. Without
// its SCAN_PHOTO_R2_* settings it is nil, and the upload call is refused.
func ProvideScanPhotoStore() ScanPhotoStore {
	if !config.R2(config.Global.ScanPhotoR2).Configured() {
		return nil
	}
	return ScanPhotoStore(objectstoreadapter.NewR2Store(config.R2(config.Global.ScanPhotoR2)))
}

// ProvideScanPhotoService builds the scan photo upload service over store.
func ProvideScanPhotoService(store ScanPhotoStore) service.ScanPhotoService {
	return service.NewScanPhotoService(store)
}
