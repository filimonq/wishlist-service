package app

import (
	"errors"
	"strings"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantError string
	}{
		{name: "defaults"},
		{name: "overrides", env: map[string]string{"HTTP_PORT": "9090", "MIGRATIONS_PATH": "/app/sql"}},
		{name: "missing database", env: map[string]string{"DB_CONN": ""}, wantError: "DB_CONN"},
		{name: "blank database", env: map[string]string{"DB_CONN": " \t"}, wantError: "DB_CONN"},
		{name: "missing secret", env: map[string]string{"JWT_SECRET": ""}, wantError: "JWT_SECRET"},
		{name: "blank secret", env: map[string]string{"JWT_SECRET": " \t"}, wantError: "JWT_SECRET"},
		{name: "invalid address", env: map[string]string{"HTTP_PORT": "8080:8080"}, wantError: "HTTP_PORT"},
		{name: "invalid port", env: map[string]string{"HTTP_PORT": "abc"}, wantError: "HTTP_PORT"},
		{name: "zero port", env: map[string]string{"HTTP_PORT": "0"}, wantError: "HTTP_PORT"},
		{name: "port out of range", env: map[string]string{"HTTP_PORT": "65536"}, wantError: "HTTP_PORT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const database = "postgres://test:test-password@localhost:5432/test"
			const secret = "test-secret"
			t.Setenv("DB_CONN", database)
			t.Setenv("JWT_SECRET", secret)
			t.Setenv("HTTP_PORT", "")
			t.Setenv("MIGRATIONS_PATH", "")
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			cfg, err := NewConfig()
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("expected error mentioning %s, got %v", tt.wantError, err)
				}
				if strings.Contains(err.Error(), database) || strings.Contains(err.Error(), secret) {
					t.Fatal("configuration error exposes credentials")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantAddr, wantPath := ":8080", "migrations"
			if tt.name == "overrides" {
				wantAddr, wantPath = ":9090", "/app/sql"
			}
			if cfg.HTTPAddr != wantAddr || cfg.MigrationsPath != wantPath || cfg.DBConn != database || cfg.JWTSecret != secret {
				t.Fatal("configuration did not use environment values or defaults")
			}
		})
	}
}

func TestRedactError(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		DBConn:    "postgres://test:password%40example@localhost/test",
		JWTSecret: "test-jwt-secret",
	}
	for _, message := range []string{
		"cannot parse " + cfg.DBConn,
		"connection failed with password@example",
		"connection failed with test:password%40example",
		"invalid secret: " + cfg.JWTSecret,
	} {
		redacted := cfg.RedactError(errors.New(message))
		if strings.Contains(redacted, "password") || strings.Contains(redacted, cfg.JWTSecret) {
			t.Fatal("initialization error exposes credentials")
		}
		if !strings.Contains(redacted, "[redacted]") {
			t.Fatal("expected a redacted error")
		}
	}
	const malformedConnection = "postgres://test:private-password%xx@localhost/test"
	cfg.DBConn = malformedConnection
	if strings.Contains(cfg.RedactError(errors.New("cannot parse "+malformedConnection)), "private-password") {
		t.Fatal("malformed connection string exposes credentials")
	}
}
