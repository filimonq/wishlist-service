package app

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr       string
	DBConn         string
	JWTSecret      string
	MigrationsPath string
}

func NewConfig() (*Config, error) {
	cfg, err := NewMigrationConfig()
	if err != nil {
		return nil, err
	}
	port := getEnv("HTTP_PORT", "8080")
	cfg.HTTPAddr = ":" + port
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, errors.New("HTTP_PORT must be an integer between 1 and 65535")
	}
	return cfg, nil
}

func NewMigrationConfig() (*Config, error) {
	cfg := &Config{
		DBConn:         os.Getenv("DB_CONN"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),
	}
	if strings.TrimSpace(cfg.DBConn) == "" {
		return nil, errors.New("DB_CONN is required")
	}
	return cfg, nil
}

func (c *Config) RedactError(err error) string {
	message := err.Error()
	values := []string{c.DBConn, c.JWTSecret}
	if databaseURL, parseErr := url.Parse(c.DBConn); parseErr == nil && databaseURL.User != nil {
		if password, ok := databaseURL.User.Password(); ok {
			values = append(values, password, url.QueryEscape(password), databaseURL.User.String())
		}
	}
	for _, value := range values {
		if value != "" {
			message = strings.ReplaceAll(message, value, "[redacted]")
		}
	}
	return message
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
