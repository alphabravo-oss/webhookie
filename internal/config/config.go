package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Addr          string
	DataDir       string
	DBPath        string
	MaxBodyBytes  int64
	RetentionDays int
	MaxEvents     int
	Password      string
	PublicBaseURL string
	Version       string
}

func FromEnv() Config {
	c := Config{
		Addr:          env("WEBHOOKIE_ADDR", ":8080"),
		DataDir:       env("WEBHOOKIE_DATA_DIR", "./data"),
		MaxBodyBytes:  envInt64("WEBHOOKIE_MAX_BODY_BYTES", 1<<20),
		RetentionDays: envInt("WEBHOOKIE_RETENTION_DAYS", 7),
		MaxEvents:     envInt("WEBHOOKIE_MAX_EVENTS", 10000),
		Password:      os.Getenv("WEBHOOKIE_PASSWORD"),
		PublicBaseURL: env("WEBHOOKIE_PUBLIC_BASE_URL", "http://localhost:8080"),
		Version:       env("WEBHOOKIE_VERSION", "0.1.0"),
	}
	c.DBPath = filepath.Join(c.DataDir, "webhookie.db")
	return c
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
