package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	HTTPAddr string
}

func Load() Config {
	return Config{
		DatabaseURL:     getenv("DATABASE_URL", ""),
		MaxOpenConns:    getenvInt("DB_MAX_OPEN", 20),
		MaxIdleConns:    getenvInt("DB_MAX_IDLE", 10),
		ConnMaxLifetime: getenvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		ConnMaxIdleTime: getenvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
	}
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
