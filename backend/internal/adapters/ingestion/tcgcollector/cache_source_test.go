package tcgcollector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCacheSource_FetchesOnceThenReadsFromDisk(t *testing.T) {
	dir := t.TempDir()
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return([]byte("<html>sets</html>"), nil).Once()

	cache := NewCacheSource(dir, pages)
	first, err := cache.Fetch(context.Background(), setsPageURL)
	require.NoError(t, err)
	second, err := cache.Fetch(context.Background(), setsPageURL)
	require.NoError(t, err)

	assert.Equal(t, "<html>sets</html>", string(first))
	assert.Equal(t, first, second, "the second fetch is served from disk, so the mock's single call is enough")
}

func TestCacheSource_RefreshRefetchesAndOverwrites(t *testing.T) {
	dir := t.TempDir()
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return([]byte("old"), nil).Once()
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return([]byte("new"), nil).Once()

	cache := NewCacheSource(dir, pages)
	_, err := cache.Fetch(context.Background(), setsPageURL)
	require.NoError(t, err)

	refreshed, err := cache.Fetch(withRefresh(context.Background()), setsPageURL)
	require.NoError(t, err)

	assert.Equal(t, "new", string(refreshed))
	onDisk, err := os.ReadFile(filepath.Join(dir, "sets-id.html"))
	require.NoError(t, err)
	assert.Equal(t, "new", string(onDisk))
}

func TestCacheSource_FilesWorkAsASavedPagesDirectory(t *testing.T) {
	dir := t.TempDir()
	pages := mocks.NewMockPageSource(t)
	pages.EXPECT().Fetch(mock.Anything, setsPageURL).Return([]byte("<html>sets</html>"), nil).Once()

	_, err := NewCacheSource(dir, pages).Fetch(context.Background(), setsPageURL)
	require.NoError(t, err)

	saved, err := NewDirSource(dir).Fetch(context.Background(), setsPageURL)
	require.NoError(t, err)
	assert.Equal(t, "<html>sets</html>", string(saved))
}
