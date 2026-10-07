package routes

import (
	"cmp"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/db/postgres/migrations"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/adapters/http/ratelimit"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// testImageBase is the configured image base address (IMAGE_BASE_URL) the
// feature tests serve hosted keys under.
const testImageBase = "https://img.example.test"

const (
	testIssuer = "https://cardstack.clerk.accounts.dev"
	testOrigin = "https://cardstack.example"
	testKeyID  = "ins_test"
)

// testAPI is the whole API over a real local Postgres (docs/adr/0005: feature
// tests run against the full real stack). Tokens are real RS256 JWTs checked by
// the real ClerkVerifier; only the fetch of Clerk's published keys is faked, so
// the tests need no network access to Clerk. See
// internal/domain/repository/repository_test_helper_test.go for how to start a
// matching Postgres 18 locally.
type testAPI struct {
	humatest.TestAPI
	db  *gorm.DB
	key *rsa.PrivateKey
}

// newTestAPI deliberately truncates no tables: this database is shared with
// other packages' tests, which `go test ./...` runs concurrently in separate
// processes. Each test uses uuid-suffixed subjects and emails instead. Its
// per-user limits are high enough that no other test meets them.
func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	unreachable := config.Tier{PerMinute: 1_000_000, Burst: 1_000_000}
	return newTestAPIWithLimits(t, ratelimit.NewLimits(config.RateLimit{User: unreachable, Search: unreachable, Facets: unreachable}, time.Now))
}

func newTestAPIWithLimits(t *testing.T, limits ratelimit.Limits) testAPI {
	t.Helper()

	db, sqlDB := openTestDB(t)
	migrateTestDB(t, sqlDB)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keys := mocks.NewMockKeySource(t)
	keys.EXPECT().FindKey(mock.Anything, testKeyID).
		Return(&clerk.JSONWebKey{Key: &key.PublicKey, KeyID: testKeyID, Algorithm: "RS256", Use: "sig"}, nil).Maybe()

	ds := &provider.DataSources{Gorm: db, SQL: sqlDB}
	images := mapper.NewImageHost(testImageBase)
	services := provider.ProvideServices(
		authpkg.NewClerkVerifier(testIssuer, []string{testOrigin}, keys),
		provider.ProvideUserService(ds, provider.ProvideUserRepository(ds), provider.ProvideIdentityCache()),
		provider.ProvideCatalogService(ds, images),
		provider.ProvideCollectionService(ds),
		provider.ProvideInventoryService(ds, images),
		provider.ProvideMatchService(ds, images),
	)

	_, api := humatest.New(t, httpapi.NewConfig())
	RegisterRoutes(api, services, limits)

	return testAPI{TestAPI: api, db: db, key: key}
}

func openTestDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()

	dsn := "host=" + envOr("DB_HOST", "localhost") +
		" port=" + envOr("DB_PORT", "5432") +
		" user=" + envOr("DB_USER", "cardstack") +
		" password=" + envOr("DB_PASSWORD", "cardstack") +
		" dbname=" + envOr("DB_NAME", "cardstack") +
		" sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err, "connecting to test postgres")
	sqlDB, err := db.DB()
	require.NoError(t, err)

	return db, sqlDB
}

func migrateTestDB(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	goose.SetBaseFS(migrations.Migrations)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, "."))
}

func envOr(key, fallback string) string {
	return cmp.Or(os.Getenv(key), fallback)
}

// validClaims is a session token Clerk would mint for a brand-new user of the
// test instance and origin.
func validClaims() map[string]any {
	now := time.Now()
	id := uuid.NewString()
	return map[string]any{
		"iss":   testIssuer,
		"sub":   "user_" + id,
		"azp":   testOrigin,
		"iat":   now.Add(-time.Minute).Unix(),
		"nbf":   now.Add(-time.Minute).Unix(),
		"exp":   now.Add(time.Minute).Unix(),
		"email": id + "@example.com",
		"name":  "Test User",
	}
}

func (a testAPI) sign(t *testing.T, claims map[string]any) string {
	t.Helper()

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: a.key, KeyID: testKeyID}},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", testKeyID),
	)
	require.NoError(t, err)
	token, err := josejwt.Signed(signer).Claims(claims).CompactSerialize()
	require.NoError(t, err)

	return token
}

// newUserToken returns a valid token for a user who has never called the API.
func (a testAPI) newUserToken(t *testing.T) string {
	t.Helper()
	return a.sign(t, validClaims())
}

// bearer is the Authorization header carrying token.
func bearer(token string) string {
	return "Authorization: Bearer " + strings.TrimSpace(token)
}
