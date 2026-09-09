// Package config loads backend configuration from the environment.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	MongoURI string
	MongoDB  string
	Port     string
}

// Load reads configuration from backend/.env (if present) and the process
// environment. It exits the process if a required value is missing.
func Load() Config {
	// .env is optional: in deployed environments the vars are set directly.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("config: could not read .env: %v", err)
	}

	cfg := Config{
		MongoURI: os.Getenv("MONGODB_URI"),
		MongoDB:  getenvDefault("MONGO_DB", "commitin"),
		Port:     getenvDefault("PORT", "8080"),
	}

	if cfg.MongoURI == "" {
		log.Fatal("config: MONGODB_URI is required (see backend/.env.example)")
	}

	return cfg
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
