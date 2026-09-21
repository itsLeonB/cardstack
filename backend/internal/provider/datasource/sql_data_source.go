package datasource

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	ezgorm "github.com/itsLeonB/ezutil/v2/gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProvideAndConfigureSQL opens and configures the GORM/SQL connection pool.
// Unlike cashus's version, this has no sync.Once singleton guard: wire
// already guarantees InitializeProviders (and therefore this function) runs
// exactly once, so the guard would just be solving a problem wire doesn't
// have here.
func ProvideAndConfigureSQL(cfg config.DB) (*gorm.DB, *sql.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn(cfg)), &gorm.Config{
		Logger: ezgorm.NewGormLogger(logger.Global),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error opening gorm connection: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("error obtaining sql.DB instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("error pinging SQL DB: %w", err)
	}

	return gormDB, sqlDB, nil
}

func dsn(cfg config.DB) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/" + cfg.Name,
	}
	return u.String()
}
