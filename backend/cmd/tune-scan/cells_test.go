package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedCells embeds Pikachu on axis 0 and Bulbasaur on axis 1 under model, and
// returns the vectors of the two-cell photos used with them: page-a's first cell
// is Pikachu's art, its second is 0.6 Pikachu and 0.8 Bulbasaur (so Bulbasaur is
// the top card at 0.800), and page-b is the reverse.
func seedCells(t *testing.T, db *gorm.DB, f fixture, model string) map[string][]float32 {
	t.Helper()
	pikachu := f.card(t, "Pikachu", "001")
	bulbasaur := f.card(t, "Bulbasaur", "002")
	embed(t, db, pikachu.ID, model, unitVector(0))
	embed(t, db, bulbasaur.ID, model, unitVector(1))

	return map[string][]float32{
		"page-a/r01c01": unitVector(0),
		"page-a/r01c02": mixVector(map[int]float64{0: 0.6, 1: 0.8}),
		"page-b/r01c01": unitVector(1),
		"page-b/r01c02": unitVector(0),
	}
}

func TestCells_PrintsTheTopSimilarityOfEachCellAndSummarisesEachPhoto(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	h := newHarness(t, db, model, seedCells(t, db, f, model))
	photos := dirWithPhotos(t, "page-b.jpg", "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x2", "--threshold", "0.85")

	require.Equal(t, 0, code, h.stderr.String())
	assert.Equal(t, []string{
		"model " + model + ", 1x2 grid, threshold 0.850",
		"page-a",
		"row 1 col 1 1.000 Pikachu (" + f.set.Code + ")",
		"row 1 col 2 0.800 Bulbasaur (" + f.set.Code + ")",
		"lowest 0.800, highest 1.000, 1 of 2 below 0.850",
		"page-b",
		"row 1 col 1 1.000 Bulbasaur (" + f.set.Code + ")",
		"row 1 col 2 1.000 Pikachu (" + f.set.Code + ")",
		"lowest 1.000, highest 1.000, 0 of 2 below 0.850",
	}, lines(h.stdout.String()), "photos come out in name order, cells in grid order")
	h.embedder.AssertNumberOfCalls(t, "Embed", 4) // one embedding per cell
}

func TestCells_WithoutThresholdPrintsOnlyTheLowestAndHighest(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	h := newHarness(t, db, model, seedCells(t, db, f, model))
	photos := dirWithPhotos(t, "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x2")

	require.Equal(t, 0, code, h.stderr.String())
	assert.Equal(t, []string{
		"model " + model + ", 1x2 grid",
		"page-a",
		"row 1 col 1 1.000 Pikachu (" + f.set.Code + ")",
		"row 1 col 2 0.800 Bulbasaur (" + f.set.Code + ")",
		"lowest 0.800, highest 1.000",
	}, lines(h.stdout.String()))
}

func TestCells_ACellWithNoPublishedCardIsShownButLeftOutOfTheSummary(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	embed(t, db, f.card(t, "Pikachu", "001").ID, model, unitVector(0))
	require.NoError(t, db.Model(&entity.ExpansionSet{}).Where("id = ?", f.set.ID).Update("published", false).Error)
	h := newHarness(t, db, model, map[string][]float32{"page-a/r01c01": unitVector(0)})
	photos := dirWithPhotos(t, "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x1", "--threshold", "0.85")

	require.Equal(t, 0, code, h.stderr.String())
	assert.Equal(t, []string{
		"model " + model + ", 1x1 grid, threshold 0.850",
		"page-a",
		"row 1 col 1 - no published Card to compare",
		"no cell had a published Card to compare",
	}, lines(h.stdout.String()))
}

func TestCells_NoEmbeddingsForTheModelFailsBeforeAnyCellIsEmbedded(t *testing.T) {
	db := testDB(t)
	model := "tune-" + uuid.NewString()
	h := newHarness(t, db, model, nil)
	photos := dirWithPhotos(t, "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x1")

	assert.Equal(t, 1, code)
	assert.Contains(t, h.stderr.String(), `the catalog has no embeddings for model "`+model+`"`)
	assert.Equal(t, 0, h.cuts, "no photo is cut")
	h.embedder.AssertNumberOfCalls(t, "Embed", 0) // no Gemini call is made
	assert.Empty(t, h.stdout.String())
}

func TestCells_MissingBunFailsBeforeTheDatabaseIsOpened(t *testing.T) {
	db := testDB(t)
	h := newHarness(t, db, "tune-"+uuid.NewString(), nil)
	h.env.lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	photos := dirWithPhotos(t, "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x1")

	assert.Equal(t, 1, code)
	assert.Contains(t, h.stderr.String(), "bun is needed to cut the Page photos")
	assert.Equal(t, 0, h.opens)
}

func TestCells_MissingCutterScriptFailsBeforeTheDatabaseIsOpened(t *testing.T) {
	db := testDB(t)
	h := newHarness(t, db, "tune-"+uuid.NewString(), nil)
	h.env.frontendDir = t.TempDir()
	photos := dirWithPhotos(t, "page-a.jpg")

	code := h.run("cells", "--photos", photos, "--grid", "1x1")

	assert.Equal(t, 1, code)
	assert.Contains(t, h.stderr.String(), filepath.Join("tools", "cut-pages.ts"))
	assert.Equal(t, 0, h.opens)
}

func TestCells_BadFlagsAreUsageErrors(t *testing.T) {
	db := testDB(t)
	photos := dirWithPhotos(t, "page-a.jpg")
	cases := map[string][]string{
		"no photos folder":    {"cells", "--grid", "1x1"},
		"no grid":             {"cells", "--photos", photos},
		"zero rows":           {"cells", "--photos", photos, "--grid", "0x3"},
		"threshold above one": {"cells", "--photos", photos, "--grid", "1x1", "--threshold", "1.5"},
		"stray argument":      {"cells", "--photos", photos, "--grid", "1x1", "extra"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, db, "tune-"+uuid.NewString(), nil)

			code := h.run(args...)

			assert.Equal(t, 2, code, h.stderr.String())
			assert.Equal(t, 0, h.opens, "a bad flag opens nothing")
		})
	}
}

func TestCells_APhotosFolderThatIsNotThereFailsBeforeTheDatabaseIsOpened(t *testing.T) {
	db := testDB(t)
	h := newHarness(t, db, "tune-"+uuid.NewString(), nil)
	missing := filepath.Join(t.TempDir(), "nowhere")

	code := h.run("cells", "--photos", missing, "--grid", "1x1")

	assert.Equal(t, 1, code)
	assert.Contains(t, h.stderr.String(), "photos folder not found: "+missing)
	assert.Equal(t, 0, h.opens)
}

func TestCells_WritesNothing(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	h := newHarness(t, db, model, seedCells(t, db, f, model))
	photos := dirWithPhotos(t, "page-a.jpg", "page-b.jpg")
	sets, cards, embeddings := writeCounts(t, db, f, model)

	require.Equal(t, 0, h.run("cells", "--photos", photos, "--grid", "1x2", "--threshold", "0.85"), h.stderr.String())

	gotSets, gotCards, gotEmbeddings := writeCounts(t, db, f, model)
	assert.Equal(t, sets, gotSets)
	assert.Equal(t, cards, gotCards)
	assert.Equal(t, embeddings, gotEmbeddings)
}

func TestCells_RemovesItsTemporaryCutFolderUnlessAskedToKeepIt(t *testing.T) {
	db := testDB(t)
	f := newFixture(t, db)
	model := "tune-" + uuid.NewString()
	photos := dirWithPhotos(t, "page-a.jpg")

	h := newHarness(t, db, model, seedCells(t, db, f, model))
	require.Equal(t, 0, h.run("cells", "--photos", photos, "--grid", "1x2"), h.stderr.String())
	require.Len(t, h.cutDirs, 1)
	_, err := os.Stat(h.cutDirs[0])
	assert.ErrorIs(t, err, os.ErrNotExist, "the temporary folder is removed")

	// A second run needs its own set: local ids are unique within one.
	again := newFixture(t, db)
	model = "tune-" + uuid.NewString()
	keep := filepath.Join(t.TempDir(), "cut")
	h = newHarness(t, db, model, seedCells(t, db, again, model))
	require.Equal(t, 0, h.run("cells", "--photos", photos, "--grid", "1x2", "--cut-dir", keep), h.stderr.String())
	_, err = os.Stat(filepath.Join(keep, "page-a", "r01c02.jpg"))
	assert.NoError(t, err, "--cut-dir keeps the cut cells")
}
