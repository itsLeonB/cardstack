package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newSet(code string) entity.ExpansionSet {
	id := uuid.New()
	return entity.ExpansionSet{BaseEntity: crud.BaseEntity{ID: id}, Code: code, ImageKey: "expansion-sets/" + id.String()}
}

func keyFor(set entity.ExpansionSet, content string) string {
	sum := sha256.Sum256([]byte(content))
	return "expansion-sets/" + set.ID.String() + "." + hex.EncodeToString(sum[:4]) + ".webp"
}

func fakeResize(_ context.Context, original []byte) ([]byte, error) {
	return append([]byte("small:"), original...), nil
}

func TestCoverKey(t *testing.T) {
	sum := sha256.Sum256([]byte("abc"))
	assert.Equal(t, "expansion-sets/id1."+hex.EncodeToString(sum[:4])+".webp", coverKey("id1", []byte("abc")))
	assert.NotEqual(t, coverKey("id1", []byte("x")), coverKey("id1", []byte("y")), "a changed size must get a new key")
}

func TestPresizeCovers_ResizesUploadsAndSavesKey(t *testing.T) {
	set := newSet("MA6")
	key := keyFor(set, "small:original")

	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "").Return([]entity.ExpansionSet{set}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Get(mock.Anything, set.ImageKey).Return([]byte("original"), nil)
	store.EXPECT().Put(mock.Anything, key, "image/webp", []byte("small:original")).Return(nil)
	updated := set
	updated.CoverKey = key
	sets.EXPECT().Update(mock.Anything, updated).Return(updated, nil)

	sum, err := presizeCovers(context.Background(), sets, store, fakeResize, "")
	require.NoError(t, err)
	assert.Equal(t, summary{Selected: 1, Resized: 1}, sum)
}

func TestPresizeCovers_PassesSetFilter(t *testing.T) {
	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "MA6").Return(nil, nil)

	sum, err := presizeCovers(context.Background(), sets, mocks.NewMockObjectStore(t), fakeResize, "MA6")
	require.NoError(t, err)
	assert.Equal(t, summary{}, sum)
}

func TestPresizeCovers_NothingMissingIsNoOp(t *testing.T) {
	// A second run finds no row without a cover: no Get, Put or Update (the
	// strict mocks fail the test on any of them).
	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "").Return(nil, nil)

	sum, err := presizeCovers(context.Background(), sets, mocks.NewMockObjectStore(t), fakeResize, "")
	require.NoError(t, err)
	assert.Equal(t, 0, sum.Resized)
}

func TestPresizeCovers_FailedRowIsSkippedAndTheRestStillRun(t *testing.T) {
	getFails := newSet("A")
	resizeFails := newSet("B")
	putFails := newSet("C")
	good := newSet("D")

	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "").Return([]entity.ExpansionSet{getFails, resizeFails, putFails, good}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Get(mock.Anything, getFails.ImageKey).Return(nil, errors.New("r2 get down"))
	store.EXPECT().Get(mock.Anything, resizeFails.ImageKey).Return([]byte("corrupt"), nil)
	store.EXPECT().Get(mock.Anything, putFails.ImageKey).Return([]byte("p"), nil)
	store.EXPECT().Get(mock.Anything, good.ImageKey).Return([]byte("g"), nil)
	store.EXPECT().Put(mock.Anything, keyFor(putFails, "small:p"), "image/webp", []byte("small:p")).Return(errors.New("r2 put down"))
	goodKey := keyFor(good, "small:g")
	store.EXPECT().Put(mock.Anything, goodKey, "image/webp", []byte("small:g")).Return(nil)
	good.CoverKey = goodKey
	sets.EXPECT().Update(mock.Anything, good).Return(good, nil)

	resize := func(ctx context.Context, original []byte) ([]byte, error) {
		if string(original) == "corrupt" {
			return nil, errors.New("vips failed")
		}
		return fakeResize(ctx, original)
	}

	sum, err := presizeCovers(context.Background(), sets, store, resize, "")
	require.Error(t, err)
	for _, want := range []string{"set A", "r2 get down", "set B", "vips failed", "set C", "r2 put down"} {
		assert.Contains(t, err.Error(), want)
	}
	assert.Equal(t, summary{Selected: 4, Resized: 1, Failed: 3}, sum)
}

func TestPresizeCovers_SaveFailureLeavesRowForNextRun(t *testing.T) {
	set := newSet("A")
	key := keyFor(set, "small:o")

	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "").Return([]entity.ExpansionSet{set}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Get(mock.Anything, set.ImageKey).Return([]byte("o"), nil)
	store.EXPECT().Put(mock.Anything, key, "image/webp", []byte("small:o")).Return(nil)
	set.CoverKey = key
	sets.EXPECT().Update(mock.Anything, set).Return(entity.ExpansionSet{}, errors.New("db down"))

	sum, err := presizeCovers(context.Background(), sets, store, fakeResize, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
	assert.Equal(t, summary{Selected: 1, Failed: 1}, sum)
}

func TestPresizeCovers_ListFailureFails(t *testing.T) {
	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(mock.Anything, "").Return(nil, errors.New("db down"))

	_, err := presizeCovers(context.Background(), sets, mocks.NewMockObjectStore(t), fakeResize, "")
	require.Error(t, err)
}
