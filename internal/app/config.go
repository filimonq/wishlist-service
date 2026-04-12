package app

import "os"

type Config struct {
	HTTPAddr       string
	DBConn         string
	JWTSecret      string
	MigrationsPath string
}

func NewConfig() *Config {
	return &Config{
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		DBConn:         getEnv("DB_CONN", "postgres://postgres:postgres@localhost:5432/wishlist?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
