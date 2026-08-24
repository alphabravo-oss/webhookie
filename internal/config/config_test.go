package config

import (
	"path/filepath"
	"testing"
)

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("WEBHOOKIE_ADDR", "")
	t.Setenv("WEBHOOKIE_DATA_DIR", "")
	t.Setenv("WEBHOOKIE_MAX_BODY_BYTES", "")
	t.Setenv("WEBHOOKIE_RETENTION_DAYS", "")
	t.Setenv("WEBHOOKIE_MAX_EVENTS", "")
	t.Setenv("WEBHOOKIE_PASSWORD", "")
	t.Setenv("WEBHOOKIE_PUBLIC_BASE_URL", "")
	c := FromEnv()
	if c.Addr != ":8080" {
		t.Fatalf("addr %q", c.Addr)
	}
	if c.DataDir != "./data" {
		t.Fatalf("data dir %q", c.DataDir)
	}
	if c.DBPath != filepath.Join("./data", "webhookie.db") {
		t.Fatalf("db path %q", c.DBPath)
	}
	if c.MaxBodyBytes != 1<<20 {
		t.Fatalf("max body %d", c.MaxBodyBytes)
	}
	if c.RetentionDays != 7 || c.MaxEvents != 10000 {
		t.Fatalf("retention %d max %d", c.RetentionDays, c.MaxEvents)
	}
	if c.PublicBaseURL != "http://localhost:8080" {
		t.Fatalf("base url %q", c.PublicBaseURL)
	}
}

func TestFromEnvOverrides(t *testing.T) {
	t.Setenv("WEBHOOKIE_ADDR", ":9090")
	t.Setenv("WEBHOOKIE_DATA_DIR", "/tmp/wh")
	t.Setenv("WEBHOOKIE_MAX_BODY_BYTES", "2048")
	t.Setenv("WEBHOOKIE_RETENTION_DAYS", "3")
	t.Setenv("WEBHOOKIE_MAX_EVENTS", "50")
	t.Setenv("WEBHOOKIE_PASSWORD", "secret")
	t.Setenv("WEBHOOKIE_PUBLIC_BASE_URL", "http://webhookie:8080")
	c := FromEnv()
	if c.Addr != ":9090" {
		t.Fatalf("addr %q", c.Addr)
	}
	if c.DBPath != filepath.Join("/tmp/wh", "webhookie.db") {
		t.Fatalf("db path %q", c.DBPath)
	}
	if c.MaxBodyBytes != 2048 || c.RetentionDays != 3 || c.MaxEvents != 50 {
		t.Fatalf("%+v", c)
	}
	if c.Password != "secret" {
		t.Fatalf("password %q", c.Password)
	}
	if c.PublicBaseURL != "http://webhookie:8080" {
		t.Fatalf("base %q", c.PublicBaseURL)
	}
}
