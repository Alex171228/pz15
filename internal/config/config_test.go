package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure clean env for this test.
	os.Clearenv()

	cfg := Load()
	require.Equal(t, "", cfg.DatabaseURL)
	require.Equal(t, 20, cfg.MaxOpenConns)
	require.Equal(t, 10, cfg.MaxIdleConns)
	require.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	require.Equal(t, 5*time.Minute, cfg.ConnMaxIdleTime)
	require.Equal(t, ":8080", cfg.HTTPAddr)
}

func TestLoad_OverridesAndInvalidValues(t *testing.T) {
	t.Cleanup(os.Clearenv)

	t.Run("valid overrides", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
		os.Setenv("DB_MAX_OPEN", "5")
		os.Setenv("DB_MAX_IDLE", "2")
		os.Setenv("DB_CONN_MAX_LIFETIME", "1m")
		os.Setenv("DB_CONN_MAX_IDLE_TIME", "10s")
		os.Setenv("HTTP_ADDR", ":9999")

		cfg := Load()
		require.Equal(t, "postgres://u:p@localhost:5432/db?sslmode=disable", cfg.DatabaseURL)
		require.Equal(t, 5, cfg.MaxOpenConns)
		require.Equal(t, 2, cfg.MaxIdleConns)
		require.Equal(t, time.Minute, cfg.ConnMaxLifetime)
		require.Equal(t, 10*time.Second, cfg.ConnMaxIdleTime)
		require.Equal(t, ":9999", cfg.HTTPAddr)
	})

	t.Run("invalid numbers fall back to defaults", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DB_MAX_OPEN", "abc")
		os.Setenv("DB_MAX_IDLE", "xyz")
		os.Setenv("DB_CONN_MAX_LIFETIME", "bad")
		os.Setenv("DB_CONN_MAX_IDLE_TIME", "bad")

		cfg := Load()
		require.Equal(t, 20, cfg.MaxOpenConns)
		require.Equal(t, 10, cfg.MaxIdleConns)
		require.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
		require.Equal(t, 5*time.Minute, cfg.ConnMaxIdleTime)
	})
}
