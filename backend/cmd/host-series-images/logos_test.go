package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func writeLogo(t *testing.T, dir, code, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, code+".webp"), []byte(content), 0o600))
	sum := sha256.Sum256([]byte(content))
	return "series/" + code + "." + hex.EncodeToString(sum[:4]) + ".webp"
}

func seriesSpec(code string) crud.Specification[entity.Series] {
	return crud.Specification[entity.Series]{Model: entity.Series{Code: code}}
}

func TestLogoKey(t *testing.T) {
	sum := sha256.Sum256([]byte("abc"))
	assert.Equal(t, "series/scarlet-violet."+hex.EncodeToString(sum[:4])+".webp", logoKey("scarlet-violet", []byte("abc")))
	assert.NotEqual(t, logoKey("a", []byte("x")), logoKey("a", []byte("y")), "a changed file must get a new key")
}

func TestHostLogos_UploadsAndSavesKey(t *testing.T) {
	dir := t.TempDir()
	key := writeLogo(t, dir, "evolusi-mega", "bytes")
	row := entity.Series{BaseEntity: crud.BaseEntity{ID: uuid.New()}, Code: "evolusi-mega"}

	rows := mocks.NewMockRepository[entity.Series](t)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("evolusi-mega")).Return([]entity.Series{row}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Put(mock.Anything, key, "image/webp", []byte("bytes")).Return(nil)
	updated := row
	updated.ImageKey = key
	rows.EXPECT().Update(mock.Anything, updated).Return(updated, nil)

	sum, err := hostLogos(context.Background(), dir, rows, store)
	require.NoError(t, err)
	assert.Equal(t, 1, sum.Uploaded)
	assert.Equal(t, 1, sum.Matched)
}

func TestHostLogos_UnchangedFileIsNoOp(t *testing.T) {
	dir := t.TempDir()
	key := writeLogo(t, dir, "evolusi-mega", "bytes")
	row := entity.Series{Code: "evolusi-mega", ImageKey: key}

	rows := mocks.NewMockRepository[entity.Series](t)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("evolusi-mega")).Return([]entity.Series{row}, nil)
	store := mocks.NewMockObjectStore(t) // any Put or Update would fail the test

	sum, err := hostLogos(context.Background(), dir, rows, store)
	require.NoError(t, err)
	assert.Equal(t, 0, sum.Uploaded)
	assert.Equal(t, 1, sum.Matched, "an unchanged row still counts as matched")
}

func TestHostLogos_ChangedFileGetsNewKey(t *testing.T) {
	dir := t.TempDir()
	key := writeLogo(t, dir, "evolusi-mega", "new bytes")
	row := entity.Series{Code: "evolusi-mega", ImageKey: "series/evolusi-mega.00000000.webp"}

	rows := mocks.NewMockRepository[entity.Series](t)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("evolusi-mega")).Return([]entity.Series{row}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Put(mock.Anything, key, "image/webp", []byte("new bytes")).Return(nil)
	updated := row
	updated.ImageKey = key
	rows.EXPECT().Update(mock.Anything, updated).Return(updated, nil)

	sum, err := hostLogos(context.Background(), dir, rows, store)
	require.NoError(t, err)
	assert.Equal(t, 1, sum.Uploaded)
}

func TestHostLogos_NoMatchingRowFailsWithoutUploading(t *testing.T) {
	dir := t.TempDir()
	writeLogo(t, dir, "typo-code", "bytes")

	rows := mocks.NewMockRepository[entity.Series](t)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("typo-code")).Return(nil, nil)
	store := mocks.NewMockObjectStore(t)

	sum, err := hostLogos(context.Background(), dir, rows, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "typo-code")
	assert.Equal(t, 0, sum.Matched)
}

func TestHostLogos_UploadFailureKeepsGoingAndFails(t *testing.T) {
	dir := t.TempDir()
	badKey := writeLogo(t, dir, "a-bad", "bad")
	goodKey := writeLogo(t, dir, "b-good", "good")
	bad := entity.Series{Code: "a-bad"}
	good := entity.Series{Code: "b-good"}

	rows := mocks.NewMockRepository[entity.Series](t)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("a-bad")).Return([]entity.Series{bad}, nil)
	rows.EXPECT().FindAll(mock.Anything, seriesSpec("b-good")).Return([]entity.Series{good}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Put(mock.Anything, badKey, "image/webp", []byte("bad")).Return(errors.New("r2 down"))
	store.EXPECT().Put(mock.Anything, goodKey, "image/webp", []byte("good")).Return(nil)
	good.ImageKey = goodKey
	rows.EXPECT().Update(mock.Anything, good).Return(good, nil)

	sum, err := hostLogos(context.Background(), dir, rows, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "r2 down")
	assert.Equal(t, 1, sum.Uploaded, "the other logo is still hosted")
}

func TestHostLogos_NoFilesIsAnError(t *testing.T) {
	_, err := hostLogos(context.Background(), t.TempDir(), mocks.NewMockRepository[entity.Series](t), mocks.NewMockObjectStore(t))
	require.Error(t, err)
}
